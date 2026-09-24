package assets

import "embed"

//go:embed schema.sql templates static
var Files embed.FS
