package nodes

import (
	"context"

	"git.bitnp.net/BITNP/guild-go/internal/access"
	"git.bitnp.net/BITNP/guild-go/internal/auth"
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
