package httpapi

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"github.com/davepartner/crew/internal/payauth"
)

type errorBody struct {
	OK         bool   `json:"ok"`
	Error      string `json:"error"`
	RetryAfter int    `json:"retry_after,omitempty"`
	Code       string `json:"code,omitempty"`
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if payload == nil {
		return
	}
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		slog.Error("write json", "err", err)
	}
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, errorBody{OK: false, Error: message})
}

func decodeJSON(r *http.Request, dst any) error {
	defer r.Body.Close()

	dec := json.NewDecoder(io.LimitReader(r.Body, 1<<20))
	dec.DisallowUnknownFields()

	if err := dec.Decode(dst); err != nil {
		return errors.New("we couldn't read that request. Please try again.")
	}

	return nil
}

func trim(value string, max int) string {
	value = strings.TrimSpace(value)
	if len(value) > max {
		return value[:max]
	}
	return value
}

// authErrorMessage turns a PayTrackify failure into a sentence a person can act
// on. Raw provider text never reaches the browser.
func authErrorMessage(err error) (string, int, int) {
	var apiErr *payauth.Error
	if errors.As(err, &apiErr) {
		switch apiErr.Status {
		case http.StatusTooManyRequests:
			return "Too many tries. Please wait a moment and try again.", apiErr.Status, apiErr.RetryAfter
		case http.StatusUnprocessableEntity:
			return "That code is not right, or it has expired. Try again, or ask for a new code.", apiErr.Status, 0
		case http.StatusUnauthorized:
			return "Please sign in again.", apiErr.Status, 0
		default:
			if apiErr.Message != "" && apiErr.Status < 500 {
				return apiErr.Message, apiErr.Status, 0
			}
		}
	}

	return "Something went wrong on our side. Please try again in a moment.", http.StatusBadGateway, 0
}
