package htmlparse

// ParsedChapterContent represents structured content parsed from HTML
type ParsedChapterContent struct {
	Title    string           `json:"title"`
	Sections []ContentSection `json:"sections"`
}

// ContentSection represents a section of content with heading and content
type ContentSection struct {
	Heading ContentHeading `json:"heading"`
	Content []any          `json:"content"`
}

// ContentHeading represents a heading element
type ContentHeading struct {
	Level int    `json:"level"`
	Text  string `json:"text"`
	ID    string `json:"id,omitempty"`
}

// ParagraphElement represents a paragraph in section content
type ParagraphElement struct {
	Type string `json:"type"` // "paragraph"
	Text string `json:"text"`
}

// CodeBlockElement represents a code block in section content
type CodeBlockElement struct {
	Type     string `json:"type"` // "code_block"
	Language string `json:"language,omitempty"`
	Code     string `json:"code"`
	Caption  string `json:"caption,omitempty"`
}

// ImageElement represents an image in section content
type ImageElement struct {
	Type    string `json:"type"` // "image"
	Src     string `json:"src"`
	Alt     string `json:"alt,omitempty"`
	Caption string `json:"caption,omitempty"`
}

// ListElement represents a list in section content
type ListElement struct {
	Type    string   `json:"type"` // "list"
	Ordered bool     `json:"ordered"`
	Items   []string `json:"items"`
}

// LinkElement represents a standalone link in section content
type LinkElement struct {
	Type     string `json:"type"` // "link"
	Href     string `json:"href"`
	Text     string `json:"text"`
	LinkType string `json:"link_type"` // "external" / "internal" / "anchor"
}
