package models

import "github.com/pocketbase/pocketbase/models"

func NewRecord(collection *models.Collection) *models.Record {
	return models.NewRecord(collection)
}

type NucleiFinding struct {
	TemplateID       string   `json:"template-id"`
	TemplatePath     string   `json:"template-path"`
	Info             Info     `json:"info"`
	Type             string   `json:"type"`
	Host             string   `json:"host"`
	Port             string   `json:"port"`
	Scheme           string   `json:"scheme"`
	URL              string   `json:"url"`
	MatchedAt        string   `json:"matched-at"`
	MatcherName      string   `json:"matcher-name"`
	ExtractedResults []string `json:"extracted-results"`
	Request          string   `json:"request"`
	Response         string   `json:"response"`
	IP               string   `json:"ip"`
	Timestamp        string   `json:"timestamp"`
	CurlCommand      string   `json:"curl-command"`
	MatcherStatus    bool     `json:"matcher-status"`
}

type Info struct {
	Name           string         `json:"name"`
	Author         []string       `json:"author"`
	Tags           []string       `json:"tags"`
	Description    string         `json:"description"`
	Reference      []string       `json:"reference"`
	Severity       string         `json:"severity"`
	Metadata       Metadata       `json:"metadata"`
	Classification Classification `json:"classification"`
}

type Metadata struct {
	MaxRequest int `json:"max-request"`
}

type Classification struct {
	CveID       interface{} `json:"cve-id"`
	CweID       []string    `json:"cwe-id"`
	CvssMetrics string      `json:"cvss-metrics"`
}
