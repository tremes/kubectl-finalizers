package discovery

import (
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func TestFilterResources(t *testing.T) {
	tests := []struct {
		name             string
		apiResourceLists []*metav1.APIResourceList
		clusterScoped    bool
		expected         map[schema.GroupVersionResource]struct{}
	}{
		{
			name: "core API group namespaced resources",
			apiResourceLists: []*metav1.APIResourceList{
				{
					GroupVersion: "v1",
					APIResources: []metav1.APIResource{
						{Name: "pods", Namespaced: true},
						{Name: "services", Namespaced: true},
					},
				},
			},
			clusterScoped: false,
			expected: map[schema.GroupVersionResource]struct{}{
				{Group: "", Version: "v1", Resource: "pods"}:     {},
				{Group: "", Version: "v1", Resource: "services"}: {},
			},
		},
		{
			name: "extended API group",
			apiResourceLists: []*metav1.APIResourceList{
				{
					GroupVersion: "apps/v1",
					APIResources: []metav1.APIResource{
						{Name: "deployments", Namespaced: true},
					},
				},
			},
			clusterScoped: false,
			expected: map[schema.GroupVersionResource]struct{}{
				{Group: "apps", Version: "v1", Resource: "deployments"}: {},
			},
		},
		{
			name: "cluster scoped only filters out namespaced",
			apiResourceLists: []*metav1.APIResourceList{
				{
					GroupVersion: "v1",
					APIResources: []metav1.APIResource{
						{Name: "pods", Namespaced: true},
						{Name: "nodes", Namespaced: false},
						{Name: "namespaces", Namespaced: false},
					},
				},
			},
			clusterScoped: true,
			expected: map[schema.GroupVersionResource]struct{}{
				{Group: "", Version: "v1", Resource: "nodes"}:      {},
				{Group: "", Version: "v1", Resource: "namespaces"}: {},
			},
		},
		{
			name: "namespaced only filters out cluster scoped",
			apiResourceLists: []*metav1.APIResourceList{
				{
					GroupVersion: "v1",
					APIResources: []metav1.APIResource{
						{Name: "pods", Namespaced: true},
						{Name: "nodes", Namespaced: false},
					},
				},
			},
			clusterScoped: false,
			expected: map[schema.GroupVersionResource]struct{}{
				{Group: "", Version: "v1", Resource: "pods"}: {},
			},
		},
		{
			name: "subresources are filtered out",
			apiResourceLists: []*metav1.APIResourceList{
				{
					GroupVersion: "v1",
					APIResources: []metav1.APIResource{
						{Name: "pods", Namespaced: true},
						{Name: "pods/status", Namespaced: true},
						{Name: "pods/log", Namespaced: true},
					},
				},
			},
			clusterScoped: false,
			expected: map[schema.GroupVersionResource]struct{}{
				{Group: "", Version: "v1", Resource: "pods"}: {},
			},
		},
		{
			name: "resource with explicit group and version",
			apiResourceLists: []*metav1.APIResourceList{
				{
					GroupVersion: "apps/v1",
					APIResources: []metav1.APIResource{
						{Name: "deployments", Namespaced: true, Group: "apps", Version: "v1"},
					},
				},
			},
			clusterScoped: false,
			expected: map[schema.GroupVersionResource]struct{}{
				{Group: "apps", Version: "v1", Resource: "deployments"}: {},
			},
		},
		{
			name:             "empty input",
			apiResourceLists: []*metav1.APIResourceList{},
			clusterScoped:    false,
			expected:         map[schema.GroupVersionResource]struct{}{},
		},
		{
			name: "duplicate GVRs are deduplicated",
			apiResourceLists: []*metav1.APIResourceList{
				{
					GroupVersion: "v1",
					APIResources: []metav1.APIResource{
						{Name: "pods", Namespaced: true},
						{Name: "pods", Namespaced: true},
					},
				},
			},
			clusterScoped: false,
			expected: map[schema.GroupVersionResource]struct{}{
				{Group: "", Version: "v1", Resource: "pods"}: {},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := filterResources(tt.apiResourceLists, tt.clusterScoped)

			if len(result) != len(tt.expected) {
				t.Errorf("expected %d GVRs, got %d", len(tt.expected), len(result))
			}

			for gvr := range tt.expected {
				if _, ok := result[gvr]; !ok {
					t.Errorf("expected GVR %v not found in result", gvr)
				}
			}

			for gvr := range result {
				if _, ok := tt.expected[gvr]; !ok {
					t.Errorf("unexpected GVR %v in result", gvr)
				}
			}
		})
	}
}
