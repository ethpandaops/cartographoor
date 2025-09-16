package validatorranges

// ValidatorRanges represents the complete validator range data for a network.
type ValidatorRanges struct {
	Nodes      map[string]*Node    `json:"nodes"`
	Validators *ValidatorSummary   `json:"validators"`
	Groups     map[string][]string `json:"groups"`
	Tags       map[string][]string `json:"tags"`
	Metadata   *Metadata           `json:"metadata"`
}

// Node represents a single node with its validator range assignments.
type Node struct {
	Groups          []string               `json:"groups"`
	Tags            []string               `json:"tags"`
	Attributes      map[string]interface{} `json:"attributes"`
	ValidatorRanges []*ValidatorRange      `json:"validatorRanges"`
	Source          string                 `json:"source"`
}

// ValidatorRange represents a range of validators assigned to a node.
type ValidatorRange struct {
	Start int `json:"start"`
	End   int `json:"end"`
}

// ValidatorSummary provides summary statistics about validators.
type ValidatorSummary struct {
	TotalCount int `json:"totalCount"`
}

// Metadata contains information about the validator ranges data.
type Metadata struct {
	NetworkName string   `json:"networkName"`
	Sources     []string `json:"sources"`
}

// Config represents the configuration for validator ranges.
type Config struct {
	AdditionalSources map[string][]SourceConfig `json:"additionalSources" mapstructure:"additionalSources"`
}

// SourceConfig represents configuration for a single validator data source.
type SourceConfig struct {
	URL         string `json:"url" mapstructure:"url"`
	Name        string `json:"name" mapstructure:"name"`
	RangeOffset int    `json:"rangeOffset" mapstructure:"rangeOffset"`
}
