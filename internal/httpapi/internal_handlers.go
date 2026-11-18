package httpapi

import (
	"crypto/subtle"
	"net/http"
	"strings"

	"git.bitnp.net/BITNP/guild-go/internal/apperr"
	"git.bitnp.net/BITNP/guild-go/internal/auth"
	"git.bitnp.net/BITNP/guild-go/internal/nodes"
)

func (s *Server) internalAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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

		next.ServeHTTP(w, r.WithContext(auth.WithSession(r.Context(), session)))
	})
}

func (s *Server) createNodeInternal(w http.ResponseWriter, r *http.Request) {
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

func (s *Server) replaceMembersInternal(w http.ResponseWriter, r *http.Request) {
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

func bearerToken(header string) string {
	fields := strings.Fields(header)
	if len(fields) != 2 || !strings.EqualFold(fields[0], "Bearer") {
		return ""
	}
	return fields[1]
}
