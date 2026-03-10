package handlers

import "html/template"

func TemplateFunctions() template.FuncMap {
	return template.FuncMap{
		"add": func(a, b int) int {
			return a + b
		},
	}
}
