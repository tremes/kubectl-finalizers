package discovery

import (
	"strings"

	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

type DiscoverAPI struct {
	configFlags *genericclioptions.ConfigFlags
}

func New(cFlags *genericclioptions.ConfigFlags) *DiscoverAPI {
	return &DiscoverAPI{
		configFlags: cFlags,
	}
}

// Discover finds all the API resources and returns them as maps/sets of GVRs
func (d *DiscoverAPI) Discover(clusterScopedOnly bool) (map[schema.GroupVersionResource]struct{}, error) {
	return d.find(clusterScopedOnly)
}

func (d *DiscoverAPI) find(clusterScopedOnly bool) (map[schema.GroupVersionResource]struct{}, error) {
	discovery, err := d.configFlags.ToDiscoveryClient()
	if err != nil {
		return nil, err
	}
	apiResourcesLists, err := discovery.ServerPreferredResources()
	if err != nil {
		return nil, err
	}

	result := make(map[schema.GroupVersionResource]struct{})

	for _, apiResourceList := range apiResourcesLists {
		for _, apiResource := range apiResourceList.APIResources {

			if clusterScopedOnly && apiResource.Namespaced || !clusterScopedOnly && !apiResource.Namespaced {
				continue
			}

			if strings.Contains(apiResource.Name, "/status") {
				continue
			}
			gvr := schema.GroupVersionResource{
				Resource: apiResource.Name,
			}

			if apiResource.Group == "" {
				gv := strings.Split(apiResourceList.GroupVersion, "/")
				if len(gv) != 2 {
					//likely empty group
					gvr.Version = gv[0]
				} else {
					gvr.Group = gv[0]
					gvr.Version = gv[1]
				}
			} else {
				gvr.Group = apiResource.Group
				gvr.Version = apiResource.Version
			}

			if _, exist := result[gvr]; !exist {
				result[gvr] = struct{}{}
			}
		}
	}
	return result, nil
}
