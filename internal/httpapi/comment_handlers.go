package httpapi

import (
	"errors"
	"io"
	"net/http"
	"strconv"

	"git.bitnp.net/BITNP/guild-go/internal/apperr"
	"git.bitnp.net/BITNP/guild-go/internal/comments"
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
	body, uploads, err := readCommentForm(w, r)
	if err != nil {
		writeError(w, r, err)
		return
	}
	result, err := s.comments.Create(r.Context(), session, nodeID, body, uploads)
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

func readCommentForm(w http.ResponseWriter, r *http.Request) (string, []comments.Upload, error) {
	maxBytes := int64(storage.MaxAttachments*storage.MaxAttachmentBytes) + (2 << 20)
	r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			return "", nil, apperr.TooLarge("附件总大小超过限制")
		}
		return "", nil, apperr.BadRequest("评论表单格式不正确")
	}
	if r.MultipartForm != nil {
		defer r.MultipartForm.RemoveAll()
	}
	body := r.FormValue("body")
	files := r.MultipartForm.File["files"]
	if len(files) > storage.MaxAttachments {
		return "", nil, apperr.BadRequest("每条评论最多 4 张图片")
	}
	uploads := make([]comments.Upload, 0, len(files))
	for _, header := range files {
		file, err := header.Open()
		if err != nil {
			return "", nil, apperr.BadRequest("无法读取上传图片")
		}
		data, err := io.ReadAll(io.LimitReader(file, storage.MaxAttachmentBytes+1))
		_ = file.Close()
		if err != nil {
			return "", nil, apperr.BadRequest("无法读取上传图片")
		}
		if len(data) > storage.MaxAttachmentBytes {
			return "", nil, apperr.TooLarge("单张图片不能超过 5MB")
		}
		uploads = append(uploads, comments.Upload{Data: data})
	}
	return body, uploads, nil
}
