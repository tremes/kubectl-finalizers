package find

import "k8s.io/apimachinery/pkg/runtime/schema"

type ResourceIdentifier struct {
	schema.GroupVersionResource
	Name       string
	Namespace  string
	Finalizers []string
}
