## Observability Configuration Guide
This document serves as a guide to help you properly set up observability for the Kvrocks in kubernetes.

## Prerequisites
Before you proceed, ensure that the [Kvrocks operator](../deploy/operator/) and [kvrocks instances](../examples/standard.yaml) are deployed.

The kvrocks_exporter is set as a sidecar for each pod, as a result, the following should be displayed:

```bash
kubectl get pods -n kvrocks
NAME                        READY   STATUS    RESTARTS   AGE
kvrocks-standard-1-demo-0   2/2     Running   0          33s
kvrocks-standard-1-demo-1   2/2     Running   0          33s
kvrocks-standard-1-demo-2   2/2     Running   0          33s
```

## Configuring Prometheus
To scrape the metrics of Kvrocks, you can utilize the configuration below:

```yaml
 - job_name: 'kvrocks'
      kubernetes_sd_configs:
      - role: pod
      relabel_configs:
      - source_labels: [__meta_kubernetes_pod_name]
        action: keep
        regex: kvrocks-standard-1-demo-.*
      - source_labels: [__meta_kubernetes_pod_container_name]
        action: keep
        regex: kvrocks-exporter
      - source_labels: [__meta_kubernetes_pod_ip]
        regex: (.*)
        target_label: __address__
        replacement: $1:9121
```

## Reference
more details about the kvrocks_exporter, please refer to [kvrocks_exporter](https://github.com/RocksLabs/kvrocks_exporter)
