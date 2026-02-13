package cmd

import (
	"context"
	"flag"

	"github.com/spf13/cobra"
	"github.com/tremes/kubectl-finalizers/pkg/discovery"
	"github.com/tremes/kubectl-finalizers/pkg/find"
	"github.com/tremes/kubectl-finalizers/pkg/patch"

	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/rest"
	"k8s.io/klog/v2"
)

type options struct {
	ClusterScoped bool
}

type FinalizersPlugin struct {
	configFlags *genericclioptions.ConfigFlags
}

func NewFinalizersPlugin() *cobra.Command {
	opt := options{}
	cFlags := genericclioptions.NewConfigFlags(true)
	cmd := &cobra.Command{
		Use:          "finalizers",
		Short:        "Find pending resources with finalizers",
		Example:      "kubectl finalizers --namespace default",
		SilenceUsage: true,
		RunE: func(c *cobra.Command, args []string) error {

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			restConfig, err := cFlags.ToRESTConfig()
			if err != nil {
				return err
			}
			restConfig.WarningHandler = rest.NoWarnings{}
			restConfig.QPS = -1
			restConfig.Burst = -1

			d := discovery.New(cFlags)
			resources, err := d.Discover(opt.ClusterScoped)

			finder, err := find.NewFinder(restConfig)
			if err != nil {
				return err
			}

			klog.V(4).InfoS("Found API resources", "resources", len(resources))

			var finalizersCh <-chan *find.ResourceIdentifier
			if opt.ClusterScoped {
				finalizersCh = finder.Find(ctx, resources, "")
			} else {
				ns, _, err := cFlags.ToRawKubeConfigLoader().Namespace()
				if err != nil {
					return err
				}

				finalizersCh = finder.Find(ctx, resources, ns)
			}

			patcher, err := patch.New(restConfig)
			if err != nil {
				return err
			}

			patcher.Patch(ctx, finalizersCh)

			return nil
		},
	}
	cmd.Flags().BoolVarP(&opt.ClusterScoped, "clusterscoped", "c", false, "Check only clusterscoped resources")
	cmd.Flags().AddGoFlagSet(flag.CommandLine)
	cFlags.AddFlags(cmd.Flags())

	return cmd
}

func Execute() error {
	klog.InitFlags(nil)
	return NewFinalizersPlugin().Execute()
}
