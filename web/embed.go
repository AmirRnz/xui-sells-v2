package web

import "embed"

// DistFS contains the embedded compiled React web panel.
//
//go:embed all:dist
var DistFS embed.FS
