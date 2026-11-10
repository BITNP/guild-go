package httpapi

import (
	"net/http"

	"git.bitnp.net/BITNP/guild-go/internal/auth"
)

func (s *Server) listUsers(w http.ResponseWriter, r *http.Request, session auth.Session) {
	result, err := s.users.List(r.Context(), session)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) getUser(w http.ResponseWriter, r *http.Request, session auth.Session) {
	subject := r.PathValue("subject")
	result, err := s.users.Get(r.Context(), session, subject)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}
