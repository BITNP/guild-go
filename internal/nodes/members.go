package nodes

import (
	"context"
	"sort"
	"strings"

	"git.bitnp.net/BITNP/guild-go/internal/apperr"
	"git.bitnp.net/BITNP/guild-go/internal/auth"
	"git.bitnp.net/BITNP/guild-go/internal/users"
)

// ReplaceMembers replaces direct leaders and members.
func (s *Service) ReplaceMembers(ctx context.Context, session auth.Session, id int64, req MembersRequest) (Detail, error) {
	if _, err := s.repo.Get(ctx, id); err != nil {
		return Detail{}, err
	}
	if err := s.requireStructural(ctx, session, id, "只有节点负责人可以管理成员"); err != nil {
		return Detail{}, err
	}

	leaders, members, err := s.normalizeMembers(ctx, req.Leaders, req.Members, "")
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

func (s *Service) normalizeMembers(ctx context.Context, leaderInput, memberInput []string, fallbackLeader string) ([]string, []string, error) {
	leaders := normalizeSubjects(leaderInput)
	members := normalizeSubjects(memberInput)

	if len(leaders) == 0 && fallbackLeader != "" {
		leaders = []string{fallbackLeader}
	}

	if len(leaders) == 0 {
		return nil, nil, apperr.BadRequest("节点至少需要一个直接负责人")
	}

	leaderSet := stringSet(leaders)
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

func sortMembers(members []EffectiveMember) {
	sort.SliceStable(members, func(i, j int) bool {
		if members[i].Direct != members[j].Direct {
			return members[i].Direct
		}

		return strings.ToLower(members[i].User.UserName) < strings.ToLower(members[j].User.UserName)
	})
}
