package httpapi

import (
	"net/http"
	"time"

	"git.bitnp.net/BITNP/guild-go/internal/apperr"
	"git.bitnp.net/BITNP/guild-go/internal/comments"
)

func (s *Server) listProgress(w http.ResponseWriter, r *http.Request) {
	session, ok := currentSession(r)
	if !ok {
		writeError(w, r, apperr.Unauthorized("需要登录"))
		return
	}
	nodeID, err := parseID(r, "id")
	if err != nil {
		writeError(w, r, err)
		return
	}
	limit, err := parseLimit(r)
	if err != nil {
		writeError(w, r, err)
		return
	}
	before, err := parseProgressCursor(r)
	if err != nil {
		writeError(w, r, err)
		return
	}
	result, err := s.comments.Progress(r.Context(), session, nodeID, before, limit)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func parseProgressCursor(r *http.Request) (*comments.ProgressCursor, error) {
	query := r.URL.Query()
	if !query.Has("before") && !query.Has("beforeKind") && !query.Has("beforeId") {
		return nil, nil
	}
	createdAt, err := time.Parse(time.RFC3339Nano, query.Get("before"))
	if err != nil {
		return nil, apperr.BadRequest("进展分页参数不正确")
	}
	kind := query.Get("beforeKind")
	if kind != "comment" && kind != "description_change" {
		return nil, apperr.BadRequest("进展分页参数不正确")
	}
	id, err := parseOptionalInt64(r, "beforeId")
	if err != nil || id == nil {
		return nil, apperr.BadRequest("进展分页参数不正确")
	}
	return &comments.ProgressCursor{CreatedAt: createdAt, Kind: kind, ID: *id}, nil
}
