package nodes

import (
	"context"
	"log/slog"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"git.bitnp.net/BITNP/guild-go/internal/access"
	"git.bitnp.net/BITNP/guild-go/internal/apperr"
	"git.bitnp.net/BITNP/guild-go/internal/auth"
	"git.bitnp.net/BITNP/guild-go/internal/comments"
	"git.bitnp.net/BITNP/guild-go/internal/storage"
	"git.bitnp.net/BITNP/guild-go/internal/users"
)

// Service implements node use cases.
type Service struct {
	repo   *Repository
	users  *users.Service
	access *access.Checker
	store  storage.Store
}

// NewService creates a node service.
func NewService(repo *Repository, userService *users.Service, checker *access.Checker, store storage.Store) *Service {
	return &Service{repo: repo, users: userService, access: checker, store: store}
}

// List returns the flat tree used by the sidebar.
func (s *Service) List(ctx context.Context, session auth.Session) ([]Summary, error) {
	rows, err := s.repo.List(ctx)
	if err != nil {
		return nil, err
	}
	leaders, members, err := s.access.Relations(ctx, session.Subject)
	if err != nil {
		return nil, err
	}
	result := make([]Summary, 0, len(rows))
	for _, row := range rows {
		result = append(result, Summary{
			ID:              row.ID,
			ParentID:        row.ParentID,
			Title:           row.Title,
			DueAt:           row.DueAt,
			CompletedAt:     row.CompletedAt,
			Tags:            nonNil(row.Tags),
			SortOrder:       row.SortOrder,
			ChildCount:      row.ChildCount,
			CommentCount:    row.CommentCount,
			LatestCommentAt: row.LatestCommentAt,
			MyRelation:      relationFor(row.ID, leaders, members),
			UpdatedAt:       row.UpdatedAt,
		})
	}
	return result, nil
}

// Overview returns roots and the recent-comment feed.
func (s *Service) Overview(ctx context.Context, session auth.Session) (Overview, error) {
	roots, err := s.repo.Children(ctx, nil)
	if err != nil {
		return Overview{}, err
	}
	rootIDs := make([]int64, 0, len(roots))
	for _, root := range roots {
		rootIDs = append(rootIDs, root.ID)
	}
	rootLeaders, err := s.repo.RootLeaders(ctx, rootIDs)
	if err != nil {
		return Overview{}, err
	}
	relatedLeaders, relatedMembers, err := s.access.Relations(ctx, session.Subject)
	if err != nil {
		return Overview{}, err
	}
	rootViews, err := s.childViews(ctx, session, roots, relatedLeaders, relatedMembers)
	if err != nil {
		return Overview{}, err
	}
	rootSummaries := make([]RootSummary, 0, len(rootViews))
	for _, view := range rootViews {
		leaderSubjects := rootLeaders[view.ID]
		leaderMap := s.users.Summaries(ctx, session, leaderSubjects)
		leaders := make([]users.Summary, 0, len(leaderSubjects))
		for _, subject := range leaderSubjects {
			if summary, ok := leaderMap[subject]; ok {
				leaders = append(leaders, summary)
			}
		}
		rootSummaries = append(rootSummaries, RootSummary{
			Summary:       view.Summary,
			Leaders:       leaders,
			LatestComment: view.LatestComment,
		})
	}

	recentRows, err := s.repo.RecentComments(ctx, 50)
	if err != nil {
		return Overview{}, err
	}
	recentViews, err := s.recentViews(ctx, session, recentRows)
	if err != nil {
		return Overview{}, err
	}
	if rootSummaries == nil {
		rootSummaries = []RootSummary{}
	}
	return Overview{Roots: rootSummaries, RecentComments: recentViews}, nil
}

// Get returns the full node view.
func (s *Service) Get(ctx context.Context, session auth.Session, id int64) (Detail, error) {
	node, err := s.repo.Get(ctx, id)
	if err != nil {
		return Detail{}, err
	}
	breadcrumb, err := s.repo.Breadcrumb(ctx, id)
	if err != nil {
		return Detail{}, err
	}
	memberRows, err := s.repo.Members(ctx, id)
	if err != nil {
		return Detail{}, err
	}
	effectiveLeaders, err := s.access.EffectiveLeaders(ctx, id)
	if err != nil {
		return Detail{}, err
	}
	effectiveMembers, err := s.access.EffectiveMembers(ctx, id)
	if err != nil {
		return Detail{}, err
	}
	counts, err := s.access.SubtreeCounts(ctx, id)
	if err != nil {
		return Detail{}, err
	}
	children, err := s.repo.Children(ctx, &id)
	if err != nil {
		return Detail{}, err
	}
	relatedLeaders, relatedMembers, err := s.access.Relations(ctx, session.Subject)
	if err != nil {
		return Detail{}, err
	}
	childViews, err := s.childViews(ctx, session, children, relatedLeaders, relatedMembers)
	if err != nil {
		return Detail{}, err
	}

	directLeaders, directMembers := s.resolveDirectMembers(ctx, session, memberRows)
	directSubjects := make(map[string]struct{}, len(memberRows))
	for _, member := range memberRows {
		directSubjects[member.Subject] = struct{}{}
	}
	effectiveLeaderSet := stringSet(effectiveLeaders)
	effectiveMemberSet := stringSet(effectiveMembers)
	_, isLeader := effectiveLeaderSet[session.Subject]
	_, isMember := effectiveMemberSet[session.Subject]
	structural := isLeader || session.Permissions.SystemAdmin
	canComment := isMember || session.Permissions.SystemAdmin

	inheritedLeaderCount := len(effectiveLeaderSet) - len(intersect(effectiveLeaderSet, directSubjects))
	inheritedMemberCount := len(effectiveMemberSet) - len(intersect(effectiveMemberSet, directSubjects))
	if inheritedLeaderCount < 0 {
		inheritedLeaderCount = 0
	}
	if inheritedMemberCount < 0 {
		inheritedMemberCount = 0
	}

	return Detail{
		Node:                   node,
		CreatedByUser:          s.users.SummaryFor(ctx, session, node.CreatedBy),
		Breadcrumb:             nonNilBreadcrumb(breadcrumb),
		DirectLeaders:          directLeaders,
		DirectMembers:          directMembers,
		InheritedLeaderCount:   inheritedLeaderCount,
		InheritedMemberCount:   inheritedMemberCount,
		EffectiveMemberCount:   len(effectiveMemberSet),
		SubtreeNodeCount:       counts.Nodes,
		SubtreeCommentCount:    counts.Comments,
		SubtreeAttachmentCount: counts.Attachments,
		Permissions: Permissions{
			CanEdit:            structural,
			CanDelete:          structural,
			CanComment:         canComment,
			CanManageMembers:   structural,
			CanAddChild:        structural,
			CanReorderChildren: structural,
		},
		Children: childViews,
	}, nil
}

// Create inserts a root or child node.
func (s *Service) Create(ctx context.Context, session auth.Session, req CreateRequest) (Detail, error) {
	title, description, tags, err := normalizeNodeFields(req.Title, req.Description, req.Tags)
	if err != nil {
		return Detail{}, err
	}
	if req.ParentID == nil {
		if !session.Permissions.Manager {
			return Detail{}, apperr.Forbidden("只有 manager 可以创建根节点")
		}
	} else {
		if _, err := s.repo.Get(ctx, *req.ParentID); err != nil {
			return Detail{}, err
		}
		if !session.Permissions.SystemAdmin {
			allowed, err := s.access.IsEffectiveLeader(ctx, *req.ParentID, session.Subject)
			if err != nil {
				return Detail{}, err
			}
			if !allowed {
				return Detail{}, apperr.Forbidden("只有父节点负责人可以创建子节点")
			}
		}
	}
	leaders, members, err := s.normalizeMembers(ctx, req.ParentID, req.LeaderSubjects, req.MemberSubjects, session.Subject)
	if err != nil {
		return Detail{}, err
	}
	req.Title = title
	req.Description = description
	req.Tags = tags
	id, err := s.repo.Create(ctx, req, session.Subject, leaders, members)
	if err != nil {
		return Detail{}, err
	}
	return s.Get(ctx, session, id)
}

// Update replaces editable fields.
func (s *Service) Update(ctx context.Context, session auth.Session, id int64, req UpdateRequest) (Detail, error) {
	if _, err := s.repo.Get(ctx, id); err != nil {
		return Detail{}, err
	}
	if err := s.requireStructural(ctx, session, id, "只有节点负责人可以编辑节点"); err != nil {
		return Detail{}, err
	}
	title, description, tags, err := normalizeNodeFields(req.Title, req.Description, req.Tags)
	if err != nil {
		return Detail{}, err
	}
	req.Title = title
	req.Description = description
	req.Tags = tags
	if _, err := s.repo.Update(ctx, id, req, session.Subject); err != nil {
		return Detail{}, err
	}
	return s.Get(ctx, session, id)
}

// Delete removes a node and its whole subtree.
func (s *Service) Delete(ctx context.Context, session auth.Session, id int64) error {
	if _, err := s.repo.Get(ctx, id); err != nil {
		return err
	}
	if err := s.requireStructural(ctx, session, id, "只有节点负责人可以删除节点"); err != nil {
		return err
	}
	keys, err := s.access.AttachmentKeysForNode(ctx, id)
	if err != nil {
		return err
	}
	if err := s.repo.Delete(ctx, id); err != nil {
		return err
	}
	for _, key := range keys {
		deleteCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		if err := s.store.Delete(deleteCtx, key); err != nil {
			slog.Warn("delete node attachment failed", "key", key, "error", err)
		}
		cancel()
	}
	return nil
}

// Reorder rewrites sibling order.
func (s *Service) Reorder(ctx context.Context, session auth.Session, parentID *int64, ids []int64) error {
	if parentID == nil {
		if !session.Permissions.Manager {
			return apperr.Forbidden("只有 manager 可以调整根节点顺序")
		}
		return s.repo.Reorder(ctx, nil, ids)
	}
	if _, err := s.repo.Get(ctx, *parentID); err != nil {
		return err
	}
	if err := s.requireStructural(ctx, session, *parentID, "只有父节点负责人可以调整子节点顺序"); err != nil {
		return err
	}
	return s.repo.Reorder(ctx, parentID, ids)
}

// ReplaceMembers replaces direct leaders and members.
func (s *Service) ReplaceMembers(ctx context.Context, session auth.Session, id int64, req MembersRequest) (Detail, error) {
	node, err := s.repo.Get(ctx, id)
	if err != nil {
		return Detail{}, err
	}
	if err := s.requireStructural(ctx, session, id, "只有节点负责人可以管理成员"); err != nil {
		return Detail{}, err
	}
	leaders, members, err := s.normalizeMembers(ctx, node.ParentID, req.Leaders, req.Members, "")
	if err != nil {
		return Detail{}, err
	}
	if len(leaders) == 0 {
		return Detail{}, apperr.BadRequest("节点至少需要一个直接负责人")
	}
	if err := s.repo.ReplaceMembers(ctx, id, leaders, members); err != nil {
		return Detail{}, err
	}
	return s.Get(ctx, session, id)
}

// EffectiveMembers returns inherited members and their source nodes.
func (s *Service) EffectiveMembers(ctx context.Context, session auth.Session, id int64) (EffectiveMembers, error) {
	if _, err := s.repo.Get(ctx, id); err != nil {
		return EffectiveMembers{}, err
	}
	details, err := s.access.EffectiveMemberDetails(ctx, id)
	if err != nil {
		return EffectiveMembers{}, err
	}
	type aggregate struct {
		leader  bool
		direct  bool
		sources map[int64]string
	}
	bySubject := make(map[string]*aggregate)
	for _, detail := range details {
		item, ok := bySubject[detail.Subject]
		if !ok {
			item = &aggregate{sources: make(map[int64]string)}
			bySubject[detail.Subject] = item
		}
		if detail.Role == "leader" {
			item.leader = true
		}
		if detail.NodeID == id {
			item.direct = true
		} else {
			item.sources[detail.NodeID] = detail.NodeTitle
		}
	}
	subjects := make([]string, 0, len(bySubject))
	for subject := range bySubject {
		subjects = append(subjects, subject)
	}
	summaries := s.users.Summaries(ctx, session, subjects)
	leaders := make([]EffectiveMember, 0)
	members := make([]EffectiveMember, 0)
	for subject, item := range bySubject {
		sources := make([]Source, 0, len(item.sources))
		if !item.direct {
			for sourceID, title := range item.sources {
				sources = append(sources, Source{ID: sourceID, Title: title})
			}
			sort.Slice(sources, func(i, j int) bool { return sources[i].ID < sources[j].ID })
		}
		summary, ok := summaries[subject]
		if !ok {
			summary = users.Summary{Subject: subject, UserName: subject, Positions: []users.Position{}}
		}
		entry := EffectiveMember{User: summary, Direct: item.direct, Sources: sources}
		if item.leader {
			leaders = append(leaders, entry)
		} else {
			members = append(members, entry)
		}
	}
	sortMembers(leaders)
	sortMembers(members)
	return EffectiveMembers{Leaders: leaders, Members: members}, nil
}

func (s *Service) requireStructural(ctx context.Context, session auth.Session, nodeID int64, message string) error {
	if session.Permissions.SystemAdmin {
		return nil
	}
	allowed, err := s.access.IsEffectiveLeader(ctx, nodeID, session.Subject)
	if err != nil {
		return err
	}
	if !allowed {
		return apperr.Forbidden(message)
	}
	return nil
}

func (s *Service) normalizeMembers(ctx context.Context, parentID *int64, leaderInput, memberInput []string, fallbackLeader string) ([]string, []string, error) {
	leaders := normalizeSubjects(leaderInput)
	members := normalizeSubjects(memberInput)
	if len(leaders) == 0 && fallbackLeader != "" {
		leaders = []string{fallbackLeader}
	}
	if len(leaders) == 0 {
		return nil, nil, apperr.BadRequest("节点至少需要一个直接负责人")
	}
	leaderSet := stringSet(leaders)
	if len(members) == 0 && parentID != nil {
		parentMembers, err := s.repo.Members(ctx, *parentID)
		if err != nil {
			return nil, nil, err
		}
		for _, member := range parentMembers {
			if _, isLeader := leaderSet[member.Subject]; member.Role == "member" && !isLeader {
				members = append(members, member.Subject)
			}
		}
	}
	for _, member := range members {
		if _, ok := leaderSet[member]; ok {
			return nil, nil, apperr.BadRequest("负责人和成员不能重叠")
		}
	}
	all := append(append([]string{}, leaders...), members...)
	exists, err := s.users.AllExist(ctx, all)
	if err != nil {
		return nil, nil, err
	}
	if !exists {
		return nil, nil, apperr.BadRequest("包含不存在的用户")
	}
	return leaders, members, nil
}

func (s *Service) resolveDirectMembers(ctx context.Context, session auth.Session, rows []MemberRow) ([]users.Summary, []users.Summary) {
	subjects := make([]string, 0, len(rows))
	for _, row := range rows {
		subjects = append(subjects, row.Subject)
	}
	summaries := s.users.Summaries(ctx, session, subjects)
	leaders := make([]users.Summary, 0)
	members := make([]users.Summary, 0)
	for _, row := range rows {
		summary, ok := summaries[row.Subject]
		if !ok {
			summary = users.Summary{Subject: row.Subject, UserName: row.Subject, Positions: []users.Position{}}
		}
		if row.Role == "leader" {
			leaders = append(leaders, summary)
		} else {
			members = append(members, summary)
		}
	}
	return leaders, members
}

func (s *Service) childViews(ctx context.Context, session auth.Session, rows []ChildRow, leaders, members map[int64]struct{}) ([]Child, error) {
	if len(rows) == 0 {
		return []Child{}, nil
	}
	latestIDs := make([]int64, 0)
	authorSubjects := make([]string, 0)
	for _, row := range rows {
		if row.Latest != nil {
			latestIDs = append(latestIDs, row.Latest.ID)
			authorSubjects = append(authorSubjects, row.Latest.Author)
		}
	}
	authors := s.users.Summaries(ctx, session, authorSubjects)
	reactions, err := s.repo.ReactionsForComments(ctx, latestIDs, session.Subject)
	if err != nil {
		return nil, err
	}
	result := make([]Child, 0, len(rows))
	for _, row := range rows {
		summary := Summary{
			ID:           row.ID,
			ParentID:     row.ParentID,
			Title:        row.Title,
			DueAt:        row.DueAt,
			CompletedAt:  row.CompletedAt,
			Tags:         nonNil(row.Tags),
			SortOrder:    row.SortOrder,
			ChildCount:   row.ChildCount,
			CommentCount: row.CommentCount,
			MyRelation:   relationFor(row.ID, leaders, members),
			UpdatedAt:    row.UpdatedAt,
		}
		child := Child{Summary: summary}
		if row.Latest != nil {
			author, ok := authors[row.Latest.Author]
			if !ok {
				author = users.Summary{Subject: row.Latest.Author, UserName: row.Latest.Author, Positions: []users.Position{}}
			}
			preview := comments.BuildPreview(row.Latest.ID, author, row.Latest.Body, row.Latest.CreatedAt, reactions[row.Latest.ID], row.Latest.AttachmentCount)
			child.LatestComment = &preview
			latestAt := row.Latest.CreatedAt
			child.Summary.LatestCommentAt = &latestAt
		}
		result = append(result, child)
	}
	return result, nil
}

func (s *Service) recentViews(ctx context.Context, session auth.Session, rows []RecentRow) ([]comments.Recent, error) {
	if len(rows) == 0 {
		return []comments.Recent{}, nil
	}
	ids := make([]int64, 0, len(rows))
	subjects := make([]string, 0, len(rows))
	for _, row := range rows {
		ids = append(ids, row.ID)
		subjects = append(subjects, row.Author)
	}
	authors := s.users.Summaries(ctx, session, subjects)
	reactions, err := s.repo.ReactionsForComments(ctx, ids, session.Subject)
	if err != nil {
		return nil, err
	}
	result := make([]comments.Recent, 0, len(rows))
	for _, row := range rows {
		author, ok := authors[row.Author]
		if !ok {
			author = users.Summary{Subject: row.Author, UserName: row.Author, Positions: []users.Position{}}
		}
		summary := reactions[row.ID]
		if summary == nil {
			summary = []comments.Reaction{}
		}
		result = append(result, comments.Recent{
			ID:              row.ID,
			NodeID:          row.NodeID,
			NodeTitle:       row.NodeTitle,
			RootTitle:       row.RootTitle,
			Author:          author,
			BodyPreview:     comments.TextPreview(row.Body, 120),
			CreatedAt:       row.CreatedAt,
			ReactionSummary: summary,
		})
	}
	return result, nil
}

func normalizeNodeFields(title, description string, tags []string) (string, string, []string, error) {
	title = strings.TrimSpace(title)
	if count := utf8.RuneCountInString(title); count < 1 || count > 200 {
		return "", "", nil, apperr.BadRequest("标题长度必须为 1–200 字符")
	}
	description = strings.TrimSpace(description)
	if utf8.RuneCountInString(description) > 20000 {
		return "", "", nil, apperr.BadRequest("描述不能超过 20000 字符")
	}
	normalizedTags, err := normalizeTags(tags)
	if err != nil {
		return "", "", nil, err
	}
	return title, description, normalizedTags, nil
}

func normalizeTags(tags []string) ([]string, error) {
	result := make([]string, 0, len(tags))
	seen := make(map[string]struct{}, len(tags))
	for _, tag := range tags {
		tag = strings.TrimSpace(tag)
		if tag == "" {
			continue
		}
		if utf8.RuneCountInString(tag) > 32 {
			return nil, apperr.BadRequest("单个标签不能超过 32 字符")
		}
		if _, ok := seen[tag]; ok {
			continue
		}
		seen[tag] = struct{}{}
		result = append(result, tag)
	}
	if len(result) > 10 {
		return nil, apperr.BadRequest("最多只能有 10 个标签")
	}
	return result, nil
}

func normalizeSubjects(subjects []string) []string {
	result := make([]string, 0, len(subjects))
	seen := make(map[string]struct{}, len(subjects))
	for _, subject := range subjects {
		subject = strings.TrimSpace(subject)
		if subject == "" {
			continue
		}
		if _, ok := seen[subject]; ok {
			continue
		}
		seen[subject] = struct{}{}
		result = append(result, subject)
	}
	return result
}

func relationFor(id int64, leaders, members map[int64]struct{}) *string {
	if leaders != nil {
		if _, ok := leaders[id]; ok {
			value := "leader"
			return &value
		}
	}
	if members != nil {
		if _, ok := members[id]; ok {
			value := "member"
			return &value
		}
	}
	return nil
}

func stringSet(values []string) map[string]struct{} {
	result := make(map[string]struct{}, len(values))
	for _, value := range values {
		result[value] = struct{}{}
	}
	return result
}

func intersect(a, b map[string]struct{}) map[string]struct{} {
	result := make(map[string]struct{})
	for value := range a {
		if _, ok := b[value]; ok {
			result[value] = struct{}{}
		}
	}
	return result
}

func nonNil(values []string) []string {
	if values == nil {
		return []string{}
	}
	return values
}

func nonNilBreadcrumb(values []Breadcrumb) []Breadcrumb {
	if values == nil {
		return []Breadcrumb{}
	}
	return values
}

func sortMembers(members []EffectiveMember) {
	sort.SliceStable(members, func(i, j int) bool {
		if members[i].Direct != members[j].Direct {
			return members[i].Direct
		}
		return strings.ToLower(members[i].User.UserName) < strings.ToLower(members[j].User.UserName)
	})
}
