package main

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"go/ast"
	"go/doc"
	"go/parser"
	"go/token"
	"io"
	"os"
	"path"
	"reflect"
	"strings"
	"text/template"

	sprig "github.com/Masterminds/sprig/v3"
	"sigs.k8s.io/yaml"

	"k8s.io/apimachinery/pkg/runtime/schema"
)

type Docgen struct {
	opts Options
}

type Options struct {
	TemplateFile string
}

func (opts *Options) Default() {
}

type Option interface {
	Apply(opts *Options)
}

type TemplateFile string

func (f TemplateFile) Apply(opts *Options) {
	opts.TemplateFile = string(f)
}

func NewDocgen(opts ...Option) *Docgen {
	d := &Docgen{}
	for _, opt := range opts {
		opt.Apply(&d.opts)
	}
	d.opts.Default()
	return d
}

func (d *Docgen) Parse(ctx context.Context, folder string, writer io.Writer) error {
	docPkg, err := d.parseGoPackage(ctx, folder)
	if err != nil {
		return fmt.Errorf("parsing go package: %w", err)
	}

	apiGroup, err := d.loadAPIGroup(ctx, docPkg)
	if err != nil {
		return fmt.Errorf("load API objects: %w", err)
	}

	t := template.New("doc").Funcs(sprig.TxtFuncMap())
	if d.opts.TemplateFile != "" {
		c, err := os.ReadFile(d.opts.TemplateFile)
		if err != nil {
			return fmt.Errorf("reading template file: %w", err)
		}

		if t, err = t.Parse(string(c)); err != nil {
			return fmt.Errorf("parsing template: %w", err)
		}
	} else {
		// default template
		if t, err = t.Parse(defaultTemplate); err != nil {
			return fmt.Errorf("parsing default template: %w", err)
		}
	}

	var buf bytes.Buffer
	if err := t.Execute(&buf, apiGroup); err != nil {
		return fmt.Errorf("executing template: %w", err)
	}

	fmt.Fprint(writer, strings.TrimSpace(buf.String()))
	return nil
}

func (d *Docgen) parseGoPackage(ctx context.Context, folder string) (*doc.Package, error) {
	fileSet := token.NewFileSet()
	files := map[string]*ast.File{}

	entries, err := os.ReadDir(folder)
	if err != nil {
		return nil, fmt.Errorf("read directory: %w", err)
	}
	for _, entry := range entries {
		if entry.IsDir() || path.Ext(entry.Name()) != ".go" {
			continue
		}

		filename := path.Join(folder, entry.Name())
		astFile, err := parser.ParseFile(
			fileSet, filename, nil, parser.ParseComments)
		if err != nil {
			return nil, fmt.Errorf("parse AST: %w", err)
		}
		files[filename] = astFile
	}

	// This will fail due to unresolved imports,
	// but we don't care for just generating documentation.
	apkg, _ := ast.NewPackage(fileSet, files, nil, nil)
	return doc.New(apkg, "", 0), nil
}

func (d *Docgen) loadAPIGroup(ctx context.Context, pkg *doc.Package) (*APIGroup, error) {
	apiGroup := &APIGroup{}

	pkgDoc, err := d.formatRawDoc(ctx, pkg.Doc)
	if err != nil {
		return nil, fmt.Errorf("formatting package documentation: %w", err)
	}
	apiGroup.Doc = pkgDoc
	apiGroup.GroupVersion = schema.GroupVersion{
		Group:   pkgDoc.Annotations[groupNameAnnotation],
		Version: pkg.Name,
	}
	apiGroup.CRs, apiGroup.SubObjects, err = d.loadObjects(ctx, pkg, apiGroup.GroupVersion)
	if err != nil {
		return nil, fmt.Errorf("load objects: %w", err)
	}

	return apiGroup, nil
}

func (d *Docgen) loadObjects(
	ctx context.Context, pkg *doc.Package, gv schema.GroupVersion,
) (crs []CustomResource, objs []CustomResourceSubObject, err error) {
	for _, t := range pkg.Types {
		typeDoc, err := d.formatRawDoc(ctx, t.Doc)
		if err != nil {
			return nil, nil, fmt.Errorf(
				"formatting type %s documentation: %w", t.Name, err)
		}

		fields, embeddedTypes, err := d.Fields(ctx, t)
		if err != nil {
			return nil, nil, fmt.Errorf("reading fields for CR: %w", err)
		}

		if len(fields) == 0 {
			continue
		}

		if typeDoc.Annotations[objectRootAnnotation] == "true" {
			// Custom Resource!
			if strings.HasSuffix(t.Name, "List") {
				continue
			}

			cr := CustomResource{
				GroupVersionKind: gv.WithKind(t.Name),
				Doc:              typeDoc,
				Fields:           fields,
			}
			if typeDoc.Annotations[scopeAnnotation] == "Cluster" {
				cr.Scope = CustomResourceScopeCluster
			} else {
				cr.Scope = CustomResourceScopeNamespaced
			}
			crs = append(crs, cr)
			continue
		}

		// Some other Type
		objs = append(objs, CustomResourceSubObject{
			Name:               t.Name,
			Doc:                typeDoc,
			Fields:             fields,
			EmbeddedSubObjects: embeddedTypes,
		})
	}
	d.loadEmbeddedFields(ctx, objs)
	d.loadParents(ctx, crs, objs)
	if err := d.loadExampleYaml(ctx, crs, objs); err != nil {
		return nil, nil, fmt.Errorf("loading example YAML: %w", err)
	}
	// filter embedded objects
	var filteredObjs []CustomResourceSubObject
	for _, obj := range objs {
		if obj.IsEmbedded {
			continue
		}
		filteredObjs = append(filteredObjs, obj)
	}
	d.defaultFieldDocsToTypeDocs(ctx, crs, filteredObjs)
	return crs, filteredObjs, nil
}

// Load embedded fields from other types.
func (d *Docgen) loadEmbeddedFields(
	ctx context.Context,
	objs []CustomResourceSubObject,
) {
	subObjectsByName := map[string]CustomResourceSubObject{}
	embeddedObjects := map[string]struct{}{}
	fieldObjectTypes := map[string]struct{}{}
	for _, obj := range objs {
		subObjectsByName[obj.Name] = obj
		for _, embeddedType := range obj.EmbeddedSubObjects {
			embeddedObjects[embeddedType] = struct{}{}
		}
		for _, field := range obj.Fields {
			fieldObjectTypes[strings.TrimPrefix(field.Type, "[]")] = struct{}{}
		}
	}

	for i, obj := range objs {
		_, isEmbedded := embeddedObjects[obj.Name]
		_, isReferencedByField := fieldObjectTypes[obj.Name]
		if isEmbedded && !isReferencedByField {
			// Only mark (and later filter) objects that are embedded AND
			objs[i].IsEmbedded = true
		}

		for _, embedded := range obj.EmbeddedSubObjects {
			embeddedObj := subObjectsByName[embedded]
			objs[i].Fields = append(
				objs[i].Fields, embeddedObj.Fields...)
		}
	}
}

func (d *Docgen) loadParents(
	ctx context.Context,
	crds []CustomResource,
	objs []CustomResourceSubObject,
) {
	objectsUsingType := map[string][]string{}
	for _, subObject := range objs {
		if subObject.IsEmbedded {
			continue
		}
		for _, field := range subObject.Fields {
			k := strings.TrimLeft(field.Type, "[]")
			objectsUsingType[k] = append(objectsUsingType[k], subObject.Name)
		}
	}
	for _, cr := range crds {
		for _, field := range cr.Fields {
			k := strings.TrimLeft(field.Type, "[]")
			objectsUsingType[k] = append(objectsUsingType[k], cr.Kind)
		}
	}
	for i := range objs {
		objs[i].Parents = objectsUsingType[objs[i].Name]
	}
}

func (d *Docgen) loadExampleYaml(
	ctx context.Context,
	crs []CustomResource,
	objs []CustomResourceSubObject,
) error {
	// Add Examples
	for i, cr := range crs {
		example, err := d.buildExampleYaml(&cr, objs)
		if err != nil {
			return fmt.Errorf("building example for CR %s: %w",
				cr.GroupVersionKind.String(), err)
		}

		crs[i].ExampleYaml = example
	}
	return nil
}

func (d *Docgen) defaultFieldDocsToTypeDocs(
	ctx context.Context,
	crs []CustomResource,
	objs []CustomResourceSubObject,
) {
	subObjectsByName := map[string]CustomResourceSubObject{}
	for _, obj := range objs {
		subObjectsByName[obj.Name] = obj
	}

	for crdi, obj := range crs {
		for fi, field := range obj.Fields {
			if len(field.Doc.Sanitized) > 0 {
				continue
			}

			if fieldTypeObj, ok := subObjectsByName[field.Type]; ok {
				field.Doc.Sanitized = fieldTypeObj.Doc.Sanitized
				crs[crdi].Fields[fi].Doc.Sanitized = fieldTypeObj.Doc.Sanitized
			}
		}
	}

	for oi, obj := range objs {
		for fi, field := range obj.Fields {
			if len(field.Doc.Sanitized) > 0 {
				continue
			}

			if fieldTypeObj, ok := subObjectsByName[field.Type]; ok {
				objs[oi].Fields[fi].Doc.Sanitized = fieldTypeObj.Doc.Sanitized
			}
		}
	}
}

func (d *Docgen) Fields(
	ctx context.Context, t *doc.Type,
) (
	fields []CustomResourceField,
	embeddedTypes []string, err error,
) {
	structType, ok := t.Decl.Specs[0].(*ast.TypeSpec).Type.(*ast.StructType)
	if !ok {
		// not a struct
		return nil, nil, nil
	}

	for _, field := range structType.Fields.List {
		fieldType := fieldType(field.Type)

		jsonTag := fieldJSONTag(field)
		if len(jsonTag) == 0 {
			panic(fmt.Sprintf("no json tag: %v", field))
		}
		if strings.Contains(jsonTag, "inline") {
			// embedded object
			embeddedTypes = append(embeddedTypes, fieldType)
			continue
		}

		fieldName := fieldName(field, jsonTag)
		if fieldName == "-" {
			// field excluded in JSON -> DROP
			continue
		}

		fieldDoc, err := d.formatRawDoc(ctx, field.Doc.Text())
		if err != nil {
			return nil, nil, fmt.Errorf(
				"formatting field %s documentation: %w", fieldName, err)
		}

		fields = append(fields, CustomResourceField{
			Name:       fieldName,
			Doc:        fieldDoc,
			Type:       fieldType,
			IsRequired: !strings.Contains(jsonTag, "omitempty"),
		})
	}

	return
}

func fieldType(typ ast.Expr) string {
	switch astType := typ.(type) {
	case *ast.Ident:
		return astType.Name

	case *ast.StarExpr:
		return fieldType(astType.X)

	case *ast.SelectorExpr:
		pkg := astType.X.(*ast.Ident)
		t := astType.Sel
		return pkg.Name + "." + t.Name

	case *ast.ArrayType:
		return "[]" + fieldType(astType.Elt)

	case *ast.MapType:
		return fmt.Sprintf("map[%s]%s", fieldType(astType.Key), fieldType(astType.Value))

	default:
		panic(fmt.Sprintf("unhandled field type: %v", astType))
	}
}

func fieldJSONTag(field *ast.Field) string {
	if field.Tag == nil {
		return ""
	}
	return reflect.
		// Delete first and last quotation
		StructTag(field.Tag.Value[1 : len(field.Tag.Value)-1]).
		Get("json")
}

// fieldName returns the name of the field as it should appear in JSON format
// "-" indicates that this field is not part of the JSON representation
func fieldName(field *ast.Field, jsonTag string) string {
	jsonTag = strings.Split(jsonTag, ",")[0] // This can return "-"
	if jsonTag == "" {
		if field.Names != nil {
			return field.Names[0].Name
		}
		return field.Type.(*ast.Ident).Name
	}
	return jsonTag
}

func (d *Docgen) formatRawDoc(
	ctx context.Context, rawDoc string,
) (docBlock DocumentationBlock, err error) {
	buf := bytes.NewBufferString(rawDoc)
	r := bufio.NewReader(buf)

	var (
		out         bytes.Buffer
		annotations = map[string]string{}
	)
	for {
		line, err := r.ReadString('\n')
		if err == io.EOF {
			break
		}
		if err != nil {
			return DocumentationBlock{}, fmt.Errorf("reading line: %w", err)
		}
		line = strings.TrimSpace(line)

		switch {
		case strings.HasPrefix(line, "TODO:"): // Ignore TODOs
		case strings.HasPrefix(line, "todo:"): // Ignore TODOs
		case strings.HasPrefix(line, "+example"):
			annotations[exampleAnnotation] = strings.SplitN(line, "=", 2)[1]
		case strings.HasPrefix(line, "+groupName"):
			annotations[groupNameAnnotation] = strings.SplitN(line, "=", 2)[1]
		case strings.HasPrefix(line, "+optional"):
			annotations[line] = ""
		case strings.HasPrefix(line, "+kubebuilder:default"):
			annotations[defaultAnnotation] = strings.SplitN(line, "=", 2)[1]
		case strings.HasPrefix(line, "+kubebuilder:resource"):
			line = strings.TrimPrefix(line, "+kubebuilder:resource:")
			if !strings.Contains(line, "=") {
				annotations[line] = ""
			} else if strings.HasPrefix(line, "scope") {
				parts := strings.SplitN(line, ",", 2)
				for _, arg := range parts {
					values := strings.Split(arg, "=")
					annotations[values[0]] = values[1]
				}
			} else {
				values := strings.Split(line, "=")
				annotations[values[0]] = values[1]
			}
		case strings.HasPrefix(line, "+"): // Code generator annotations
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 1 {
				annotations[parts[0][1:]] = ""
				continue
			}
			annotations[parts[0][1:]] = parts[1]

		default:
			out.WriteString(line)
			out.WriteRune('\n')
		}
	}

	return DocumentationBlock{
		Raw:         rawDoc,
		Sanitized:   strings.TrimSpace(out.String()),
		Annotations: annotations,
	}, nil
}

func (d *Docgen) buildExampleYaml(
	cr *CustomResource,
	subObjects []CustomResourceSubObject,
) (y string, err error) {
	subObjectsMap := map[string]CustomResourceSubObject{}
	for _, so := range subObjects {
		subObjectsMap[so.Name] = so
	}

	obj := exampleObject(cr.Fields, subObjectsMap)
	obj["kind"] = cr.Kind
	obj["apiVersion"] = cr.GroupVersionKind.GroupVersion().String()
	metadata := map[string]string{
		"name": "example",
	}
	if scope, ok := cr.Doc.Annotations[scopeAnnotation]; !ok || scope != "Cluster" {
		metadata["namespace"] = "default"
	}
	obj["metadata"] = metadata

	yb, err := yaml.Marshal(obj)
	y = string(yb)
	return
}

func exampleObject(
	fields []CustomResourceField,
	subObjects map[string]CustomResourceSubObject,
) map[string]interface{} {
	example := map[string]interface{}{}
	for _, field := range fields {
		example[field.Name] = exampleFieldValue(
			field.Type, field.Doc.Annotations, subObjects)
	}
	return example
}

func exampleFieldValue(
	fieldType string,
	annotations map[string]string,
	subObjects map[string]CustomResourceSubObject,
) interface{} {
	if example, ok := annotations[exampleAnnotation]; ok {
		var val interface{}
		if err := yaml.Unmarshal([]byte(example), &val); err != nil {
			return example
		}
		return val
	}
	if def, ok := annotations[defaultAnnotation]; ok {
		var val interface{}
		if err := yaml.Unmarshal([]byte(def), &val); err != nil {
			return def
		}
		return val
	}

	if strings.HasPrefix(fieldType, "[]") {
		// array
		return []interface{}{
			exampleFieldValue(
				fieldType[2:], annotations, subObjects),
		}
	}

	switch fieldType {
	case "int", "int32", "int64":
		return 42
	case "string":
		return nextWord()
	case "bool":
		return true
	default:
		if subObject, ok := subObjects[fieldType]; ok {
			return exampleObject(subObject.Fields, subObjects)
		}
		return map[string]interface{}{}
	}
}

var (
	wordCounter int
	words       = []string{
		"lorem", "ipsum", "dolor", "sit",
		"amet", "consetetur", "sadipscing",
		"elitr", "sed", "diam", "nonumy", "eirmod", "tempor",
	}
)

func nextWord() string {
	if wordCounter == len(words) {
		wordCounter = 0
	}
	word := words[wordCounter]
	wordCounter++
	return word
}
