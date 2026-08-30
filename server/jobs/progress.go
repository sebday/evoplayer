package jobs

type Progress struct {
	Phase  string `json:"phase,omitempty"`
	Folder string `json:"folder,omitempty"`
	Done   int    `json:"done,omitempty"`
	Total  int    `json:"total,omitempty"`
}
