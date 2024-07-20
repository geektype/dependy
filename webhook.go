package main

import (
	"io"
	"log/slog"
	"net/http"
)

// TODO: Implement Webhooks
func hookHandler(w http.ResponseWriter, _ *http.Request) {
	slog.Info("Hook received")

	_, err := io.WriteString(w, "Pong")
	if err != nil {
		slog.Error("error", slog.Any("error", err))
	}
}
