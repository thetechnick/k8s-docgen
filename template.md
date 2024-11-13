{{define "typelink" -}}
  {{$simpleTypes := list "string" "int" "int64" "uint64" "[]string" "[]int" "bool"}}
  {{- if or (contains "." .) (has . $simpleTypes) -}}
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

{{if .ExampleYaml}}
**Example**

```yaml
{{.ExampleYaml}}
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

## {{.GroupVersion}}

{{.Doc.Sanitized}}

{{range .CRs -}}
* [{{.Kind}}](#{{.Kind|lower}})
{{end}}

{{range .CRs -}}
{{template "cr" .}}
{{end}}

---

{{range .SubObjects -}}
{{template "subobject" .}}
{{end}}
