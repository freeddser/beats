package submitter

import "time"

type Data struct {
	Timestamp time.Time `json:"@timestamp"`
	Metadata  struct {
		Beat    string `json:"beat"`
		Type    string `json:"type"`
		Version string `json:"version"`
	} `json:"@metadata"`
	Host struct {
		Mac          []string `json:"mac"`
		Name         string   `json:"name"`
		Hostname     string   `json:"hostname"`
		Architecture string   `json:"architecture"`
		Os           struct {
			Type     string `json:"type"`
			Platform string `json:"platform"`
			Version  string `json:"version"`
			Family   string `json:"family"`
			Name     string `json:"name"`
			Kernel   string `json:"kernel"`
			Build    string `json:"build"`
		} `json:"os"`
		ID string   `json:"id"`
		IP []string `json:"ip"`
	} `json:"host"`
	Agent struct {
		EphemeralID string `json:"ephemeral_id"`
		ID          string `json:"id"`
		Name        string `json:"name"`
		Type        string `json:"type"`
		Version     string `json:"version"`
	} `json:"agent"`
	Message string `json:"message"`
	Log     struct {
		Offset int `json:"offset"`
		File   struct {
			Path string `json:"path"`
		} `json:"file"`
	} `json:"log"`
	Input struct {
		Type string `json:"type"`
	} `json:"input"`
	Ecs struct {
		Version string `json:"version"`
	} `json:"ecs"`
}
