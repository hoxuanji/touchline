// Package snapshotdata embeds the bundled WC2026 structural snapshot JSON.
package snapshotdata

import _ "embed"

//go:embed venues.json
var VenuesJSON []byte

//go:embed teams.json
var TeamsJSON []byte
