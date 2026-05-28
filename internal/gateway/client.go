package gateway

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Client talks to the Bell API gateway.
type Client struct {
	BaseURL      string
	AccessToken  string
	RefreshToken string
	HTTP         *http.Client
}

// NewClient creates a gateway client.
func NewClient(baseURL string) *Client {
	return &Client{
		BaseURL: strings.TrimRight(baseURL, "/"),
		HTTP:    &http.Client{Timeout: 30 * time.Second},
	}
}

func (c *Client) do(method, path string, body any, auth bool) ([]byte, int, error) {
	var reader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, 0, err
		}
		reader = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, c.BaseURL+path, reader)
	if err != nil {
		return nil, 0, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if auth && c.AccessToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.AccessToken)
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, err
	}
	return data, resp.StatusCode, nil
}

// LoginResponse is returned from POST /auth/login.
type LoginResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"`
	User         struct {
		UserID string `json:"user_id"`
		Name   string `json:"name"`
		Phone  string `json:"phone"`
		Role   string `json:"role"`
	} `json:"user"`
}

// Login authenticates an admin operator.
func (c *Client) Login(phone, password string) (*LoginResponse, error) {
	data, status, err := c.do(http.MethodPost, "/auth/login", map[string]string{
		"phone":    phone,
		"password": password,
	}, false)
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, parseError(status, data)
	}
	var resp LoginResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, err
	}
	c.AccessToken = resp.AccessToken
	c.RefreshToken = resp.RefreshToken
	return &resp, nil
}

func parseError(status int, data []byte) error {
	var errResp struct {
		Error   string `json:"error"`
		Message string `json:"message"`
	}
	_ = json.Unmarshal(data, &errResp)
	msg := errResp.Message
	if msg == "" {
		msg = strings.TrimSpace(string(data))
	}
	if msg == "" {
		msg = fmt.Sprintf("HTTP %d", status)
	}
	return fmt.Errorf("%s", msg)
}
