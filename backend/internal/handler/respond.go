package handler

import (
	"net/http"

	"github.com/knowledgeos/backend/internal/respond"
)

func JSON(w http.ResponseWriter, status int, data interface{}) {
	respond.JSON(w, status, data)
}

func JSONList(w http.ResponseWriter, status int, data interface{}, total int64) {
	respond.JSONList(w, status, data, total)
}

func Error(w http.ResponseWriter, status int, msg string) {
	respond.Error(w, status, msg)
}

func Decode(r *http.Request, v interface{}) error {
	return respond.Decode(r, v)
}
