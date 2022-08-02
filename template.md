{{define "typelink" -}}
  {{- if contains "." . -}}
    {{.}}
  {{- else -}}
    <a href="#{{. | replace "[]" "" | lower}}">{{.}}</a>
  {{- end}}
{{- end}}

{{define "fields" -}}
| Field | Description |
| ----- | ----------- |
{{range .Fields -}}
| `{{.Name}}` {{if .IsRequired}}<b>required</b>{{end}}<br>{{template "typelink" .Type}} | {{.Doc.Sanitized | replace "\n" "<br>"}} |
{{end -}}
{{end}}

{{define "cr" -}}
### {{.Kind}}

{{.Doc.Sanitized}}

{{if .Example}}
**Example**

```yaml
{{.Example}}
```
{{end}}

{{template "fields" .}}
{{end}}

{{define "subobject" -}}
### {{.Name}}

{{.Doc.Sanitized}}

{{template "fields" .}}

Used in:
{{range .Parents -}}
* [{{.}}](#{{. | lower}})
{{end}}
{{end}}


## `{{.GroupVersion}}`

{{.Doc.Sanitized}}

{{range .CRs -}}
* [{{.Kind}}](#{{.Kind|lower}})
{{end}}

{{range .CRs -}}
{{template "cr" .}}
{{end}}

{{range .SubObjects -}}
{{template "subobject" .}}
{{end}}
