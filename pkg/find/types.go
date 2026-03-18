package find

import "k8s.io/apimachinery/pkg/runtime/schema"

// ResourceIdentifier is helper type
// representing resource with some finalizers
type ResourceIdentifier struct {
	schema.GroupVersionResource
	Name       string
	Namespace  string
	Finalizers []string
}
