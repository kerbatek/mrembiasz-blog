package analytics

const (
	PostViewEventType  = "post_view"
	PostSlugAttribute  = "post_slug"
	ViewCountAttribute = "view_count"
)

type Event struct {
	EventType  string `json:"event_type"`
	PostSlug   string `json:"post_slug"`
	ReceivedAt string `json:"received_at"`
	ClientIP   string `json:"client_ip,omitempty"`
	UserAgent  string `json:"user_agent,omitempty"`
	Referer    string `json:"referer,omitempty"`
}
