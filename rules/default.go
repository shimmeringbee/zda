package rules

import "embed"

//go:embed *.yaml
var Embedded embed.FS
