// Package data embeds the default site catalog.
package data

import _ "embed"

// SitesJSON is the default curated site catalog.
//
//go:embed sites.json
var SitesJSON []byte
