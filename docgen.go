package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"go/ast"
	"go/doc"
	"go/parser"
	"go/token"
	"io"
	"io/ioutil"
	"os"
	"path"
	"reflect"
	"strings"
	"text/template"

	sprig "github.com/Masterminds/sprig/v3"
	"sigs.k8s.io/yaml"

	"k8s.io/apimachinery/pkg/runtime/schema"
)

func main() {
	templateFile := flag.String("template", "", "Go template for the documentation.")
	flag.Parse()

	docPkg, err := ParseDocPackage(flag.Arg(0))
	if err != nil {
		panic(err)
	}

	apiGroup, err := APIGroupFromDocPackage(docPkg)
	if err != nil {
		panic(err)
	}

	t := template.New("doc").Funcs(sprig.TxtFuncMap())
	if templateFile != nil && *templateFile != "" {
		c, err := ioutil.ReadFile(*templateFile)
		if err != nil {
			panic(fmt.Errorf("read file: %w", err))
		}

		if t, err = t.Parse(string(c)); err != nil {
			panic(err)
		}
	} else {
		// default template
		panic("no template")
	}

	var buf bytes.Buffer
	if err := t.Execute(&buf, apiGroup); err != nil {
		panic(err)
	}
	fmt.Print(strings.TrimSpace(buf.String()))
}

func ParseDocPackage(folder string) (*doc.Package, error) {
	fileSet := token.NewFileSet()
	files := map[string]*ast.File{}

	entries, err := os.ReadDir(folder)
	if err != nil {
		return nil, fmt.Errorf("read directory: %w", err)
	}
	for _, entry := range entries {
		if entry.IsDir() {
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
	// but we don't care for just templating documentation.
	apkg, _ := ast.NewPackage(fileSet, files, nil, nil)
	return doc.New(apkg, "", 0), nil
}

type APIGroup struct {
	schema.GroupVersion
	Doc        DocumentationBlock
	CRs        []CustomResource
	SubObjects []CustomResourceSubObject
}

// CustomResource
type CustomResource struct {
	schema.GroupVersionKind
	Doc     DocumentationBlock
	Example string
	Link    string
	Fields  []CustomResourceField
}

type CustomResourceField struct {
	Name       string
	Doc        DocumentationBlock
	Type       string
	IsRequired bool
}

type CustomResourceSubObject struct {
	// Name of the go struct
	Name               string
	Doc                DocumentationBlock
	Fields             []CustomResourceField
	EmbeddedSubObjects []string
	IsEmbedded         bool
	Link               string
	Parents            []string
}

type DocumentationBlock struct {
	// Raw documentation as written in the file.
	Raw string
	// Sanitized documentation string, where TODO and
	// code-generator comments are removed.
	Sanitized string
	// Codegen annotations, if present.
	Annotations map[string]string
}

const (
	groupNameAnnotation  = "groupName"
	objectRootAnnotation = "kubebuilder:object:root"
	scopeAnnotation      = "kubebuilder:resource:scope"
	exampleAnnotation    = "example"
	defaultAnnotation    = "kubebuilder:default"
)

func APIGroupFromDocPackage(pkg *doc.Package) (*APIGroup, error) {
	apiGroup := &APIGroup{}

	pkgDoc, err := FormatRawDoc(pkg.Doc)
	if err != nil {
		return nil, fmt.Errorf("formatting package documentation: %w", err)
	}
	apiGroup.Doc = *pkgDoc
	apiGroup.GroupVersion = schema.GroupVersion{
		Group:   pkgDoc.Annotations[groupNameAnnotation],
		Version: pkg.Name,
	}

	subObjectMap := map[string]struct{}{}
	subObjects := map[string]CustomResourceSubObject{}
	for _, t := range pkg.Types {
		typeDoc, err := FormatRawDoc(t.Doc)
		if err != nil {
			return nil, fmt.Errorf(
				"formatting type %s documentation: %w", t.Name, err)
		}

		fields, embedded, err := Fields(t)
		if err != nil {
			return nil, fmt.Errorf("reading fields for CR: %w", err)
		}

		if typeDoc.Annotations[objectRootAnnotation] == "true" {
			// Custom Resource!
			if strings.HasSuffix(t.Name, "List") {
				continue
			}

			cr := CustomResource{
				GroupVersionKind: apiGroup.WithKind(t.Name),
				Doc:              *typeDoc,
				Fields:           fields,
			}

			apiGroup.CRs = append(apiGroup.CRs, cr)
			continue
		}

		if len(fields) == 0 {
			continue
		}

		for _, e := range embedded {
			subObjectMap[e] = struct{}{}
		}

		// Some other Type
		subObjects[t.Name] = CustomResourceSubObject{
			Name:               t.Name,
			Doc:                *typeDoc,
			Fields:             fields,
			EmbeddedSubObjects: embedded,
		}
		apiGroup.SubObjects = append(apiGroup.SubObjects, subObjects[t.Name])
	}

	// Handle embedded fields and parents.
	for i, subObject := range apiGroup.SubObjects {
		if _, ok := subObjectMap[subObject.Name]; ok {
			apiGroup.SubObjects[i].IsEmbedded = true
		}

		for _, embedded := range subObject.EmbeddedSubObjects {
			embeddedObj := subObjects[embedded]
			apiGroup.SubObjects[i].Fields = append(
				apiGroup.SubObjects[i].Fields, embeddedObj.Fields...)
		}
	}

	// Add Parents
	objectsUsingType := map[string][]string{}
	for _, subObject := range apiGroup.SubObjects {
		for _, field := range subObject.Fields {
			k := strings.TrimLeft(field.Type, "[]")
			objectsUsingType[k] = append(objectsUsingType[k], subObject.Name)
		}
	}
	for _, cr := range apiGroup.CRs {
		for _, field := range cr.Fields {
			k := strings.TrimLeft(field.Type, "[]")
			objectsUsingType[k] = append(objectsUsingType[k], cr.Kind)
		}
	}
	for i := range apiGroup.SubObjects {
		apiGroup.SubObjects[i].Parents = objectsUsingType[apiGroup.SubObjects[i].Name]
	}

	// Add Examples
	for i, cr := range apiGroup.CRs {
		example, err := BuildExampleYaml(&cr, apiGroup.SubObjects)
		if err != nil {
			return nil, fmt.Errorf("building example for CR %s: %w",
				cr.GroupVersionKind.String(), err)
		}

		apiGroup.CRs[i].Example = example
	}

	return apiGroup, nil
}

func Fields(t *doc.Type) (
	fields []CustomResourceField,
	embedded []string, err error,
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
			embedded = append(embedded, fieldType)
			continue
		}

		fieldName := fieldName(field, jsonTag)
		if fieldName == "-" {
			// field excluded in JSON -> DROP
			continue
		}

		fieldDoc, err := FormatRawDoc(field.Doc.Text())
		if err != nil {
			return nil, nil, fmt.Errorf(
				"formatting field %s documentation: %w", fieldName, err)
		}

		fields = append(fields, CustomResourceField{
			Name:       fieldName,
			Doc:        *fieldDoc,
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

func FormatRawDoc(rawDoc string) (docBlock *DocumentationBlock, err error) {
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
			return nil, fmt.Errorf("reading line: %w", err)
		}
		line = strings.TrimSpace(line)

		switch {
		case strings.HasPrefix(line, "TODO:"): // Ignore TODOs
		case strings.HasPrefix(line, "todo:"): // Ignore TODOs
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

	return &DocumentationBlock{
		Raw:         rawDoc,
		Sanitized:   strings.TrimSpace(out.String()),
		Annotations: annotations,
	}, nil
}

func BuildExampleYaml(
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
	if strings.HasPrefix(fieldType, "[]") {
		// array
		return []interface{}{
			exampleFieldValue(
				fieldType[2:], annotations, subObjects),
		}
	}

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

	switch fieldType {
	case "int", "int32", "int64":
		return 42
	case "string":
		return nextWord()

	default:
		if subObject, ok := subObjects[fieldType]; ok {
			return exampleObject(subObject.Fields, subObjects)
		}
		return fieldType
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
