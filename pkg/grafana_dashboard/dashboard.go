package grafana_dashboard

// GrafanaDashboards represent the layout of a dashboard in grafana. When `json.Marshall`ed, Grafana
// can load them from disk to populate its visualizations.
type GrafanaDashboard struct {
	Title       string             `json:"title"`
	Rows        []GrafanaRow       `json:"rows"`
	Annotations GrafanaAnnotations `json:"annotations"`
}

type GrafanaAnnotations struct {
	List []GrafanaAnnotation `json:"list"`
}

type GrafanaAnnotation struct {
	Enable      bool   `json:"enable"`
	Expr        string `json:"expr"`
	TagKeys     string `json:"tagKeys"`
	TitleFormat string `json:"titleFormat"`
	IconColor   string `json:"iconColor"`
}

type GrafanaRow struct {
	Title  string         `json:"title"`
	Panels []GrafanaPanel `json:"panels"`
}

type GrafanaPanel struct {
	Title       string             `json:"title"`
	Type        string             `json:"type"`
	Description string             `json:"description,omitempty"`
	Targets     []GrafanaTarget    `json:"targets"`
	Thresholds  []GrafanaThreshold `json:"thresholds"`
}

type GrafanaTarget struct {
	Expression string `json:"expr"`
}

type GrafanaThreshold struct {
	Value     float64 `json:"value"`
	ColorMode string  `json:"colorMode"`
	Op        string  `json:"op"`
	Fill      bool    `json:"fill"`
	Line      bool    `json:"line"`
	Yaxis     string  `json:"yaxis"`
}
