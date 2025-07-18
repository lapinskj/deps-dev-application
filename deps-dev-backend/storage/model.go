package storage

type Dependency struct {
	System       string   `json:"system"`
	Name         string   `json:"name"`
	Version      string   `json:"version"`
	Relation     string   `json:"relation,omitempty"`
	SourceRepo   string   `json:"source_repo,omitempty"`
	OpenSSFScore *float64 `json:"openssf_score,omitempty"`
}
