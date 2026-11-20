package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	"app/internal/models"
)

const (
	maxTitleRunes   = 255
	maxExcerptRune  = 500
	maxContentRunes = 20000
	defaultCategory = "Umum"
)

const systemPrompt = `Kamu adalah penulis blog profesional berbahasa Indonesia.
Tulis artikel lengkap dalam format Markdown berdasarkan topik dari pengguna.
Jika diberikan "Sumber rujukan", gunakan HANYA informasi dari sumber tersebut untuk fakta terkini dan sitasi dengan format [1], [2] sesuai nomor sumber. Jangan menambahkan URL atau sumber lain.
Balas HANYA dengan JSON valid tanpa teks lain: {"title":"...","category":"salah satu dari Teknologi|Pendidikan|Gaya Hidup|Bisnis|Opini|Umum","excerpt":"1-2 kalimat","content":"artikel markdown"}`

// Service menghasilkan draf artikel: (opsional) Search Engine -> OpenAI Chat Completions.
type Service struct {
	chat   ChatClient
	search Searcher
}

// NewService membuat Service AI.
func NewService(chat ChatClient, search Searcher) *Service {
	return &Service{chat: chat, search: search}
}

func buildPrompt(topic string, sources []SearchResult) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Topik: %s\n", topic)
	if len(sources) > 0 {
		b.WriteString("\nSumber rujukan hasil penelusuran:\n")
		for i, s := range sources {
			fmt.Fprintf(&b, "[%d] %s (%s)\n%s\n", i+1, s.Title, s.URL, s.Snippet)
		}
	}
	return b.String()
}

// Generate membuat draf artikel. Sumber rujukan selalu berasal dari Search Engine, bukan klaim AI.
func (s *Service) Generate(ctx context.Context, req GenerateRequest) (*BlogDraft, error) {
	var sources []SearchResult
	searched := req.IncludeWebSearch == nil || *req.IncludeWebSearch
	if searched {
		found, err := s.search.Search(ctx, req.Topic)
		if err != nil {
			return nil, err
		}
		sources = found
	}
	raw, err := s.chat.Chat(ctx, []Message{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: buildPrompt(req.Topic, sources)},
	})
	if err != nil {
		return nil, err
	}
	draft, err := parseDraft(raw, req.Category)
	if err != nil {
		return nil, err
	}
	draft.Sources = append([]SearchResult{}, sources...)
	draft.Searched = searched
	return draft, nil
}

func truncate(s string, n int) string {
	r := []rune(strings.TrimSpace(s))
	if len(r) > n {
		r = r[:n]
	}
	return string(r)
}

func parseDraft(raw, fallbackCategory string) (*BlogDraft, error) {
	start, end := strings.Index(raw, "{"), strings.LastIndex(raw, "}")
	if start < 0 || end <= start {
		return nil, ErrInvalidAI
	}
	var d BlogDraft
	if err := json.Unmarshal([]byte(raw[start:end+1]), &d); err != nil {
		return nil, ErrInvalidAI
	}
	d.Title = truncate(d.Title, maxTitleRunes)
	d.Excerpt = truncate(d.Excerpt, maxExcerptRune)
	d.Content = truncate(d.Content, maxContentRunes)
	if d.Title == "" || d.Content == "" {
		return nil, ErrInvalidAI
	}
	if !slices.Contains(models.Categories, d.Category) {
		d.Category = fallbackCategory
		if d.Category == "" {
			d.Category = defaultCategory
		}
	}
	return &d, nil
}
