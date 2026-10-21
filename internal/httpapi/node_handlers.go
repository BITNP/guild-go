package httpapi

import (
	"net/http"

	"git.bitnp.net/BITNP/guild-go/internal/nodes"
)

func (s *Server) overview(w http.ResponseWriter, r *http.Request) {
	session, ok := currentSession(r)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, errorResponse{Status: 401, Error: "UNAUTHORIZED", Message: "需要登录"})
		return
	}
	result, err := s.nodes.Overview(r.Context(), session)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) listNodes(w http.ResponseWriter, r *http.Request) {
	session, ok := currentSession(r)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, errorResponse{Status: 401, Error: "UNAUTHORIZED", Message: "需要登录"})
		return
	}
	result, err := s.nodes.List(r.Context(), session)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) getNode(w http.ResponseWriter, r *http.Request) {
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
	result, err := s.nodes.Get(r.Context(), session, id)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) createNode(w http.ResponseWriter, r *http.Request) {
	session, ok := currentSession(r)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, errorResponse{Status: 401, Error: "UNAUTHORIZED", Message: "需要登录"})
		return
	}
	var request nodes.CreateRequest
	if err := decodeJSON(w, r, &request); err != nil {
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

func (s *Server) updateNode(w http.ResponseWriter, r *http.Request) {
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
	var request nodes.UpdateRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeError(w, r, err)
		return
	}
	result, err := s.nodes.Update(r.Context(), session, id, request)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) deleteNode(w http.ResponseWriter, r *http.Request) {
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
	if err := s.nodes.Delete(r.Context(), session, id); err != nil {
		writeError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) orderNodes(w http.ResponseWriter, r *http.Request) {
	session, ok := currentSession(r)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, errorResponse{Status: 401, Error: "UNAUTHORIZED", Message: "需要登录"})
		return
	}
	var request nodes.OrderRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeError(w, r, err)
		return
	}
	if err := s.nodes.Reorder(r.Context(), session, request.ParentID, request.IDs); err != nil {
		writeError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) getMembers(w http.ResponseWriter, r *http.Request) {
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
	result, err := s.nodes.EffectiveMembers(r.Context(), session, id)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) replaceMembers(w http.ResponseWriter, r *http.Request) {
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
	var request nodes.MembersRequest
	if err := decodeJSON(w, r, &request); err != nil {
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
