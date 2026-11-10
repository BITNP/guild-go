package httpapi

import (
	"encoding/json"
	"net/http"

	"git.bitnp.net/BITNP/guild-go/internal/apperr"
	"git.bitnp.net/BITNP/guild-go/internal/auth"
	"git.bitnp.net/BITNP/guild-go/internal/nodes"
)

func (s *Server) overview(w http.ResponseWriter, r *http.Request, session auth.Session) {
	result, err := s.nodes.Overview(r.Context(), session)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) listNodes(w http.ResponseWriter, r *http.Request, session auth.Session) {
	result, err := s.nodes.List(r.Context(), session)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) getNode(w http.ResponseWriter, r *http.Request, session auth.Session) {
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

func (s *Server) createNode(w http.ResponseWriter, r *http.Request, session auth.Session) {
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

func (s *Server) updateNode(w http.ResponseWriter, r *http.Request, session auth.Session) {
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

func (s *Server) moveNode(w http.ResponseWriter, r *http.Request, session auth.Session) {
	id, err := parseID(r, "id")
	if err != nil {
		writeError(w, r, err)
		return
	}
	var payload struct {
		ParentID json.RawMessage `json:"parentId"`
	}
	if err := decodeJSON(w, r, &payload); err != nil {
		writeError(w, r, err)
		return
	}
	var request nodes.MoveRequest
	if len(payload.ParentID) == 0 || json.Unmarshal(payload.ParentID, &request.ParentID) != nil {
		writeError(w, r, apperr.BadRequest("必须指定 parentId，移动为根节点时传 null"))
		return
	}
	result, err := s.nodes.Move(r.Context(), session, id, request)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) deleteNode(w http.ResponseWriter, r *http.Request, session auth.Session) {
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

func (s *Server) orderNodes(w http.ResponseWriter, r *http.Request, session auth.Session) {
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

func (s *Server) getMembers(w http.ResponseWriter, r *http.Request, session auth.Session) {
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

func (s *Server) replaceMembers(w http.ResponseWriter, r *http.Request, session auth.Session) {
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
