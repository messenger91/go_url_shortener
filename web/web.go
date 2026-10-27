// Package web embeds the UI assets served at / and /static/*.
package web

import "embed"

//go:embed all:static
var Static embed.FS
