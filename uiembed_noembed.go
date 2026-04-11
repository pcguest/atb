//go:build noembed

package atbembed

import "embed"

// WebOutFS is a zero-value stub used when building without embedded web UI assets.
var WebOutFS embed.FS
