
# KVRocksOperator

## Getting Start

> Note the naming rules of kvrocks
> kvrocks-cluster-1-demo
> consists of four parts
> 1. kvrocks logo
> 2. cluster|standard|sentinel
> 3. Indicate the sentinel cluster to be used. In this case, the sentinel cluster is: sentinel-1
> 4. kvrocks cluster name

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
helm install kvrocks-crd deploy/crd -n kvrocks
helm install kvrocks-operator deploy/operator -n kvrocks
```

### Test

1. standard:
   - kubectl apply -f examples/standard.yaml
2. cluster:
   - kubectl apply -f examples/cluster.yaml

```text
If a storage class is not specified, we will use the default storage class. For instance, in Amazon EKS, the default storage class is gp2.
  storage:
    size: 10Gi  # If size is not specified, its default value is 10Gi.
    class: xxxxx # storage class
```

### Notice

```text
Since kvrocks slot migration does not support migration operations, scaling will be slow (because slots need to be migrated one by one)
```

## Introduction to CR

```yaml
apiVersion: kvrocks.com/v1alpha1
kind: KVRocks
metadata:
   finalizers:
      - kvrocks/finalizer
   labels:
      kvrocks/system: xxx
   name: kvrocks-cluster-1-demo
   namespace: kvrocks
spec:
   enableSentinel: true
   image: xxxx
   imagePullPolicy: IfNotPresent
   kvrocksConfig:
      auto-resize-block-and-sst: "yes"
      bind: 0.0.0.0
      daemonize: "no"
      db-name: change.me.db
      max-backup-keep-hours: "24"
      max-backup-to-keep: "1"
      max-db-size: "0"
      max-io-mb: "500"
      maxclients: "10000"
      port: "6379"
      profiling-sample-ratio: "0"
      profiling-sample-record-max-len: "256"
      profiling-sample-record-threshold-ms: "100"
      purge-backup-on-fullsync: "no"
      rocksdb.compression: "no"
      rocksdb.wal_size_limit_mb: "0"
      rocksdb.wal_ttl_seconds: "0"
      slave-empty-db-before-fullsync: "no"
      slave-priority: "100"
      slave-read-only: "yes"
      slave-serve-stale-data: "yes"
      slowlog-log-slower-than: "200000"
      supervised: "no"
      tcp-backlog: "512"
      timeout: "0"
      workers: "8"
   master: 3
   password: "123456"
   replicas: 3
   resources:
      limits:
         cpu: "2"
         memory: 8Gi
      requests:
         cpu: "1"
         memory: 4Gi
   storage:
      class: openebs-hostpath
      size: 32Gi
   toleration:
      - effect: NoSchedule
        key: kvrocks
        operator: Exists
      - effect: NoSchedule
        key: node
        operator: Exists
   type: cluster
status:
   status: Running
   topo:
      - partitionName: kvrocks-cluster-1-demo-0
        topology:
           - ip: 10.0.59.4
             nodeId: fc3e4218iz4c7btt4ce1u8aa33oqe54bc4adf111
             pod: kvrocks-cluster-1-demo-0-0
             port: 6379
             role: master
             slots:
                - 0-5461
           - ip: 10.0.17.175
             masterId: fc3e4218iz4c7btt4ce1u8aa33oqe54bc4adf111
             nodeId: 8fd06c8ciza7b0tt46afu88c0foq750fe2c753a1
             pod: kvrocks-cluster-1-demo-0-1
             port: 6379
             role: slave
           - ip: 10.0.49.158
             masterId: fc3e4218iz4c7btt4ce1u8aa33oqe54bc4adf111
             nodeId: c30254e4iz9385tt4161u88936oqd1c6b3ba23c6
             pod: kvrocks-cluster-1-demo-0-2
             port: 6379
             role: slave
      - partitionName: kvrocks-cluster-1-demo-1
        topology:
           - ip: 10.0.51.164
             nodeId: 50ea58f5iza091tt4450u8b64aoq9e5245769d2f
             pod: kvrocks-cluster-1-demo-1-0
             port: 6379
             role: master
             slots:
                - 5462-10922
           - ip: 10.0.50.150
             masterId: 50ea58f5iza091tt4450u8b64aoq9e5245769d2f
             nodeId: c59f1d1dize97btt4e71u8b0d8oqeaef5db17323
             pod: kvrocks-cluster-1-demo-1-1
             port: 6379
             role: slave
           - ip: 10.0.16.247
             masterId: 50ea58f5iza091tt4450u8b64aoq9e5245769d2f
             nodeId: 08803e14izc435tt4593u898b3oq64a91bf1db39
             pod: kvrocks-cluster-1-demo-1-2
             port: 6379
             role: slave
      - partitionName: kvrocks-cluster-1-demo-2
        topology:
           - ip: 10.0.59.37
             nodeId: 4cd4fb6dizb9f4tt4bc1u89d8boqd42829f9ee29
             pod: kvrocks-cluster-1-demo-2-0
             port: 6379
             role: master
             slots:
                - 10923-16383
           - ip: 10.0.50.69
             masterId: 4cd4fb6dizb9f4tt4bc1u89d8boqd42829f9ee29
             nodeId: f3d2405biz74b9tt45f5u8a79doq1f7566febdf2
             pod: kvrocks-cluster-1-demo-2-1
             port: 6379
             role: slave
           - ip: 10.0.48.248
             masterId: 4cd4fb6dizb9f4tt4bc1u89d8boqd42829f9ee29
             nodeId: 81df179fiz33fftt44deu88262oq8b1d5e4ef22a
             pod: kvrocks-cluster-1-demo-2-2
             port: 6379
             role: slave
   version: 1
```

1. spec.type: Specifies the type of KVRocks created, mainly including cluster mode, standard mode and sentinel creation.
2. spec.storage is used to specify the stroageclass for creating pv. If it is changed to be empty, the mounted /data directory uses empty dir.
3. status.version is used to record the current cluster version, which is used for cluster expansion and contraction.
4. status.topo records the cluster topo information corresponding to each pod in the cluster mode

## Function Introduction

1. Support sentinel mode and cluster mode
2. Support automatic deployment and operation and maintenance
3. Cluster mode supports fault detection and recovery

## Design

### Sentinel

![avatar](https://github.com/RocksLabs/kvrocks-operator/blob/unstable/images/sentinel.png)

1. Use statefulSet to deploy sentinel Pod
2. For scaling, just modify the spec.replicas field

#### Expansion

1. Add the spec.replicas field, but the number of replicas must be an odd number after the increase
2. The new sentinel copy will automatically add the master information that the current sentinel has monitored
3. Modify the number of quorum to (number of copies/2)+1

#### Shrink

1. Reduce the spec.replicas field, but the number of replicas after shrinking must be an odd number and must be greater than or equal to 3
2. Modify the number of quorum to (number of copies/2)+1

#### Delete

1. If sentinel still has a master to monitor, it is not allowed to delete the sentinel cluster.

#### Fault Detection Recovery

1. When an event of sentinel type is received, the following steps will be performed
   - Determine whether the pods of the statefulSet are all in the Running state, if not, wait
   - The operator starts a coroutine subscription +odown message for each sentinel for cluster mode failure detection and recovery
   - Detect all pods with the label sentinel=xxx (xxx is the name of the current sentinel cluster), and add monitoring if the master ip changes or is not monitored.

### Standard

![avatar](https://github.com/RocksLabs/kvrocks-operator/blob/unstable/images/standard.png)

1. Use statefulSet to deploy kvrocks pod
2. Sentinel is used to monitor kvrocks master-slave mode, and perform failover and discovery

#### Delete

1. Clear sentinel's monitoring of the master before deleting the kvrocks instance
2. Delete the kvrocks instance


#### Fault Detection Recovery

1. When an event of standard type is received, the following steps will be executed
   - Detect whether the pods of the statefulSet are all in the Running state, if not waiting
   - newly created kvrocks instance, slaveof myself on startup to make it a slave
   - newly created kvrocks instance slaveof current master
   - sentinel Check whether the monitoring information is correct, delete the old monitoring information incorrectly, and create a new one

### Cluster

![avatar](https://github.com/RocksLabs/kvrocks-operator/blob/unstable/images/cluster.png)

1. Use statefulSet to control pods
2. Sentinel is used to monitor the cluster
   - Sentinel monitors all master nodes in the cluster
   - operator subscribes to +odown messages, responding to failover
   - Refresh the cluster topology after receiving the message
3. storageClass is used for persistent storage of cluster data.

#### Expansion

1. You can expand the master by increasing the partition, or you can modify spec.replicas to increase the number of replicas.
2. The expansion steps are as follows
   - matser increase
      - Refresh the cluster topo structure for each node of the cluster (the new master slot is initialized to null)
      - Perform slot migration
         - Run clusterx migrate $slot $node_id for slot data migration
         - Run clustex setslot $slot NODE $node_id $new_version to refresh the cluster topo
   - spec.replicas increase
      - Refresh the new topo structure for each node of the cluster

#### Shrink

1. The operation steps are the same as the expansion operation

#### Delete
1. Before deleting the kvrocks cluster, clear sentinl's monitoring of all masters of the cluster
2. Delete the kvrocks cluster instance

#### Fault Detection Recovery

1. After receiving an event of cluster type, perform the following steps
   - Detect whether all pods of the statefulSet attached to the cluster mode are in the Running state, and wait if not.
   - Sentinel monitors the master node of each partition
   - Check the cluster information topo relationship for each pod, if it is incorrect, refresh the topo organization
   - If +odown receives the message, it will refresh the cluster topo structure (note that slaveof no one operation is not allowed in cluster mode, that is to say, sentinel will not fail over the partition, but will only detect it)


## Development Guide

**NOTE! The development guide is not yet complete, feel free to ask us any questions through [issues](https://github.com/KvrocksLabs/kvrocks-operator/issues) or [pull requests](https://github.com/KvrocksLabs/kvrocks-operator/pulls).**


Please refer to the [Development Guide](/docs/development.md) for more information.

