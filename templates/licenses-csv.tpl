Name,License{{ range . }}
{{.Name}},{{.LicenseName}}
{{- end }}
