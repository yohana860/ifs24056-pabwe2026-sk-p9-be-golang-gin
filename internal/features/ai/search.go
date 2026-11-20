package ai

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

// Kesalahan integrasi Search Engine.
var (
	ErrSearchNotConfigured = errors.New("search engine belum dikonfigurasi")
	ErrSearchFailed        = errors.New("search engine gagal merespons")
	ErrSearchEmpty         = errors.New("search engine tidak mengembalikan sumber yang valid")
)

const (
	maxSources        = 5
	maxSnippetRunes   = 500
	maxSearchBodySize = 1 << 20
)

// SearchResult adalah satu sumber rujukan hasil penelusuran (nyata, bukan buatan AI).
type SearchResult struct {
	Title   string `json:"title"`
	URL     string `json:"url"`
	Snippet string `json:"snippet"`
}

// Searcher adalah kontrak Search Engine Service (dapat di-mock pada test).
type Searcher interface {
	Search(ctx context.Context, query string) ([]SearchResult, error)
}

// SearchClient memanggil Search Engine endpoint melalui HTTP GET ?q=<topik>.
type SearchClient struct {
	endpoint string
	key      string
	http     HTTPDoer
}

// NewSearchClient membuat SearchClient; doer nil memakai http.Client dengan timeout 15 detik.
func NewSearchClient(endpoint, key string, doer HTTPDoer) *SearchClient {
	if doer == nil {
		doer = &http.Client{Timeout: defaultSearchTimeout}
	}
	return &SearchClient{endpoint: endpoint, key: key, http: doer}
}

type searchItem struct {
	Title       string `json:"title"`
	URL         string `json:"url"`
	Link        string `json:"link"`
	Snippet     string `json:"snippet"`
	Content     string `json:"content"`
	Description string `json:"description"`
}

func (i searchItem) address() string {
	if i.URL != "" {
		return i.URL
	}
	return i.Link
}

func (i searchItem) text() string {
	for _, v := range []string{i.Snippet, i.Content, i.Description} {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

// decodeItems menerima bentuk array langsung, {"results":[...]}, atau {"data":[...]}.
func decodeItems(raw []byte) ([]searchItem, error) {
	var arr []searchItem
	if err := json.Unmarshal(raw, &arr); err == nil {
		return arr, nil
	}
	var obj struct {
		Results []searchItem `json:"results"`
		Data    []searchItem `json:"data"`
	}
	if err := json.Unmarshal(raw, &obj); err != nil {
		return nil, err
	}
	return append(obj.Results, obj.Data...), nil
}

func validURL(raw string) bool {
	u, err := url.Parse(raw)
	return err == nil && (u.Scheme == "http" || u.Scheme == "https") && u.Host != ""
}

// Search mengambil sumber rujukan; hanya hasil dengan URL http(s) valid yang dipakai.
func (c *SearchClient) Search(ctx context.Context, query string) ([]SearchResult, error) {
	if c.endpoint == "" {
		return nil, ErrSearchNotConfigured
	}
	u, err := url.Parse(c.endpoint)
	if err != nil {
		return nil, err
	}
	q := u.Query()
	q.Set("q", query)
	u.RawQuery = q.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	if c.key != "" {
		req.Header.Set("Authorization", "Bearer "+c.key)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrSearchFailed, err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, maxSearchBodySize))
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return nil, fmt.Errorf("%w: status %d", ErrSearchFailed, resp.StatusCode)
	}
	items, err := decodeItems(raw)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrSearchFailed, err)
	}
	results := []SearchResult{}
	for _, it := range items {
		title, addr := strings.TrimSpace(it.Title), strings.TrimSpace(it.address())
		if title == "" || !validURL(addr) {
			continue
		}
		results = append(results, SearchResult{Title: truncate(title, maxTitleRunes), URL: addr, Snippet: truncate(it.text(), maxSnippetRunes)})
		if len(results) == maxSources {
			break
		}
	}
	if len(results) == 0 {
		return nil, ErrSearchEmpty
	}
	return results, nil
}
