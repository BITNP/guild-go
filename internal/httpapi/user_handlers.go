package httpapi

import (
	"net/http"
)

func (s *Server) listUsers(w http.ResponseWriter, r *http.Request) {
	session, ok := currentSession(r)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, errorResponse{Status: 401, Error: "UNAUTHORIZED", Message: "需要登录"})
		return
	}
	result, err := s.users.List(r.Context(), session)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) getUser(w http.ResponseWriter, r *http.Request) {
	session, ok := currentSession(r)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, errorResponse{Status: 401, Error: "UNAUTHORIZED", Message: "需要登录"})
		return
	}
	subject := r.PathValue("subject")
	result, err := s.users.Get(r.Context(), session, subject)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}
