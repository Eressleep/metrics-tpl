package handlers

import (
	"encoding/json"
	"net/http"
)

type ResponseWriterHelper struct{}

func (h *ResponseWriterHelper) JSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if data != nil {
		json.NewEncoder(w).Encode(data)
	}
}

func (h *ResponseWriterHelper) Error(w http.ResponseWriter, status int, message string) {
	h.JSON(w, status, map[string]string{"error": message})
}
