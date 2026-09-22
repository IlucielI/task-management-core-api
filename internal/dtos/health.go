package dtos

// HealthServices represents the operational status of sub-services.
type HealthServices struct {
	Database string `json:"database"`
	Redis    string `json:"redis"`
}

// HealthData contains runtime health check details.
type HealthData struct {
	Version  string         `json:"version"`
	GitHash  string         `json:"git_hash"`
	Uptime   string         `json:"uptime"`
	Services HealthServices `json:"services"`
}
