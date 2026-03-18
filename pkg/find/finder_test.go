package find

import (
	"context"
	"testing"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/metadata/fake"
)

var deploymentGVR = schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "deployments"}

func TestFinder_Find(t *testing.T) {
	tests := []struct {
		name          string
		gvrs          map[schema.GroupVersionResource]struct{}
		namespace     string
		mockObjects   []runtime.Object
		expectedCount int
		//expectFinished bool
	}{
		{
			name: "find resources with finalizers",
			gvrs: map[schema.GroupVersionResource]struct{}{
				deploymentGVR: {},
			},
			namespace: "default",
			mockObjects: []runtime.Object{
				&v1.PartialObjectMetadata{
					TypeMeta: v1.TypeMeta{
						APIVersion: "apps/v1",
						Kind:       "Deployment",
					},
					ObjectMeta: v1.ObjectMeta{
						Name:              "test-deployment",
						Namespace:         "default",
						DeletionTimestamp: &v1.Time{Time: time.Now()},
						Finalizers:        []string{"test.finalizer/cleanup"},
					},
				},
			},
			expectedCount: 1,
		},
		{
			name: "no resources with finalizers",
			gvrs: map[schema.GroupVersionResource]struct{}{
				deploymentGVR: {},
			},
			namespace: "default",
			mockObjects: []runtime.Object{
				&v1.PartialObjectMetadata{
					TypeMeta: v1.TypeMeta{
						APIVersion: "apps/v1",
						Kind:       "Deployment",
					},
					ObjectMeta: v1.ObjectMeta{
						Name:      "test-deployment",
						Namespace: "default",
					},
				},
			},
			expectedCount: 0,
		},
		{
			name: "resource with deletion timestamp but no finalizers",
			gvrs: map[schema.GroupVersionResource]struct{}{
				deploymentGVR: {},
			},
			namespace: "default",
			mockObjects: []runtime.Object{
				&v1.PartialObjectMetadata{
					TypeMeta: v1.TypeMeta{
						APIVersion: "apps/v1",
						Kind:       "Deployment",
					},
					ObjectMeta: v1.ObjectMeta{
						Name:              "test-deployment",
						Namespace:         "default",
						DeletionTimestamp: &v1.Time{Time: time.Now()},
						Finalizers:        []string{},
					},
				},
			},
			expectedCount: 0,
		},
		{
			name: "resource with finalizers but no deletion timestamp",
			gvrs: map[schema.GroupVersionResource]struct{}{
				deploymentGVR: {},
			},
			namespace: "default",
			mockObjects: []runtime.Object{
				&v1.PartialObjectMetadata{
					TypeMeta: v1.TypeMeta{
						APIVersion: "apps/v1",
						Kind:       "Deployment",
					},
					ObjectMeta: v1.ObjectMeta{
						Name:       "test-deployment",
						Namespace:  "default",
						Finalizers: []string{"test.finalizer/cleanup"},
					},
				},
			},
			expectedCount: 0,
		},
		{
			name: "multiple resources mixed conditions",
			gvrs: map[schema.GroupVersionResource]struct{}{
				deploymentGVR: {},
			},
			namespace: "default",
			mockObjects: []runtime.Object{
				&v1.PartialObjectMetadata{
					TypeMeta: v1.TypeMeta{
						APIVersion: "apps/v1",
						Kind:       "Deployment",
					},
					ObjectMeta: v1.ObjectMeta{
						Name:              "stuck-deployment",
						Namespace:         "default",
						DeletionTimestamp: &v1.Time{Time: time.Now()},
						Finalizers:        []string{"test.finalizer/cleanup"},
					},
				},
				&v1.PartialObjectMetadata{
					TypeMeta: v1.TypeMeta{
						APIVersion: "apps/v1",
						Kind:       "Deployment",
					},
					ObjectMeta: v1.ObjectMeta{
						Name:      "normal-deployment",
						Namespace: "default",
					},
				},
			},
			expectedCount: 1,
		},
		{
			name:          "empty gvrs map",
			gvrs:          map[schema.GroupVersionResource]struct{}{},
			namespace:     "default",
			mockObjects:   []runtime.Object{},
			expectedCount: 0,
		},
		{
			name: "cluster-scoped resource (empty namespace)",
			gvrs: map[schema.GroupVersionResource]struct{}{
				{Group: "", Version: "v1", Resource: "nodes"}: {},
			},
			namespace: "",
			mockObjects: []runtime.Object{
				&v1.PartialObjectMetadata{
					TypeMeta: v1.TypeMeta{
						APIVersion: "v1",
						Kind:       "Node",
					},
					ObjectMeta: v1.ObjectMeta{
						Name:              "test-node",
						DeletionTimestamp: &v1.Time{Time: time.Now()},
						Finalizers:        []string{"node.finalizer/cleanup"},
					},
				},
			},
			expectedCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scheme := runtime.NewScheme()
			// Register metadata types
			scheme.AddKnownTypes(metav1.SchemeGroupVersion, &metav1.PartialObjectMetadata{}, &metav1.PartialObjectMetadataList{})
			fakeClient := fake.NewSimpleMetadataClient(scheme, tt.mockObjects...)

			finder := &Finder{
				mdCli: fakeClient,
			}

			ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
			defer cancel()

			resultCh := finder.Find(ctx, tt.gvrs, tt.namespace)

			var results []*ResourceIdentifier
			var finished bool

			// Collect results with timeout
			timeout := time.After(2 * time.Second)
			for !finished {
				select {
				case result, ok := <-resultCh:
					if !ok {
						finished = true
						break
					}
					results = append(results, result)
				case <-timeout:
					t.Errorf("Find() timed out waiting for results")
					return
				}
			}

			if !finished {
				t.Errorf("Find() expected channel to be closed but it remained open")
			}

			if len(results) != tt.expectedCount {
				t.Errorf("Find() expected %d results, got %d", tt.expectedCount, len(results))
			}

			// Verify result content for positive cases
			if tt.expectedCount > 0 && len(results) > 0 {
				result := results[0]
				if result.Name == "" {
					t.Errorf("Find() result should have non-empty Name")
				}
				if len(result.Finalizers) == 0 {
					t.Errorf("Find() result should have non-empty Finalizers")
				}
				if tt.namespace != "" && result.Namespace != tt.namespace {
					t.Errorf("Find() result namespace expected %s, got %s", tt.namespace, result.Namespace)
				}
			}
		})
	}
}

func TestFinder_FindWithCancelledContext(t *testing.T) {
	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(metav1.SchemeGroupVersion, &metav1.PartialObjectMetadata{}, &metav1.PartialObjectMetadataList{})
	fakeClient := fake.NewSimpleMetadataClient(scheme)
	finder := &Finder{
		mdCli: fakeClient,
	}

	gvrs := map[schema.GroupVersionResource]struct{}{
		deploymentGVR: {},
	}

	ctx, cancel := context.WithCancel(t.Context())
	cancel() // Cancel immediately

	resultCh := finder.Find(ctx, gvrs, "default")

	timeout := time.After(1 * time.Second)
	var finished bool

	for !finished {
		select {
		case _, ok := <-resultCh:
			if !ok {
				finished = true
				break
			}
		case <-timeout:
			t.Errorf("Find() should handle cancelled context and finish quickly")
			return
		}
	}

	if !finished {
		t.Errorf("Find() should close result channel when context is cancelled")
	}
}
