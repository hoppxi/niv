package config

import (
	"embed"
)

//go:embed all:assets/*
//go:embed all:src/*
//go:embed eww.yuck eww.scss
var embeddedFiles embed.FS

func ConfigFS() embed.FS {
	return embeddedFiles
}
