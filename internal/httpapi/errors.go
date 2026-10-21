package httpapi

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"

	"git.bitnp.net/BITNP/guild-go/internal/apperr"
)

type errorResponse struct {
	Status  int    `json:"status"`
	Error   string `json:"error"`
	Message string `json:"message"`
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if value != nil {
		_ = json.NewEncoder(w).Encode(value)
	}
}

func writeError(w http.ResponseWriter, r *http.Request, err error) {
	app := apperr.From(err)
	if app.Status >= 500 {
		slog.Error("request failed", "method", r.Method, "path", r.URL.Path, "error", err)
	}
	writeJSON(w, app.Status, errorResponse{Status: app.Status, Error: app.Code, Message: app.Message})
}

func decodeJSON(w http.ResponseWriter, r *http.Request, target any) error {
	r.Body = http.MaxBytesReader(w, r.Body, 2<<20)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		if errors.Is(err, io.EOF) {
			return apperr.BadRequest("请求体不能为空")
		}
		return apperr.BadRequest("请求体格式不正确")
	}
	if decoder.More() {
		return apperr.BadRequest("请求体只能包含一个 JSON 对象")
	}
	return nil
}
