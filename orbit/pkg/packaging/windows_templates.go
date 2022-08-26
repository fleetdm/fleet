package packaging

import (
	_ "embed"
	"text/template"
)

//go:embed wix_template.xml
var wixTemplateXML string

// Partially adapted from Launcher's wix XML in
// https://github.com/kolide/launcher/blob/master/pkg/packagekit/internal/assets/main.wxs.
var windowsWixTemplate = template.Must(template.New("").Option("missingkey=error").Parse(wixTemplateXML))
