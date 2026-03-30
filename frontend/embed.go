package frontendassets

import "embed"

// DistFS contains the built Vite assets served by the embedded Openclaw UI.
//
//go:embed all:dist
var DistFS embed.FS
