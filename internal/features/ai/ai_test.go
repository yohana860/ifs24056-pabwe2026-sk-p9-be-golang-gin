package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeDoer struct {
	status int
	body   string
	err    error
	last   *http.Request
}

func (f *fakeDoer) Do(req *http.Request) (*http.Response, error) {
	f.last = req
	if f.err != nil {
		return nil, f.err
	}
	return &http.Response{StatusCode: f.status, Body: io.NopCloser(strings.NewReader(f.body))}, nil
}

func completion(content string) string {
	b, _ := json.Marshal(map[string]any{"choices": []any{map[string]any{"message": map[string]string{"role": "assistant", "content": content}}}})
	return string(b)
}

func TestClientChat(t *testing.T) {
	ctx := context.Background()
	msgs := []Message{{Role: "user", Content: "hi"}}

	_, err := NewClient("", "k", "m", &fakeDoer{}).Chat(ctx, msgs)
	assert.ErrorIs(t, err, ErrNotConfigured)
	_, err = NewClient("http://x", "", "m", &fakeDoer{}).Chat(ctx, msgs)
	assert.ErrorIs(t, err, ErrNotConfigured)

	_, err = NewClient("http://bad host/", "k", "m", &fakeDoer{}).Chat(ctx, msgs)
	assert.Error(t, err)
	assert.NotErrorIs(t, err, ErrUpstream)

	d := &fakeDoer{err: context.DeadlineExceeded}
	_, err = NewClient("http://x", "k", "m", d).Chat(ctx, msgs)
	assert.ErrorIs(t, err, ErrUpstream)
	assert.ErrorIs(t, err, context.DeadlineExceeded)

	_, err = NewClient("http://x", "k", "m", &fakeDoer{status: 500, body: "oops"}).Chat(ctx, msgs)
	assert.ErrorIs(t, err, ErrUpstream)
	_, err = NewClient("http://x", "k", "m", &fakeDoer{status: 200, body: "{bukan json"}).Chat(ctx, msgs)
	assert.ErrorIs(t, err, ErrInvalidResponse)
	_, err = NewClient("http://x", "k", "m", &fakeDoer{status: 200, body: `{"choices":[]}`}).Chat(ctx, msgs)
	assert.ErrorIs(t, err, ErrEmptyResponse)
	_, err = NewClient("http://x", "k", "m", &fakeDoer{status: 200, body: completion("   ")}).Chat(ctx, msgs)
	assert.ErrorIs(t, err, ErrEmptyResponse)

	d = &fakeDoer{status: 200, body: completion("halo")}
	out, err := NewClient("http://x/v1/", "rahasia", "model-x", d).Chat(ctx, msgs)
	require.NoError(t, err)
	assert.Equal(t, "halo", out)
	assert.Equal(t, "http://x/v1/chat/completions", d.last.URL.String())
	assert.Equal(t, "Bearer rahasia", d.last.Header.Get("Authorization"))
	var sent chatRequest
	raw, _ := io.ReadAll(d.last.Body)
	require.NoError(t, json.Unmarshal(raw, &sent))
	assert.Equal(t, "model-x", sent.Model)
}

func TestNewClientDefaultHTTPClient(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/chat/completions", r.URL.Path)
		_, _ = w.Write([]byte(completion("dari server")))
	}))
	defer srv.Close()
	out, err := NewClient(srv.URL, "k", "m", nil).Chat(context.Background(), nil)
	require.NoError(t, err)
	assert.Equal(t, "dari server", out)
}

const goodDraft = `{"title":"Go Itu Asyik","category":"Teknologi","excerpt":"Ringkas","content":"# Halo\n\nIsi [1]."}`

func TestSearchClient(t *testing.T) {
	ctx := context.Background()
	_, err := NewSearchClient("", "k", &fakeDoer{}).Search(ctx, "x")
	assert.ErrorIs(t, err, ErrSearchNotConfigured)
	_, err = NewSearchClient("http://[::1", "k", &fakeDoer{}).Search(ctx, "x")
	assert.Error(t, err)
	_, err = NewSearchClient("http://bad host/", "k", &fakeDoer{}).Search(ctx, "x")
	assert.Error(t, err)
	var nilCtx context.Context // NewRequestWithContext menolak context nil
	_, err = NewSearchClient("http://s", "k", &fakeDoer{}).Search(nilCtx, "x")
	assert.Error(t, err)
	assert.NotErrorIs(t, err, ErrSearchFailed)

	d := &fakeDoer{err: context.DeadlineExceeded}
	_, err = NewSearchClient("http://s", "k", d).Search(ctx, "x")
	assert.ErrorIs(t, err, ErrSearchFailed)
	assert.ErrorIs(t, err, context.DeadlineExceeded)
	_, err = NewSearchClient("http://s", "k", &fakeDoer{status: 500}).Search(ctx, "x")
	assert.ErrorIs(t, err, ErrSearchFailed)
	_, err = NewSearchClient("http://s", "k", &fakeDoer{status: 200, body: "<html>"}).Search(ctx, "x")
	assert.ErrorIs(t, err, ErrSearchFailed)
	_, err = NewSearchClient("http://s", "k", &fakeDoer{status: 200, body: `{"results":[{"title":"","url":"https://a.id"},{"title":"x","url":"javascript:alert(1)"},{"title":"y","url":"ftp://a"},{"title":"z","url":"/relatif"}]}`}).Search(ctx, "x")
	assert.ErrorIs(t, err, ErrSearchEmpty)

	// bentuk {"results":[...]} dengan field alternatif, dibatasi 5 sumber.
	items := ""
	for i := 0; i < 7; i++ {
		items += `{"title":"T` + string(rune('A'+i)) + `","link":"https://site.test/` + string(rune('a'+i)) + `","description":"ringkasan"},`
	}
	d = &fakeDoer{status: 200, body: `{"results":[` + strings.TrimSuffix(items, ",") + `]}`}
	res, err := NewSearchClient("http://s/search?lang=id", "kunci", d).Search(ctx, "belajar go")
	require.NoError(t, err)
	assert.Len(t, res, 5)
	assert.Equal(t, "https://site.test/a", res[0].URL)
	assert.Equal(t, "ringkasan", res[0].Snippet)
	assert.Equal(t, "belajar go", d.last.URL.Query().Get("q"))
	assert.Equal(t, "id", d.last.URL.Query().Get("lang"))
	assert.Equal(t, "Bearer kunci", d.last.Header.Get("Authorization"))

	// bentuk array langsung, {"data":[...]}, tanpa kunci, snippet dari content / snippet.
	d = &fakeDoer{status: 200, body: `[{"title":"A","url":"https://a.test","snippet":"s1"},{"title":"B","url":"http://b.test","content":"c2"},{"title":"C","url":"https://c.test"}]`}
	res, err = NewSearchClient("http://s", "", d).Search(ctx, "x")
	require.NoError(t, err)
	assert.Equal(t, []string{"s1", "c2", ""}, []string{res[0].Snippet, res[1].Snippet, res[2].Snippet})
	assert.Empty(t, d.last.Header.Get("Authorization"))
	res, err = NewSearchClient("http://s", "", &fakeDoer{status: 200, body: `{"data":[{"title":"D","url":"https://d.test"}]}`}).Search(ctx, "x")
	require.NoError(t, err)
	assert.Len(t, res, 1)
}

func TestNewSearchClientDefault(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`[{"title":"A","url":"https://a.test","snippet":"s"}]`))
	}))
	defer srv.Close()
	res, err := NewSearchClient(srv.URL, "", nil).Search(context.Background(), "x")
	require.NoError(t, err)
	assert.Len(t, res, 1)
}

type fakeChat struct {
	out string
	err error
	got []Message
}

func (f *fakeChat) Chat(_ context.Context, m []Message) (string, error) {
	f.got = m
	return f.out, f.err
}

type fakeSearch struct {
	res   []SearchResult
	err   error
	calls int
}

func (f *fakeSearch) Search(context.Context, string) ([]SearchResult, error) {
	f.calls++
	return f.res, f.err
}

func TestServiceGenerateWithSearch(t *testing.T) {
	sources := []SearchResult{{Title: "Sumber Asli", URL: "https://asli.test/a", Snippet: "fakta"}}
	// AI mencoba menyelipkan sumber palsu; backend hanya memakai hasil Search Engine.
	forged := `{"title":"Judul","category":"Teknologi","excerpt":"e","content":"isi","sources":[{"title":"Palsu","url":"https://palsu.test"}]}`
	fc, fs := &fakeChat{out: forged}, &fakeSearch{res: sources}
	d, err := NewService(fc, fs).Generate(context.Background(), GenerateRequest{Topic: "Go 2026"})
	require.NoError(t, err)
	assert.True(t, d.Searched)
	assert.Equal(t, 1, fs.calls)
	assert.Equal(t, sources, d.Sources)
	assert.Contains(t, fc.got[1].Content, "https://asli.test/a")
	assert.Contains(t, fc.got[1].Content, "Topik: Go 2026")
}

func TestServiceGenerateWithoutSearch(t *testing.T) {
	no := false
	fc, fs := &fakeChat{out: goodDraft}, &fakeSearch{}
	d, err := NewService(fc, fs).Generate(context.Background(), GenerateRequest{Topic: "Go", IncludeWebSearch: &no, Category: "Bisnis"})
	require.NoError(t, err)
	assert.False(t, d.Searched)
	assert.Zero(t, fs.calls, "search tidak boleh dipanggil / diklaim")
	assert.Empty(t, d.Sources)
	assert.NotContains(t, fc.got[1].Content, "Sumber rujukan")
	assert.Equal(t, "Teknologi", d.Category)
}

func TestServiceGenerateErrors(t *testing.T) {
	ctx := context.Background()
	req := GenerateRequest{Topic: "Go"}
	src := []SearchResult{{Title: "a", URL: "https://a.test"}}

	_, err := NewService(&fakeChat{out: goodDraft}, &fakeSearch{err: ErrSearchFailed}).Generate(ctx, req)
	assert.ErrorIs(t, err, ErrSearchFailed)
	_, err = NewService(&fakeChat{err: ErrUpstream}, &fakeSearch{res: src}).Generate(ctx, req)
	assert.ErrorIs(t, err, ErrUpstream)
	_, err = NewService(&fakeChat{out: "bukan json"}, &fakeSearch{res: src}).Generate(ctx, req)
	assert.ErrorIs(t, err, ErrInvalidAI)
}

func TestParseDraft(t *testing.T) {
	d, err := parseDraft("```json\n"+goodDraft+"\n```", "")
	require.NoError(t, err)
	assert.Equal(t, "Go Itu Asyik", d.Title)

	d, err = parseDraft(`{"title":"T","category":"Ngawur","content":"c"}`, "")
	require.NoError(t, err)
	assert.Equal(t, "Umum", d.Category)
	d, err = parseDraft(`{"title":"T","category":"Ngawur","content":"c"}`, "Opini")
	require.NoError(t, err)
	assert.Equal(t, "Opini", d.Category)

	long := strings.Repeat("é", 30000)
	d, err = parseDraft(`{"title":"`+long+`","excerpt":"`+long+`","content":"`+long+`"}`, "")
	require.NoError(t, err)
	assert.Len(t, []rune(d.Title), 255)
	assert.Len(t, []rune(d.Excerpt), 500)
	assert.Len(t, []rune(d.Content), 20000)

	for _, bad := range []string{"tanpa json", "}{", `{"title": }`, `{"title":"","content":"x"}`, `{"title":"x","content":" "}`} {
		_, err := parseDraft(bad, "")
		assert.ErrorIs(t, err, ErrInvalidAI, bad)
	}
}

func TestMapError(t *testing.T) {
	cases := []struct {
		err  error
		code int
	}{
		{ErrNotConfigured, 503}, {ErrSearchNotConfigured, 503}, {context.DeadlineExceeded, 504},
		{ErrSearchFailed, 502}, {ErrSearchEmpty, 502},
		{ErrUpstream, 502}, {ErrInvalidResponse, 502}, {ErrEmptyResponse, 502}, {ErrInvalidAI, 502},
		{errors.New("lain"), 500},
	}
	for _, c := range cases {
		code, msg := mapError(c.err)
		assert.Equal(t, c.code, code)
		assert.NotEmpty(t, msg)
	}
}

func TestHandlerGenerate(t *testing.T) {
	gin.SetMode(gin.TestMode)
	post := func(chat ChatClient, s Searcher, body string) (int, map[string]any) {
		r := gin.New()
		r.POST("/g", NewHandler(NewService(chat, s)).Generate)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest("POST", "/g", bytes.NewBufferString(body)))
		var out map[string]any
		_ = json.Unmarshal(w.Body.Bytes(), &out)
		return w.Code, out
	}
	src := &fakeSearch{res: []SearchResult{{Title: "a", URL: "https://a.test"}}}
	code, out := post(&fakeChat{out: goodDraft}, src, `{"topic":"Belajar Go"}`)
	assert.Equal(t, 200, code)
	data := out["data"].(map[string]any)
	assert.Equal(t, "Go Itu Asyik", data["title"])
	assert.Len(t, data["sources"], 1)
	assert.Equal(t, true, data["searched"])

	code, _ = post(&fakeChat{}, src, `{"topic":"x"}`)
	assert.Equal(t, 400, code)
	code, _ = post(&fakeChat{}, src, `{"topic":"Belajar Go","category":"Ngawur"}`)
	assert.Equal(t, 400, code)
	code, out = post(&fakeChat{out: "sampah"}, src, `{"topic":"Belajar Go"}`)
	assert.Equal(t, 502, code)
	assert.Equal(t, "error", out["status"])
}
