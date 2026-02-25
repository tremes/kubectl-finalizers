package patch

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/tremes/kubectl-finalizers/pkg/find"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
)

type patcher struct {
	cli       *dynamic.DynamicClient
	patchData []byte
}

// New creates a new instance of the patcher type
func New(restConfig *rest.Config) (*patcher, error) {
	dynamicCli, err := dynamic.NewForConfig(restConfig)
	if err != nil {
		return nil, err
	}

	patchData := map[string]interface{}{
		"metadata": map[string]interface{}{
			"finalizers": []string{},
		},
	}

	patchBytes, err := json.Marshal(patchData)
	if err != nil {
		return nil, err
	}

	return &patcher{
		cli:       dynamicCli,
		patchData: patchBytes,
	}, nil
}

// Patch reads from the provided channel. If it receives some resource, it asks
// user for patching.
func (p *patcher) Patch(ctx context.Context, ch <-chan *find.ResourceIdentifier) {
	for r := range ch {
		var confirm string
		fmt.Printf("Found %s %s resource with %s finalizers. Do you want to remove the finalizers? [y/n] \n", r.Name,
			r.GroupVersionResource.Resource, r.Finalizers)
		fmt.Scan(&confirm)

		if confirm != "y" {
			continue
		}

		_, err := p.cli.Resource(r.GroupVersionResource).
			Namespace(r.Namespace).
			Patch(ctx, r.Name, types.MergePatchType, p.patchData, v1.PatchOptions{})
		if err != nil {
			// TODO handler error
			fmt.Println("ERROR patching ", err)
		}
	}
}
