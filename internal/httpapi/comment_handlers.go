package httpapi

import (
	"errors"
	"io"
	"net/http"
	"strconv"

	"git.bitnp.net/BITNP/guild-go/internal/apperr"
	"git.bitnp.net/BITNP/guild-go/internal/storage"
)

func (s *Server) listComments(w http.ResponseWriter, r *http.Request) {
	session, ok := currentSession(r)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, errorResponse{Status: 401, Error: "UNAUTHORIZED", Message: "需要登录"})
		return
	}
	nodeID, err := parseID(r, "id")
	if err != nil {
		writeError(w, r, err)
		return
	}
	before, err := parseOptionalInt64(r, "before")
	if err != nil {
		writeError(w, r, err)
		return
	}
	limit, err := parseLimit(r)
	if err != nil {
		writeError(w, r, err)
		return
	}
	result, err := s.comments.List(r.Context(), session, nodeID, before, limit)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) createComment(w http.ResponseWriter, r *http.Request) {
	session, ok := currentSession(r)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, errorResponse{Status: 401, Error: "UNAUTHORIZED", Message: "需要登录"})
		return
	}
	nodeID, err := parseID(r, "id")
	if err != nil {
		writeError(w, r, err)
		return
	}
	var request struct {
		Body          string  `json:"body"`
		AttachmentIDs []int64 `json:"attachmentIds"`
	}
	if err := decodeJSON(w, r, &request); err != nil {
		writeError(w, r, err)
		return
	}
	result, err := s.comments.Create(r.Context(), session, nodeID, request.Body, request.AttachmentIDs)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, result)
}

func (s *Server) deleteComment(w http.ResponseWriter, r *http.Request) {
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
	if err := s.comments.Delete(r.Context(), session, id); err != nil {
		writeError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) reactionUsers(w http.ResponseWriter, r *http.Request) {
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
	result, err := s.comments.ReactionUsers(r.Context(), session, id, r.PathValue("emoji"))
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) toggleReaction(w http.ResponseWriter, r *http.Request) {
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
	result, err := s.comments.ToggleReaction(r.Context(), id, session.Subject, r.PathValue("emoji"))
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"reactions": result})
}

func (s *Server) getAttachment(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r, "id")
	if err != nil {
		writeError(w, r, err)
		return
	}
	reader, contentType, size, err := s.comments.GetAttachment(r.Context(), id)
	if errors.Is(err, storage.ErrNotFound) {
		writeJSON(w, http.StatusNotFound, errorResponse{Status: 404, Error: "NOT_FOUND", Message: "附件不存在"})
		return
	}
	if err != nil {
		writeError(w, r, err)
		return
	}
	defer reader.Close()
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Cache-Control", "private, max-age=3600")
	w.Header().Set("Content-Disposition", "inline")
	if size > 0 {
		w.Header().Set("Content-Length", strconv.FormatInt(size, 10))
	}
	w.WriteHeader(http.StatusOK)
	_, _ = io.Copy(w, reader)
}

func (s *Server) uploadCommentImage(w http.ResponseWriter, r *http.Request) {
	session, ok := currentSession(r)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, errorResponse{Status: 401, Error: "UNAUTHORIZED", Message: "需要登录"})
		return
	}
	nodeID, err := parseID(r, "id")
	if err != nil {
		writeError(w, r, err)
		return
	}
	data, err := readCommentImage(w, r)
	if err != nil {
		writeError(w, r, err)
		return
	}
	id, err := s.comments.UploadImage(r.Context(), session, nodeID, data)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]int64{"id": id})
}

func (s *Server) deleteCommentUpload(w http.ResponseWriter, r *http.Request) {
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
	if err := s.comments.DeleteUpload(r.Context(), session, id); err != nil {
		writeError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func readCommentImage(w http.ResponseWriter, r *http.Request) ([]byte, error) {
	r.Body = http.MaxBytesReader(w, r.Body, storage.MaxAttachmentBytes+(1<<20))
	if err := r.ParseMultipartForm(storage.MaxAttachmentBytes); err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			return nil, apperr.TooLarge("单张图片不能超过 5MB")
		}
		return nil, apperr.BadRequest("图片上传格式不正确")
	}
	defer r.MultipartForm.RemoveAll()
	files := r.MultipartForm.File["file"]
	if len(files) != 1 || len(r.MultipartForm.File) != 1 {
		return nil, apperr.BadRequest("每次只能上传一张图片")
	}
	file, err := files[0].Open()
	if err != nil {
		return nil, apperr.BadRequest("无法读取上传图片")
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, storage.MaxAttachmentBytes+1))
	if err != nil {
		return nil, apperr.BadRequest("无法读取上传图片")
	}
	if len(data) > storage.MaxAttachmentBytes {
		return nil, apperr.TooLarge("单张图片不能超过 5MB")
	}
	return data, nil
}
