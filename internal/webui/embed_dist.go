//go:build frontend_dist

package webui

import "embed"

//go:embed dist
var FS embed.FS
