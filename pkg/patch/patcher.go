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

	patchBytes, _ := json.Marshal(patchData)

	return &patcher{
		cli:       dynamicCli,
		patchData: patchBytes,
	}, nil
}

func (p *patcher) Patch(ctx context.Context, ch <-chan *find.ResourceIdentifier) {
	for r := range ch {
		var confirm string
		fmt.Printf("Found %s %s resource with %s finalizers. Do you want to remove the finalizers? [y/n] ", r.Name,
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
