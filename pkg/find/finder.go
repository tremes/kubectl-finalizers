package find

import (
	"context"
	"sync"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/metadata"
	"k8s.io/client-go/rest"
	"k8s.io/klog/v2"
)

type Finder struct {
	mdCli metadata.Interface
}

// NewFinder
func NewFinder(restConfig *rest.Config) (*Finder, error) {
	mdClient, err := metadata.NewForConfig(restConfig)
	if err != nil {
		return nil, err
	}

	return &Finder{
		mdCli: mdClient,
	}, nil
}

// Lister
func (f *Finder) Find(ctx context.Context, gvrs map[schema.GroupVersionResource]struct{}, namespace string) <-chan *ResourceIdentifier {
	finalizerCh := make(chan *ResourceIdentifier, len(gvrs))

	go func() {
		defer close(finalizerCh)
		var wg sync.WaitGroup
		wg.Add(len(gvrs))
		for gvr := range gvrs {
			go func(gvr schema.GroupVersionResource) {
				defer wg.Done()
				f.findFinalizers(ctx, gvr, finalizerCh, namespace)
			}(gvr)
		}
		wg.Wait()
	}()

	return finalizerCh
}

func (f *Finder) findFinalizers(ctx context.Context, gvr schema.GroupVersionResource, ch chan<- *ResourceIdentifier, namespace string) {
	getter := f.mdCli.Resource(gvr)

	if namespace != "" {
		getter.Namespace(namespace)
	}

	// TODO list with limit
	l, err := getter.List(ctx, v1.ListOptions{})
	if err != nil {
		// TODO fix & propagate errors
		return
	}

	for i := range l.Items {
		partMetadata := &l.Items[i]

		if partMetadata.DeletionTimestamp != nil && len(partMetadata.Finalizers) > 0 {
			r := &ResourceIdentifier{
				Name:                 partMetadata.Name,
				Namespace:            partMetadata.Namespace,
				GroupVersionResource: gvr,
				Finalizers:           partMetadata.Finalizers,
			}
			klog.V(4).InfoS("Found pending:", "name", partMetadata.Name, "resource", "gvr", gvr, "finalizers", partMetadata.Finalizers)
			select {
			case ch <- r:
			case <-ctx.Done():
				return
			}
		}
	}
}
