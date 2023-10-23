# e2e test for Kvrocks Operator

## Running e2e test
You should run the following command to download the `ginkgo` tool:

```bash
make ginkgo  # download the ginkgo tool
```

Then, you can refer to the development guide docs to install the Telepresence tool, which supports connecting to the cluster.

To execute the end-to-end test in standard mode, use the command below:
```bash
make e2e-test mode=standard CONFIG_FILE_PATH=config/config.yaml # run the e2e test for standard mode
```

For testing in cluster mode, utilize the following command:
```bash
make e2e-test mode=cluster CONFIG_FILE_PATH=config/config.yaml # run the e2e test for cluster mode
```



The details of [config.yaml](config/config.yaml) are as follows:
```yaml
kruiseVersion: 1.4.0
clusterName: e2e-test
namespace: kvrocks
manifestDir: /path/to/your/manifests
chaosMeshEnabled: true
kubeConfig:
```
The above config.yaml is the default config file, and it performs the following actions:

1. Installs the Kubernetes Cluster named `e2e-test` using the `minikube` tool, and connects to the cluster using the `telepresence` tool.

2. Installs kruise of version `1.4.0`.

3. Installs the `kvrocks-operator/kvrocks` in the `kvrocks` namespace. If you wish to customize the namespace, ensure that the namespace mentioned in the YAML file located in the MainfestDir matches the one specified in the config file.

4. Installs kvrocks from the YAML files found in the `manifestDir` directory. The default value `../../../examples/`

5. We use the `chaosmesh` tool to kill kvrocks operator every two minutes. If you want to disable this feature, you can set `chaosMeshEnabled` to `false`.

If you want to run the e2e test in a deployed cluster, you can use the following config file:

```yaml
kruiseVersion: 1.4.0
clusterName: e2e-test
namespace: kvrocks
manifestDir: /path/to/your/manifests
chaosMeshEnabled: true
kubeConfig: /path/to/your/kubeconfig
```
Note that, you should ensure the local environment can connect to the cluster, and the `clusterName` needs to match the `cluster` in the `current-context` of your kubeconfig.
