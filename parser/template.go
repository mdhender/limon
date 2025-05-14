package parser

import (
	"embed"
)

//go:embed lempar.c.template
var defaultTemplate embed.FS

// GetDefaultTemplate returns the contents of the embedded lempar.c template file
func GetDefaultTemplate() (string, error) {
	data, err := defaultTemplate.ReadFile("lempar.c.template")
	if err != nil {
		return "", err
	}

	return string(data), nil
}