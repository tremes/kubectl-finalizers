package find

import (
	"context"
	"fmt"
	"runtime"
	"sync"

	"k8s.io/apimachinery/pkg/api/errors"
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
	if restConfig == nil {
		return nil, fmt.Errorf("please provide a valid rest.Config")
	}
	mdClient, err := metadata.NewForConfig(restConfig)
	if err != nil {
		return nil, err
	}

	return &Finder{
		mdCli: mdClient,
	}, nil
}

// Find is non-blocking and spawns Go routines (leveraging worker pool) to asynchronously read
// resources with every GVR. It returns a channel providing the resources found with the finalizers.
func (f *Finder) Find(ctx context.Context, gvrs map[schema.GroupVersionResource]struct{}, namespace string) <-chan *ResourceIdentifier {
	finalizerCh := make(chan *ResourceIdentifier, len(gvrs))
	gvrCh := make(chan schema.GroupVersionResource, len(gvrs))

	go func() {
		defer close(finalizerCh)

		var waitForWorkers sync.WaitGroup
		workers := runtime.NumCPU() * 4
		for w := range workers {
			waitForWorkers.Add(1)
			go func(id int) {
				defer waitForWorkers.Done()
				f.readResources(ctx, id, gvrCh, finalizerCh, namespace)
			}(w)
		}

		// iterate of map of GVRs and pass each to be read
		for gvr := range gvrs {
			gvrCh <- gvr
		}
		close(gvrCh)
		waitForWorkers.Wait()
	}()
	return finalizerCh
}

// readResources reads from the channel for schema.GroupVersionResource. It lists
// all the resources (for given GVR) and check the deletionTimestamp and finalizers attribute.
// If some pending resource is found, it is paased to the channel for ResourceIdentifier
func (f *Finder) readResources(ctx context.Context, workerID int, gvrCh <-chan schema.GroupVersionResource, ch chan<- *ResourceIdentifier, namespace string) {
	for gvr := range gvrCh {
		klog.V(6).InfoS("Worker ", "id", workerID, " started processing of resource", gvr)
		var getter metadata.ResourceInterface
		if namespace != "" {
			getter = f.mdCli.Resource(gvr).Namespace(namespace)
		} else {
			getter = f.mdCli.Resource(gvr)
		}

		listOpt := v1.ListOptions{
			Continue: "",
			Limit:    5,
		}

		for {
			l, err := getter.List(ctx, listOpt)
			if err != nil {
				if errors.IsMethodNotSupported(err) {
					break
				}
				klog.V(4).ErrorS(err, "Failed to list", "resource", gvr)
				break
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

			if l.Continue == "" {
				break
			}
			listOpt.Continue = l.Continue
		}
		klog.V(6).InfoS("Worker ", "id", workerID, " finished processing of resource", gvr)
	}
}
