# kubectl-finalizers
kubectl-finalizers identifies Kubernetes resources that are pending deletion (non-nil deletion timestamp and non-empty finalizers) and patches them to remove the finalizer values.

## Demo

Look at the short demonstration:

![Demo](./short-demo.gif)

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
