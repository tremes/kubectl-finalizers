package discovery

import (
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/rest"
	"k8s.io/klog/v2"
)

type DiscoverAPI struct {
	restConfig *rest.Config
}

func New(restConfig *rest.Config) *DiscoverAPI {
	return &DiscoverAPI{
		restConfig: restConfig,
	}
}

// Discover finds all the API resources and returns them as maps/sets of GVRs
func (d *DiscoverAPI) Discover(clusterScopedOnly bool) (map[schema.GroupVersionResource]struct{}, error) {
	return d.find(clusterScopedOnly)
}

func (d *DiscoverAPI) find(clusterScopedOnly bool) (map[schema.GroupVersionResource]struct{}, error) {
	discoveryClient, err := discovery.NewDiscoveryClientForConfig(d.restConfig)
	if err != nil {
		return nil, err
	}
	apiResourcesLists, err := discoveryClient.ServerPreferredResources()
	if err != nil {
		if len(apiResourcesLists) == 0 {
			return nil, err
		}
		klog.ErrorS(err, "Failed to discover some API groups, continuing with partial results")
	}

	return filterResources(apiResourcesLists, clusterScopedOnly), nil
}

func filterResources(apiResourcesLists []*metav1.APIResourceList, clusterScopedOnly bool) map[schema.GroupVersionResource]struct{} {
	result := make(map[schema.GroupVersionResource]struct{})

	for _, apiResourceList := range apiResourcesLists {
		for _, apiResource := range apiResourceList.APIResources {

			if clusterScopedOnly && apiResource.Namespaced || !clusterScopedOnly && !apiResource.Namespaced {
				continue
			}

			// filter all subresources - /status, pods/log
			if strings.Contains(apiResource.Name, "/") {
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
	return result
}
