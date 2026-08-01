package scan

type MusicItem struct {
	Name   string `json:"name"`
	Title  string `json:"title,omitempty"`
	Artist string `json:"artist,omitempty"`
	Album  string `json:"album,omitempty"`
	Length int    `json:"length,omitempty"`
	URL    string `json:"url"`
	LRC    string `json:"lrc,omitempty"`
	Cover  string `json:"cover,omitempty"`
}
