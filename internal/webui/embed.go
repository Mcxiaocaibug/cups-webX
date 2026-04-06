//go:build !frontend_dist

package webui

import "embed"

//go:embed fallback
var FS embed.FS
