
# KvrocksOperator
This project provides an operator for managing kvrocks instances on Kubernetes.It is built using the [kubebuilder](https://github.com/kubernetes-sigs/kubebuilder) framework. 

## Quick Start

### Deploy

1. Install OpenKruise

```
helm repo add openkruise https://openkruise.github.io/charts/
helm repo update
helm install kruise openkruise/kruise --version 1.4.0
```
2. Create ns kvrocks
```
kubectl create ns kvrocks
```

3. Use helm to install and manage the crd/operator 
```
helm repo add kvrocks-operator https://rockslabs.github.io/kvrocks-operator
helm install kvrocks-crd kvrocks-operator/kvrocks-crd -n kvrocks
helm install kvrocks-operator kvrocks-operator/kvrocks-operator -n kvrocks
```

4. Modify the `examples/standard.yaml examples/sentinel.yaml` and then apply
```
kubectl apply -f examples/standard.yaml examples/sentinel.yaml
```

## Notice
> 1.You need to prepare the storageclass and indicate it in the manifest.<br>
> 2.We currently support only standalone kvrocks clusters and those with sentinels. Cluster mode is temporarily unsupported.<br>
> 3.Note the naming rules of kvrocks custom resource.
> `kvrocks-standard-1-demo`
> consists of four parts<br>
> (1). kvrocks<br>
> (2). type<br>
> (3). Indicate the sentinel cluster index used. In this case, the sentinel cluster is: sentinel-1 <br>
> (4). cluster name <br>

## Running e2e tests

Please refer to the [Test README](/test/e2e/README.md) for more information.

## Observability Configuration Guide

Please refer to the [Observability Configuration Guide](/docs/observability.md) for more information.

## Development Guide

Please refer to the [Development Guide](/docs/development.md) for more information.

## Design Details
For more design etails, please refer to [Design Document](/docs/design.md).

## Community 
**Feel free to ask us any questions through [issues](https://github.com/KvrocksLabs/kvrocks-operator/issues) or [slack](https://kvrocks.slack.com/ssb/redirect#/shared-invite/email)**
