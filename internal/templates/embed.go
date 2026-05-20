// Package templates embeds the generator's *.tmpl assets so the goplate
// binary is fully self-contained.
package templates

import "embed"

// Feature holds every template used by `goplate new-feature`.
//
//go:embed feature/*.tmpl
var Feature embed.FS

// Adapter holds every template used by `goplate new-adapter`.
//
//go:embed adapter/*.tmpl
var Adapter embed.FS

// Event holds every template used by `goplate new-event`.
//
//go:embed event/*.tmpl
var Event embed.FS

// Project holds the embedded boilerplate that `goplate new` materializes
// into a fresh project directory. The `all:` prefix is required so dotfiles
// (.env.example, .gitignore) are included.
//
//go:embed all:project
var Project embed.FS
