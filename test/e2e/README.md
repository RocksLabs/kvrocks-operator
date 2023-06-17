# e2e test for Kvrocks Operator

## Cluster setup

These tests musts be running on a Kubernetes cluster, and we provide two ways to create a cluster for end-to-end(e2e) testing:

### Script
You can use the shell script in `script` to install the cluster.

```bash
make e2e-setup # create a cluster with minikube 
```

> Note: The script will install the cluster with minikube by default, if you want to use other tools, you can modify the script. (Welcome to contribute your script to support more tools).


Now you can use the following config to test the operator:

```go
var config = &Config{
	InstallKubernetes:        false,
	UninstallKubernetes:      false,
	InstallKruise:            true,
	InstallKvrocksOperator:   true,
	UsingKvorcksOperatorHelm: true,
	InstallKruiseVersion:     "1.4.0",
	ClusterName:              "e2e-test",
	Namespace:                "kvrocks",
	ClusterConnectionConfig: ClusterConnectionConfig{
		KubeConfig:  filepath.Join(homedir.HomeDir(), ".kube", "config"),
		Development: true,
	},
}
```

```bash
make e2e-destroy # destroy the cluster
```


### Only Code

We provide [kubernetes_env.go](./util/kubernetes_env.go) to create a cluster, you can use the following config to create a cluster:

```go
var config = &Config{
	InstallKubernetes:        true,
	UninstallKubernetes:      true,
	InstallKruise:            true,
	InstallKvrocksOperator:   true,
	UsingKvorcksOperatorHelm: true,
	InstallKruiseVersion:     "1.4.0",
	ClusterName:              "e2e-test",
	Namespace:                "kvrocks",
	ClusterConnectionConfig: ClusterConnectionConfig{
		KubeConfig:  filepath.Join(homedir.HomeDir(), ".kube", "config"),
		Development: true,
	},
}
```

> Note: We currently do not provide a retry function in the code, so you need to run the code again if the cluster is not ready.


## Running e2e test
You should run the following command to download the `ginkgo` tool:

```bash
make ginkgo  # download the ginkgo tool
```

Now, you can use the following command to run the e2e test for standard mode:
```bash
make e2e-test mode=standard # run the e2e test for standard mode
```