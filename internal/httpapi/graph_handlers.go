package httpapi

import (
	"net/http"
	"strconv"

	"git.bitnp.net/BITNP/guild-go/internal/apperr"
)

func (s *Server) graph(w http.ResponseWriter, r *http.Request) {
	session, ok := currentSession(r)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, errorResponse{Status: 401, Error: "UNAUTHORIZED", Message: "需要登录"})
		return
	}
	cursor, err := parseGraphCursor(r)
	if err != nil {
		writeError(w, r, err)
		return
	}
	limit, err := parseGraphLimit(r)
	if err != nil {
		writeError(w, r, err)
		return
	}
	result, err := s.nodes.Graph(r.Context(), session, cursor, limit)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) graphNeighborhood(w http.ResponseWriter, r *http.Request) {
	session, ok := currentSession(r)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, errorResponse{Status: 401, Error: "UNAUTHORIZED", Message: "需要登录"})
		return
	}
	id, err := parseID(r, "id")
	if err != nil {
		writeError(w, r, err)
		return
	}
	result, err := s.nodes.GraphNeighborhood(r.Context(), session, id)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func parseGraphCursor(r *http.Request) (int64, error) {
	value := r.URL.Query().Get("cursor")
	if value == "" {
		return 0, nil
	}
	cursor, err := strconv.ParseInt(value, 10, 64)
	if err != nil || cursor <= 0 {
		return 0, apperr.BadRequest("cursor 参数不正确")
	}
	return cursor, nil
}

func parseGraphLimit(r *http.Request) (int, error) {
	value := r.URL.Query().Get("limit")
	if value == "" {
		return 200, nil
	}
	limit, err := strconv.Atoi(value)
	if err != nil || limit <= 0 {
		return 0, apperr.BadRequest("limit 参数不正确")
	}
	if limit > 300 {
		limit = 300
	}
	return limit, nil
}
