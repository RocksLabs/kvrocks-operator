# Development Guide

This document is a guide to start developing the Kvrocks operator.

## Set up the development environment

We recommend you use a Linux/macOS platform for development.

### Install Required Software
-   Go
    
    -   Currently, building the Kvrocks operator requires Go 1.17 or later.
-   Docker
-   Kubernetes cluster
    
    -   You can use the minikube to provision your local Kubernetes cluster

## Start debugging

Before getting started, please run the following commands to perform some checks:

### Check the Kubernetes cluster

```bash
kubectl version --short
```

### Install the local manifests

The local manifests contain the CRD, run the following command to install it:

```bash
make install
```

Expected output:
```bash
kvrocks.kvrocks.com                        2023-04-22T06:23:33Z
```

### Run the operator locally

1. Install OpenKruise

```bash
helm repo add openkruise https://openkruise.github.io/charts/
helm repo update
helm install kruise openkruise/kruise --version 1.4.0
```

2. Run the operator

```bash
make run
```
Now stop the process and we're ready for debugging.

## Debugging in VSCode

Debugging in VSCode requires a launch configuration, you can use the following configuration:

```jsonc launch.json
{
    // Use IntelliSense to learn about possible attributes.
    // Hover to view descriptions of existing attributes.
    // For more information, visit: https://go.microsoft.com/fwlink/?linkid=830387
    "version": "0.2.0",
    "configurations": [
        {
            "name": "Kvrocks Operator",
            "type": "go",
            "request": "launch",
            "mode": "auto",
            "program": "${workspaceFolder}/main.go",
            "args": [],
            "env": {
                "NO_PROXY": "kubernetes.docker.internal,127.0.0.1,localhost"
            },
        }
    ]
}
```
Now start debugging by clicking the menu **[Run > Start Debugging]** or pressing **F5**. The following is a list of significant functions/methods/files that might be useful as breakpoints:

* `main() main.go`, the entry point of the Kvrocks operator
* `KVRocksReconciler.Reconcile() pkg/controller/kvrocks_controller.go`, the core function of the Kvrocks operator


## Building the Kvrocks operator

To build a binary of the Kvrocks operator, run the following command.

```bash
make build
```

The binary would be generated in the `bin` directory


To build a Docker/OCI-compatible image of the Kvrocks operator, run the following command:

```bash
# build image with tag "kvrocks.com/kvrockslabs/kvrocks-operator:latest"
make docker-build

# build image with tag "kvrockslabs/kvrocks-operator:latest"
REGISTRY=kvrockslabs make docker-build

# build image with tag "kvrocks.com/kvrockslabs/kvrocks-operator:nightly"
TAG=nightly make docker-build
```

To build image with `vendor` run the following command:

```bash
make docker-build-vendor
```


