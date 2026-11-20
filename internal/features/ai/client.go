// Package ai berisi klien OpenAI Chat Completions dan layanan AI Task Planner.
package ai

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

// Kesalahan integrasi AI.
var (
	ErrNotConfigured   = errors.New("layanan AI belum dikonfigurasi")
	ErrUpstream        = errors.New("layanan AI gagal merespons")
	ErrInvalidResponse = errors.New("respons layanan AI tidak dapat dibaca")
	ErrEmptyResponse   = errors.New("layanan AI mengembalikan jawaban kosong")
	ErrInvalidAI       = errors.New("hasil AI tidak valid")
)

// Message adalah satu pesan percakapan.
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ChatClient adalah kontrak klien chat completions (dapat di-mock pada test).
type ChatClient interface {
	Chat(ctx context.Context, messages []Message) (string, error)
}

// HTTPDoer adalah kontrak HTTP client (dapat di-mock pada test).
type HTTPDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

// Client memanggil OpenAI-compatible Chat Completions. API key hanya berada di backend.
type Client struct {
	endpoint string
	key      string
	model    string
	http     HTTPDoer
}

const (
	defaultChatTimeout   = 45 * time.Second
	defaultSearchTimeout = 15 * time.Second
)

// NewClient membuat Client; doer nil memakai http.Client dengan timeout 45 detik.
func NewClient(endpoint, key, model string, doer HTTPDoer) *Client {
	if doer == nil {
		doer = &http.Client{Timeout: defaultChatTimeout}
	}
	return &Client{endpoint: endpoint, key: key, model: model, http: doer}
}

type chatRequest struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
}

type chatResponse struct {
	Choices []struct {
		Message Message `json:"message"`
	} `json:"choices"`
}

// Chat mengirim percakapan dan mengembalikan isi jawaban pertama.
func (c *Client) Chat(ctx context.Context, messages []Message) (string, error) {
	if c.key == "" || c.endpoint == "" {
		return "", ErrNotConfigured
	}
	body, _ := json.Marshal(chatRequest{Model: c.model, Messages: messages})
	url := strings.TrimRight(c.endpoint, "/") + "/chat/completions"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.key)
	resp, err := c.http.Do(req)
	if err != nil {
		return "", fmt.Errorf("%w: %w", ErrUpstream, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 1<<20))
		return "", fmt.Errorf("%w: status %d", ErrUpstream, resp.StatusCode)
	}
	var out chatResponse
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&out); err != nil {
		return "", fmt.Errorf("%w: %v", ErrInvalidResponse, err)
	}
	if len(out.Choices) == 0 || strings.TrimSpace(out.Choices[0].Message.Content) == "" {
		return "", ErrEmptyResponse
	}
	return out.Choices[0].Message.Content, nil
}
