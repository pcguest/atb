package atbembed

import "embed"

// WebOutFS stores the exported Next.js dashboard assets.
//
//go:embed web/out
var WebOutFS embed.FS
