package payauth

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// User is the identity PayTrackify vouches for. Crew never stores a password
// and never sees anything beyond this profile.
type User struct {
	ID            string  `json:"id"`
	Email         string  `json:"email"`
	Name          string  `json:"name"`
	Avatar        *string `json:"avatar"`
	EmailVerified bool    `json:"email_verified"`
	CreatedAt     string  `json:"created_at"`
}

type Session struct {
	ID         string `json:"id"`
	Device     string `json:"device"`
	IPAddress  string `json:"ip_address"`
	LastUsedAt string `json:"last_used_at"`
	CreatedAt  string `json:"created_at"`
}

type Error struct {
	Status     int
	Message    string
	RetryAfter int
}

func (e *Error) Error() string {
	if e.Message == "" {
		return fmt.Sprintf("paytrackify returned %d", e.Status)
	}
	return e.Message
}

func IsUnauthorized(err error) bool {
	var apiErr *Error
	return errors.As(err, &apiErr) && apiErr.Status == http.StatusUnauthorized
}

func IsRateLimited(err error) bool {
	var apiErr *Error
	return errors.As(err, &apiErr) && apiErr.Status == http.StatusTooManyRequests
}

type Client struct {
	baseURL string
	http    *http.Client
}

func NewClient(baseURL string, timeout time.Duration) *Client {
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		http: &http.Client{
			Timeout: timeout,
			Transport: &http.Transport{
				MaxIdleConns:        20,
				MaxIdleConnsPerHost: 10,
				IdleConnTimeout:     90 * time.Second,
			},
		},
	}
}

type VerifyResult struct {
	Token string
	User  User
	IsNew bool
}

func (c *Client) RequestCode(ctx context.Context, email, clientIP string) (requestID string, resendAfter int, err error) {
	var out struct {
		OK          bool   `json:"ok"`
		RequestID   string `json:"requestId"`
		ResendAfter int    `json:"resend_after"`
		Error       string `json:"error"`
	}

	if err := c.do(ctx, http.MethodPost, "/api/auth/otp/request", "", map[string]any{"email": email}, clientIP, &out); err != nil {
		return "", 0, err
	}

	return out.RequestID, out.ResendAfter, nil
}

func (c *Client) ResendCode(ctx context.Context, requestID, clientIP string) (resendAfter int, err error) {
	var out struct {
		OK          bool   `json:"ok"`
		ResendAfter int    `json:"resend_after"`
		Error       string `json:"error"`
	}

	if err := c.do(ctx, http.MethodPost, "/api/auth/otp/resend", "", map[string]any{"requestId": requestID}, clientIP, &out); err != nil {
		return 0, err
	}

	return out.ResendAfter, nil
}

func (c *Client) VerifyCode(ctx context.Context, requestID, code, clientIP string) (VerifyResult, error) {
	var out struct {
		OK        bool   `json:"ok"`
		Token     string `json:"token"`
		IsNewUser bool   `json:"is_new_user"`
		Error     string `json:"error"`
		User      User   `json:"user"`
	}

	if err := c.do(ctx, http.MethodPost, "/api/auth/otp/verify", "",
		map[string]any{"requestId": requestID, "code": code}, clientIP, &out); err != nil {
		return VerifyResult{}, err
	}

	return VerifyResult{Token: out.Token, User: out.User, IsNew: out.IsNewUser}, nil
}

func (c *Client) Me(ctx context.Context, token string) (User, error) {
	var out struct {
		OK    bool   `json:"ok"`
		User  User   `json:"user"`
		Error string `json:"error"`
	}

	if err := c.do(ctx, http.MethodGet, "/api/auth/me", token, nil, "", &out); err != nil {
		return User{}, err
	}

	return out.User, nil
}

func (c *Client) UpdateProfile(ctx context.Context, token, name string) (User, error) {
	var out struct {
		OK    bool   `json:"ok"`
		User  User   `json:"user"`
		Error string `json:"error"`
	}

	if err := c.do(ctx, http.MethodPatch, "/api/auth/profile", token, map[string]any{"name": name}, "", &out); err != nil {
		return User{}, err
	}

	return out.User, nil
}

func (c *Client) Logout(ctx context.Context, token, reason string) error {
	body := map[string]any{}
	if reason != "" {
		body["reason"] = reason
	}

	return c.do(ctx, http.MethodPost, "/api/auth/logout", token, body, "", nil)
}

func (c *Client) Sessions(ctx context.Context, token string) ([]Session, error) {
	var out struct {
		OK       bool      `json:"ok"`
		Sessions []Session `json:"sessions"`
		Error    string    `json:"error"`
	}

	if err := c.do(ctx, http.MethodGet, "/api/auth/sessions", token, nil, "", &out); err != nil {
		return nil, err
	}

	return out.Sessions, nil
}

func (c *Client) RevokeSession(ctx context.Context, token, sessionID string) error {
	return c.do(ctx, http.MethodDelete, "/api/auth/sessions/"+sessionID, token, nil, "", nil)
}

func (c *Client) do(ctx context.Context, method, path, token string, body any, clientIP string, out any) error {
	var reader io.Reader
	if body != nil {
		payload, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reader = bytes.NewReader(payload)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, reader)
	if err != nil {
		return err
	}

	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	if clientIP != "" {
		// PayTrackify trusts this header only when it is configured to trust the
		// Crew host, so its per-IP limits see the real visitor.
		req.Header.Set("X-Forwarded-For", clientIP)
		req.Header.Set("X-Forwarded-Proto", "https")
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return &Error{Status: http.StatusBadGateway, Message: "We couldn't reach the sign-in service. Please try again."}
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return err
	}

	if resp.StatusCode >= 400 {
		var apiErr struct {
			Error      string `json:"error"`
			Message    string `json:"message"`
			RetryAfter int    `json:"retry_after"`
		}
		_ = json.Unmarshal(raw, &apiErr)

		message := apiErr.Error
		if message == "" {
			message = apiErr.Message
		}
		if message == "" && resp.StatusCode == http.StatusUnprocessableEntity {
			message = "That code is not right, or it has expired."
		}

		return &Error{Status: resp.StatusCode, Message: message, RetryAfter: apiErr.RetryAfter}
	}

	if out == nil {
		return nil
	}

	if err := json.Unmarshal(raw, out); err != nil {
		return fmt.Errorf("decode paytrackify response: %w", err)
	}

	return nil
}
