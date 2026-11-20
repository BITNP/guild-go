package nodes

import (
	"context"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5"

	"git.bitnp.net/BITNP/guild-go/internal/apperr"
	"git.bitnp.net/BITNP/guild-go/internal/auth"
	"git.bitnp.net/BITNP/guild-go/internal/database"
	"git.bitnp.net/BITNP/guild-go/internal/events"
	"git.bitnp.net/BITNP/guild-go/internal/users"
)

// MemberRow is a direct member row.
type MemberRow struct {
	Subject string
	Role    string
}

// Members returns direct leaders and members.
func (r *Repository) Members(ctx context.Context, nodeID int64) ([]MemberRow, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT subject, role FROM node_members WHERE node_id=$1
		ORDER BY CASE role WHEN 'leader' THEN 0 ELSE 1 END, created_at, subject`, nodeID)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (MemberRow, error) {
		var member MemberRow
		err := row.Scan(&member.Subject, &member.Role)
		return member, err
	})
}

// ReplaceMembers updates the direct membership set without recreating unchanged rows.
func (r *Repository) ReplaceMembers(ctx context.Context, id int64, leaders, members []string, actor string) error {
	var eventIDs []int64
	err := database.WithTx(ctx, r.pool, func(tx pgx.Tx) error {
		var (
			lockedID int64
			title    string
		)
		if err := tx.QueryRow(ctx, `SELECT id, title FROM nodes WHERE id=$1 FOR UPDATE`, id).Scan(&lockedID, &title); err != nil {
			if err == pgx.ErrNoRows {
				return apperr.NotFound("节点不存在")
			}
			return err
		}

		prevRoles := make(map[string]string)
		existingRows, err := tx.Query(ctx, `SELECT subject, role FROM node_members WHERE node_id=$1`, id)
		if err != nil {
			return err
		}
		for existingRows.Next() {
			var subject, role string
			if err := existingRows.Scan(&subject, &role); err != nil {
				existingRows.Close()
				return err
			}
			prevRoles[subject] = role
		}
		if err := existingRows.Err(); err != nil {
			existingRows.Close()
			return err
		}
		existingRows.Close()

		// Keep unchanged (subject, role) pairs; recreate only removed/changed roles.
		if _, err := tx.Exec(ctx, `DELETE FROM node_members
            WHERE node_id=$1 AND NOT (
                (role='leader' AND subject = ANY(COALESCE($2::text[], '{}'))) OR
                (role='member' AND subject = ANY(COALESCE($3::text[], '{}')))
            )`, id, leaders, members); err != nil {
			return err
		}
		for role, subjects := range map[string][]string{"leader": leaders, "member": members} {
			for _, subject := range subjects {
				if _, err := tx.Exec(ctx, `INSERT INTO node_members (node_id, subject, role)
                    VALUES ($1,$2,$3) ON CONFLICT (node_id, subject) DO NOTHING`, id, subject, role); err != nil {
					return err
				}
			}
		}
		if _, err := tx.Exec(ctx, `UPDATE nodes SET updated_at=now() WHERE id=$1`, id); err != nil {
			return err
		}

		newRoles := make(map[string]string, len(leaders)+len(members))
		for _, subject := range leaders {
			newRoles[subject] = "leader"
		}
		for _, subject := range members {
			newRoles[subject] = "member"
		}

		removed := make([]string, 0)
		for subject := range prevRoles {
			if _, ok := newRoles[subject]; !ok {
				removed = append(removed, subject)
			}
		}
		sort.Strings(removed)
		for _, subject := range removed {
			evID, err := events.RecordTx(ctx, tx, events.Input{
				Kind:         events.KindMemberRemoved,
				NodeID:       &id,
				ActorSubject: actor,
				Payload: map[string]any{
					"subject":   subject,
					"role":      prevRoles[subject],
					"nodeTitle": title,
				},
			})
			if err != nil {
				return err
			}
			eventIDs = append(eventIDs, evID)
		}

		assigned := make([]string, 0)
		for subject, role := range newRoles {
			if oldRole, ok := prevRoles[subject]; !ok || oldRole != role {
				assigned = append(assigned, subject)
			}
		}
		sort.Strings(assigned)
		for _, subject := range assigned {
			payload := map[string]any{
				"subject":   subject,
				"role":      newRoles[subject],
				"nodeTitle": title,
			}
			if oldRole, ok := prevRoles[subject]; ok {
				payload["previousRole"] = oldRole
			}
			evID, err := events.RecordTx(ctx, tx, events.Input{
				Kind:         events.KindMemberAssigned,
				NodeID:       &id,
				ActorSubject: actor,
				Payload:      payload,
			})
			if err != nil {
				return err
			}
			eventIDs = append(eventIDs, evID)
		}
		return nil
	})
	if err != nil {
		return err
	}
	r.broker.Publish(eventIDs...)
	return nil
}

// RootLeaders returns direct leader subjects for root nodes.
func (r *Repository) RootLeaders(ctx context.Context, rootIDs []int64) (map[int64][]string, error) {
	result := make(map[int64][]string, len(rootIDs))
	if len(rootIDs) == 0 {
		return result, nil
	}
	rows, err := r.pool.Query(ctx, `
		SELECT node_id, subject FROM node_members
		WHERE node_id = ANY($1) AND role='leader'
		ORDER BY node_id, created_at, subject`, rootIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var nodeID int64
		var subject string
		if err := rows.Scan(&nodeID, &subject); err != nil {
			return nil, err
		}
		result[nodeID] = append(result[nodeID], subject)
	}
	return result, rows.Err()
}

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
	if err := s.repo.ReplaceMembers(ctx, id, leaders, members, session.Subject); err != nil {
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
		entry := EffectiveMember{User: summaries.For(subject), Direct: item.direct, Sources: sources}
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
		summary := summaries.For(row.Subject)
		if row.Role == "leader" {
			leaders = append(leaders, summary)
		} else {
			members = append(members, summary)
		}
	}
	return leaders, members
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

func sortMembers(members []EffectiveMember) {
	sort.SliceStable(members, func(i, j int) bool {
		if members[i].Direct != members[j].Direct {
			return members[i].Direct
		}
		return strings.ToLower(members[i].User.UserName) < strings.ToLower(members[j].User.UserName)
	})
}
