package dto

// MusicSubmitReq is the OpenAI-compatible music generation request
// sent by clients like lurus-creator.
type MusicSubmitReq struct {
	Model        string `json:"model"`
	Prompt       string `json:"prompt"`
	Style        string `json:"style,omitempty"`
	Duration     int    `json:"duration,omitempty"`
	Instrumental bool   `json:"instrumental,omitempty"`
}

// MusicFetchResponse is the standardized response for music task polling.
type MusicFetchResponse struct {
	ID       string  `json:"id"`
	Status   string  `json:"status"`   // queued, in_progress, completed, failed
	Progress int     `json:"progress"` // 0-100
	AudioURL string  `json:"audio_url,omitempty"`
	Title    string  `json:"title,omitempty"`
	Duration float64 `json:"duration,omitempty"`
	Error    *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}
