// Package testfmt provides helpers for formatting test output.
package testfmt

import (
	"embed"
	"strings"
	"text/template"
)

//go:embed template/*.tmpl
var templateFS embed.FS
var tmpl = template.Must(template.ParseFS(templateFS, "template/*.tmpl"))

func Compare[T any](expected T, actual T) string {
	s := new(strings.Builder)
	err := tmpl.ExecuteTemplate(s, "diff.tmpl", struct {
		Expected T
		Actual   T
	}{
		Expected: expected,
		Actual:   actual,
	})
	if err != nil {
		panic(err)
	}

	return s.String()
}
