package httpapi

import (
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"git.bitnp.net/BITNP/guild-go/internal/apperr"
	"git.bitnp.net/BITNP/guild-go/internal/auth"
	"git.bitnp.net/BITNP/guild-go/internal/events"
	"git.bitnp.net/BITNP/guild-go/internal/nodes"
)

func (s *Server) handleInternal(mux *http.ServeMux, pattern string, fn sessionHandler) {
	mux.HandleFunc(pattern, func(w http.ResponseWriter, r *http.Request) {
		if s.internalToken == "" || s.internalAuthz == nil {
			writeJSON(w, http.StatusNotFound, errorResponse{
				Status:  http.StatusNotFound,
				Error:   "NOT_FOUND",
				Message: "接口不存在",
			})
			return
		}

		token := bearerToken(r.Header.Get("Authorization"))
		if token == "" || subtle.ConstantTimeCompare([]byte(token), []byte(s.internalToken)) != 1 {
			writeError(w, r, apperr.Unauthorized("内部服务令牌无效"))
			return
		}

		subject := strings.TrimSpace(r.Header.Get("X-Act-As-Subject"))
		if subject == "" {
			writeError(w, r, apperr.BadRequest("必须提供 X-Act-As-Subject"))
			return
		}

		session, err := s.internalAuthz.ResolveActorSession(r.Context(), subject)
		if err != nil {
			writeError(w, r, err)
			return
		}

		fn(w, r, session)
	})
}

func (s *Server) internalServiceAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if s.internalToken == "" || s.internalFeed == nil {
			writeJSON(w, http.StatusNotFound, errorResponse{
				Status:  http.StatusNotFound,
				Error:   "NOT_FOUND",
				Message: "接口不存在",
			})
			return
		}

		token := bearerToken(r.Header.Get("Authorization"))
		if token == "" || subtle.ConstantTimeCompare([]byte(token), []byte(s.internalToken)) != 1 {
			writeError(w, r, apperr.Unauthorized("内部服务令牌无效"))
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (s *Server) createNodeInternal(w http.ResponseWriter, r *http.Request, session auth.Session) {
	var request nodes.CreateRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeError(w, r, err)
		return
	}

	subjects := make([]string, 0, len(request.LeaderSubjects)+len(request.MemberSubjects))
	subjects = append(subjects, request.LeaderSubjects...)
	subjects = append(subjects, request.MemberSubjects...)
	if err := s.internalAuthz.EnsureSubjects(r.Context(), subjects); err != nil {
		writeError(w, r, err)
		return
	}

	result, err := s.nodes.Create(r.Context(), session, request)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, result)
}

func (s *Server) replaceMembersInternal(w http.ResponseWriter, r *http.Request, session auth.Session) {
	id, err := parseID(r, "id")
	if err != nil {
		writeError(w, r, err)
		return
	}

	var request nodes.MembersRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeError(w, r, err)
		return
	}

	subjects := make([]string, 0, len(request.Leaders)+len(request.Members))
	subjects = append(subjects, request.Leaders...)
	subjects = append(subjects, request.Members...)
	if err := s.internalAuthz.EnsureSubjects(r.Context(), subjects); err != nil {
		writeError(w, r, err)
		return
	}

	result, err := s.nodes.ReplaceMembers(r.Context(), session, id, request)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) listEventsInternal(w http.ResponseWriter, r *http.Request) {
	after, err := parseCursorParam(r.URL.Query().Get("after"))
	if err != nil {
		writeError(w, r, apperr.BadRequest("after 参数不正确"))
		return
	}

	limit, err := parseBoundedLimit(r.URL.Query().Get("limit"), 50, 200)
	if err != nil {
		writeError(w, r, err)
		return
	}

	result, err := s.internalFeed.ListAfter(r.Context(), after, limit)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) streamEventsInternal(w http.ResponseWriter, r *http.Request) {
	rawAfter := strings.TrimSpace(r.URL.Query().Get("after"))
	if rawAfter == "" {
		rawAfter = strings.TrimSpace(r.Header.Get("Last-Event-ID"))
	}

	var (
		cursor int64
		err    error
	)
	if rawAfter != "" {
		cursor, err = parseCursorParam(rawAfter)
		if err != nil {
			writeError(w, r, apperr.BadRequest("after 参数不正确"))
			return
		}
	}

	// Subscribe before reading the tail cursor so no commit between LatestID and
	// Subscribe can slip past unnoticed.
	notify, unsubscribe := s.eventBroker.Subscribe()
	defer unsubscribe()

	if rawAfter == "" {
		cursor, err = s.internalFeed.LatestID(r.Context())
		if err != nil {
			writeError(w, r, err)
			return
		}
	}

	rc := http.NewResponseController(w)
	_ = rc.SetWriteDeadline(time.Now().Add(45 * time.Second))

	w.Header().Set("Content-Type", "text/event-stream; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache, no-transform")
	w.Header().Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)

	if _, err := io.WriteString(w, ": connected\n\n"); err != nil {
		return
	}
	if err := rc.Flush(); err != nil {
		return
	}

	drain := func() error {
		for {
			batch, err := s.internalFeed.ListAfter(r.Context(), cursor, 100)
			if err != nil {
				return err
			}
			for _, ev := range batch.Items {
				encoded, err := json.Marshal(ev)
				if err != nil {
					return err
				}
				_ = rc.SetWriteDeadline(time.Now().Add(45 * time.Second))
				if _, err := fmt.Fprintf(w, "id: %d\nevent: %s\ndata: %s\n\n", ev.ID, ev.Kind, encoded); err != nil {
					return err
				}
				cursor = ev.ID
			}
			if len(batch.Items) > 0 {
				if err := rc.Flush(); err != nil {
					return err
				}
			}
			if !batch.HasMore {
				return nil
			}
		}
	}

	if err := drain(); err != nil {
		return
	}

	heartbeat := time.NewTicker(20 * time.Second)
	defer heartbeat.Stop()

	for {
		select {
		case <-r.Context().Done():
			return
		case _, ok := <-notify:
			if err := drain(); err != nil {
				return
			}
			if !ok {
				notify = nil
			}
		case <-heartbeat.C:
			_ = rc.SetWriteDeadline(time.Now().Add(45 * time.Second))
			if _, err := io.WriteString(w, ": ping\n\n"); err != nil {
				return
			}
			if err := rc.Flush(); err != nil {
				return
			}
			if err := drain(); err != nil {
				return
			}
		}
	}
}

func (s *Server) dueNodesInternal(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	var query events.DueNodesQuery

	if since, err := parseOptionalRFC3339(q.Get("since"), "since"); err != nil {
		writeError(w, r, err)
		return
	} else {
		query.Since = since
	}

	if dueAfter, err := parseOptionalRFC3339(q.Get("dueAfter"), "dueAfter"); err != nil {
		writeError(w, r, err)
		return
	} else {
		query.DueAfter = dueAfter
	}

	if dueBefore, err := parseOptionalRFC3339(q.Get("dueBefore"), "dueBefore"); err != nil {
		writeError(w, r, err)
		return
	} else {
		query.DueBefore = dueBefore
	}

	if rawAdvance := strings.TrimSpace(q.Get("advanceHours")); rawAdvance != "" {
		hours, err := strconv.ParseFloat(rawAdvance, 64)
		if err != nil || math.IsNaN(hours) || math.IsInf(hours, 0) || hours < 0 {
			writeError(w, r, apperr.BadRequest("advanceHours 参数不正确"))
			return
		}
		query.Advance = time.Duration(hours * float64(time.Hour))
	}

	if rawOverdue := strings.TrimSpace(q.Get("includeOverdue")); rawOverdue != "" {
		parsed, err := strconv.ParseBool(rawOverdue)
		if err != nil {
			writeError(w, r, apperr.BadRequest("includeOverdue 参数不正确"))
			return
		}
		query.IncludeOverdue = parsed
	}

	limit, err := parseBoundedLimit(q.Get("limit"), 100, 500)
	if err != nil {
		writeError(w, r, err)
		return
	}
	query.Limit = limit

	result, err := s.internalFeed.ListDueNodes(r.Context(), query)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) activitySummaryInternal(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	startPtr, err := parseOptionalRFC3339(q.Get("start"), "start")
	if err != nil {
		writeError(w, r, err)
		return
	}
	endPtr, err := parseOptionalRFC3339(q.Get("end"), "end")
	if err != nil {
		writeError(w, r, err)
		return
	}

	var start, end time.Time
	switch {
	case startPtr == nil && endPtr == nil:
		start, end = previousUTC8NaturalWeek(time.Now())
	case startPtr == nil || endPtr == nil:
		writeError(w, r, apperr.BadRequest("start 与 end 需同时提供或同时省略"))
		return
	default:
		start = *startPtr
		end = *endPtr
		if !end.After(start) {
			writeError(w, r, apperr.BadRequest("end 必须晚于 start"))
			return
		}
	}

	result, err := s.internalFeed.ActivitySummary(r.Context(), start, end)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func previousUTC8NaturalWeek(now time.Time) (time.Time, time.Time) {
	loc := time.FixedZone("UTC+8", 8*3600)
	local := now.In(loc)
	weekdayMon0 := (int(local.Weekday()) + 6) % 7
	endLocal := time.Date(local.Year(), local.Month(), local.Day()-weekdayMon0, 0, 0, 0, 0, loc)
	startLocal := endLocal.AddDate(0, 0, -7)
	return startLocal.UTC(), endLocal.UTC()
}

func parseCursorParam(raw string) (int64, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return 0, nil
	}
	parsed, err := strconv.ParseInt(trimmed, 10, 64)
	if err != nil || parsed < 0 {
		return 0, apperr.BadRequest("cursor 参数不正确")
	}
	return parsed, nil
}

func parseBoundedLimit(raw string, defaultValue, maxValue int) (int, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return defaultValue, nil
	}
	parsed, err := strconv.Atoi(trimmed)
	if err != nil || parsed <= 0 {
		return 0, apperr.BadRequest("limit 参数不正确")
	}
	if parsed > maxValue {
		return maxValue, nil
	}
	return parsed, nil
}

func parseOptionalRFC3339(raw, name string) (*time.Time, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil, nil
	}
	parsed, err := time.Parse(time.RFC3339Nano, trimmed)
	if err != nil {
		return nil, apperr.BadRequest(name + " 必须为 RFC3339 时间格式")
	}
	return &parsed, nil
}

func bearerToken(header string) string {
	fields := strings.Fields(header)
	if len(fields) != 2 || !strings.EqualFold(fields[0], "Bearer") {
		return ""
	}
	return fields[1]
}
