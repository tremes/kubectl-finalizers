# kubectl-finalizers
kubectl-finalizers identifies Kubernetes resources that are pending deletion (non-nil deletion timestamp and non-empty finalizers) and patches them to remove the finalizer values.

## Demo

Look at the short demonstration:

<video width="640" height="480" controls>
  <source src="./short-demo.mp4" type="video/mp4">
  Your browser does not support the video tag. <a href="./short-demo.mp4">Download the demo video</a>
</video>

## How to use

```bash
./bin/finalizers
```
It looks for all the namespaced pending resources in the current namespace.

```bash
./bin/finalizers --namespace test
```
It looks for all the namespaced pending resources in the `test` namespace.

```bash
./bin/finalizers --clusterscoped
```
It looks for all the clusterscoped pending resources.
