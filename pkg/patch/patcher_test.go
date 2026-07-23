package patch

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"testing"

	"github.com/tremes/kubectl-finalizers/pkg/find"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	dynamicfake "k8s.io/client-go/dynamic/fake"
	k8stesting "k8s.io/client-go/testing"
)

func TestBuildPatchData(t *testing.T) {
	tests := []struct {
		name            string
		resourceVersion string
	}{
		{name: "typical version", resourceVersion: "12345"},
		{name: "empty version", resourceVersion: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := buildPatchData(tt.resourceVersion)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			var parsed map[string]interface{}
			if err := json.Unmarshal(data, &parsed); err != nil {
				t.Fatalf("invalid JSON: %v", err)
			}

			metadata, ok := parsed["metadata"].(map[string]interface{})
			if !ok {
				t.Fatal("missing metadata key")
			}

			if rv, ok := metadata["resourceVersion"].(string); !ok || rv != tt.resourceVersion {
				t.Errorf("expected resourceVersion %q, got %v", tt.resourceVersion, metadata["resourceVersion"])
			}

			if _, exists := metadata["finalizers"]; !exists {
				t.Error("finalizers key should be present")
			}
			if metadata["finalizers"] != nil {
				t.Errorf("finalizers should be null, got %v", metadata["finalizers"])
			}
		})
	}
}

func TestPatch_UserConfirmsYes(t *testing.T) {
	scheme := runtime.NewScheme()
	fakeCli := dynamicfake.NewSimpleDynamicClient(scheme)

	var patchCalled bool
	fakeCli.PrependReactor("patch", "*", func(action k8stesting.Action) (bool, runtime.Object, error) {
		patchCalled = true
		return true, nil, nil
	})

	p := &patcher{cli: fakeCli}

	ch := make(chan *find.ResourceIdentifier, 1)
	ch <- &find.ResourceIdentifier{
		GroupVersionResource: schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "deployments"},
		Name:                 "stuck-deploy",
		Namespace:            "default",
		Finalizers:           []string{"test/finalizer"},
		ResourceVersion:      "999",
	}
	close(ch)

	r, w, _ := os.Pipe()
	fmt.Fprintln(w, "y")
	w.Close()

	oldStdin := os.Stdin
	os.Stdin = r
	defer func() { os.Stdin = oldStdin }()

	p.Patch(t.Context(), ch, find.Options{}, "")

	if !patchCalled {
		t.Error("expected patch to be called when user confirms with 'y'")
	}
}

func TestPatch_UserDeclinesNo(t *testing.T) {
	scheme := runtime.NewScheme()
	fakeCli := dynamicfake.NewSimpleDynamicClient(scheme)

	var patchCalled bool
	fakeCli.PrependReactor("patch", "*", func(action k8stesting.Action) (bool, runtime.Object, error) {
		patchCalled = true
		return true, nil, nil
	})

	p := &patcher{cli: fakeCli}

	ch := make(chan *find.ResourceIdentifier, 1)
	ch <- &find.ResourceIdentifier{
		GroupVersionResource: schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "deployments"},
		Name:                 "stuck-deploy",
		Namespace:            "default",
		Finalizers:           []string{"test/finalizer"},
		ResourceVersion:      "999",
	}
	close(ch)

	r, w, _ := os.Pipe()
	fmt.Fprintln(w, "n")
	w.Close()

	oldStdin := os.Stdin
	os.Stdin = r
	defer func() { os.Stdin = oldStdin }()

	p.Patch(t.Context(), ch, find.Options{}, "")

	if patchCalled {
		t.Error("patch should not be called when user declines with 'n'")
	}
}

func TestPatch_EmptyChannel(t *testing.T) {
	scheme := runtime.NewScheme()
	fakeCli := dynamicfake.NewSimpleDynamicClient(scheme)

	p := &patcher{cli: fakeCli}

	ch := make(chan *find.ResourceIdentifier)
	close(ch)

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	p.Patch(t.Context(), ch, find.Options{}, "default")

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	io.Copy(&buf, r)

	expected := "No resources pending deletion were found in the default namespace.\n"
	if buf.String() != expected {
		t.Errorf("expected output %q, got %q", expected, buf.String())
	}
}

func TestPatch_EmptyChannelClusterScoped(t *testing.T) {
	scheme := runtime.NewScheme()
	fakeCli := dynamicfake.NewSimpleDynamicClient(scheme)

	p := &patcher{cli: fakeCli}

	ch := make(chan *find.ResourceIdentifier)
	close(ch)

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	p.Patch(t.Context(), ch, find.Options{ClusterScoped: true}, "")

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	io.Copy(&buf, r)

	expected := "No cluster-scoped resources pending deletion were found.\n"
	if buf.String() != expected {
		t.Errorf("expected output %q, got %q", expected, buf.String())
	}
}
