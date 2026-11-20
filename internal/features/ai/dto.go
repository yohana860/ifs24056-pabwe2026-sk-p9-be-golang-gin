package ai

// GenerateRequest adalah payload AI Blog Generator.
type GenerateRequest struct {
	Topic            string `json:"topic" binding:"required,min=3,max=300"`
	Category         string `json:"category" binding:"omitempty,oneof='Teknologi' 'Pendidikan' 'Gaya Hidup' 'Bisnis' 'Opini' 'Umum'"`
	IncludeWebSearch *bool  `json:"include_web_search"`
}

// BlogDraft adalah draf artikel hasil AI yang sudah divalidasi, beserta sumber rujukan nyata.
type BlogDraft struct {
	Title    string         `json:"title"`
	Category string         `json:"category"`
	Excerpt  string         `json:"excerpt"`
	Content  string         `json:"content"`
	Sources  []SearchResult `json:"sources"`
	Searched bool           `json:"searched"`
}
