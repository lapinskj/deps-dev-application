package depsdev

type VersionKey struct {
	System  string `json:"system"`
	Name    string `json:"name"`
	Version string `json:"version"`
}

type DependencyNode struct {
	VersionKey VersionKey `json:"versionKey"`
	Relation   string     `json:"relation"`
}

type DependencyGraph struct {
	Nodes []DependencyNode `json:"nodes"`
	Error string           `json:"error"`
}

type ProjectKey struct {
	ID string `json:"id"`
}

type RelatedProject struct {
	ProjectKey         ProjectKey `json:"projectKey"`
	RelationType       string     `json:"relationType"`
	RelationProvenance string     `json:"relationProvenance,omitempty"`
}

type PackageVersionMetadata struct {
	RelatedProjects []RelatedProject `json:"relatedProjects"`
}

type ProjectMetadata struct {
	Scorecard struct {
		OverallScore float64 `json:"overallScore"`
	} `json:"scorecard"`
}

type ScorecardInfo struct {
	SourceRepo   string
	OpenSSFScore *float64
}
