package main

import (
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// APIGroup represents a versioned API group.
type APIGroup struct {
	schema.GroupVersion
	// Package documentation block.
	Doc DocumentationBlock
	// CustomResources found in the API group.
	CRs []CustomResource
	// All other objects that are part of the API group.
	SubObjects []CustomResourceSubObject
}

// CustomResource represents a top-level API resource.
type CustomResource struct {
	schema.GroupVersionKind

	// Struct documentation block.
	Doc DocumentationBlock
	// Example object rendered as YAML.
	ExampleYaml string
	// Top-Level fields in the API.
	Fields []CustomResourceField
	// API scope. "Cluster" or "Namespaced".
	Scope CustomResourceScope
}

type CustomResourceScope string

const (
	CustomResourceScopeCluster    CustomResourceScope = "Cluster"
	CustomResourceScopeNamespaced CustomResourceScope = "Namespaced"
)

// Field/Object property.
type CustomResourceField struct {
	// JSON tag or struct field name of the field.
	Name string
	// Field documentation block.
	Doc DocumentationBlock
	// Field data type.
	Type string
	// If the field is mandatory to be set.
	IsRequired bool
}

// Any other object present in the API group.
// Usually making up the structure of one or more CRs.
type CustomResourceSubObject struct {
	// Name of the go struct.
	Name string
	// Struct documentation block.
	Doc DocumentationBlock
	// Fields that are part of the object.
	// Might include fields from embedded objects.
	Fields []CustomResourceField
	// Objects embedded/inlined into this object.
	EmbeddedSubObjects []string
	// Wether this object is inlined into another object.
	IsEmbedded bool
	// Parent objects that reference this object in one of their fields.
	Parents []string
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

// Code generator annotations
const (
	// Specified on the package to set the API group name.
	// Used by kubebuilders controller-gen
	groupNameAnnotation = "groupName"
	// Indicates root of a custom resource.
	objectRootAnnotation = "kubebuilder:object:root"
	// Scope of the API. Cluster or Namespace.
	scopeAnnotation = "kubebuilder:resource:scope"
	// Field defaults applied during API server admission.
	defaultAnnotation = "kubebuilder:default"
	// Custom example value for this documentation generator.
	// use as +example=<example value>
	// Used when generating the YAML example object.
	exampleAnnotation = "example"
)
