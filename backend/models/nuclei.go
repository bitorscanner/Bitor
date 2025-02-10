package models

import "github.com/pocketbase/pocketbase/models"

// NucleiProfile represents a nuclei scanning profile
type NucleiProfile struct {
	models.BaseModel
	Name        string `db:"name" json:"name"`
	Description string `db:"description" json:"description"`
	Config      string `db:"config" json:"config"`
	Templates   string `db:"templates" json:"templates"`
	Status      string `db:"status" json:"status"`
}

// NucleiTarget represents a target for nuclei scanning
type NucleiTarget struct {
	models.BaseModel
	Name        string `db:"name" json:"name"`
	Description string `db:"description" json:"description"`
	URL         string `db:"url" json:"url"`
	ClientID    string `db:"client" json:"client"`
	Status      string `db:"status" json:"status"`
}

// NucleiInteract represents an interaction during nuclei scanning
type NucleiInteract struct {
	models.BaseModel
	Name        string `db:"name" json:"name"`
	Description string `db:"description" json:"description"`
	ScanID      string `db:"scan" json:"scan"`
	Request     string `db:"request" json:"request"`
	Response    string `db:"response" json:"response"`
	Status      string `db:"status" json:"status"`
}
