package domain

type Role string

const (
	System Role = "system"
	User   Role = "user"
	Model  Role = "model"
	Tool   Role = "tool"
)

type Message struct {
	Metadata map[string]any `json:"metadata,omitempty"`
	Content  any            `json:"content"`
	Role     Role           `json:"role,omitempty"`
}
