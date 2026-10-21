package webui

import "embed"

// Files contains the local management UI served by the control plane.
//
//go:embed index.html app.js style.css
var Files embed.FS
