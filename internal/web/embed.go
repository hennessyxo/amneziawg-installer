// Package web holds the embedded HTML templates and static assets for the
// panel, so the compiled binary is fully self-contained (no files to deploy).
package web

import "embed"

// Templates contains the HTML templates.
//
//go:embed templates/*.html
var Templates embed.FS

// Static contains client-side assets (htmx, CSS).
//
//go:embed static/*
var Static embed.FS
