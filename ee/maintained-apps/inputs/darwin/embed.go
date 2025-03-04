package darwin

import (
	"embed"
)

//go:embed *.json
var AppsJSON embed.FS
