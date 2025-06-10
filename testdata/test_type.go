package testdata

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

// +kubebuilder:object:root=true
type TestObject struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              TestObjectSpec `json:"spec"`
}

type TestObjectSpec struct {
	Object  Object   `json:"object"`
	Objects []Object `json:"objects"`
	Object  `json:",inline"`
	// +example={field1: apps/v1, field2: false}
	ExampleObj Object    `json:"exampleObj"`
	UID        types.UID `json:"uid"`
}

type Object struct {
	Field1      string      `json:"field1"`
	Field2      bool        `json:"field2,omitempty"`
	Field3      []string    `json:"field3"`
	Field4      int         `json:"field4"`
	Field5      int32       `json:"field5"`
	Field6      int64       `json:"field6"`
	EmptyObject EmptyObject `json:"empty"`
	// +example="Test 123"
	Example string `json:"example"`
}

type EmptyObject struct{}
