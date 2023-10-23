# Design

## Sentinel

<img alt="avatar" src="/docs/images/sentinel.png" width="50%"/>

1. Use deployment to deploy sentinel Pod
2. For scaling, just modify the spec.replicas field

### Expansion

1. Add the `spec.replicas` field, but the number of replicas must be an odd number after the increase
2. The new sentinel copy will automatically add the master information that the current sentinel has monitored
3. Modify the number of quorum to (number of copies/2)+1

### Shrink

1. Reduce the spec.replicas field, but the number of replicas after shrinking must be an odd number and must be greater
   than or equal to 3
2. Modify the number of quorum to (number of copies/2)+1

### Fault Detection Recovery

1. When an event of sentinel type is received, the following steps will be performed
    - Determine whether the pods of the deployment are all in the Running state, if not, wait
    - The operator starts a coroutine subscription +odown message for each sentinel for cluster mode failure detection
      and recovery
    - Detect all pods with the label sentinel=xxx (xxx is the name of the current sentinel cluster), and add monitoring
      if the master ip changes or is not monitored.

## Standard

<img src="/docs/images/standard.png" width="50%" />

1. Use statefulSet to deploy kvrocks pod
2. Sentinel is used to monitor kvrocks master-slave mode, and perform failover and discovery

### Delete

1. Clear sentinel's monitoring of the master before deleting the kvrocks instance
2. Delete the kvrocks instance

### Fault Detection Recovery

1. When an event of standard type is received, the following steps will be executed
    - Detect whether the pods of the statefulSet are all in the Running state, if not waiting
    - newly created kvrocks instance, slaveof myself on startup to make it a slave
    - newly created kvrocks instance slaveof current master
    - sentinel Check whether the monitoring information is correct, delete the old monitoring information incorrectly,
      and create a new one

## Cluster

<img src="/docs/images/cluster.png" width="50%">

### Apache Kvrocks Controller

[Apache Kvrocks Controller](https://github.com/apache/kvrocks-controller) is a cluster management tool for Apache
Kvrocks.

1. Deploy the ETCD pod using a statefulSet. ETCD stores the cluster information.
2. Deploy the Apache Kvrocks Controller pod using a deployment.

### Apache Kvrocks Cluster

1. Deploy the Apache Kvrocks Cluster pod using a statefulSet. Each statefulSet represents a shard of the cluster.
2. Sentinel monitors the kvrocks master-slave mode, facilitating failover and discovery.

### Expansion

#### Expand Shard
1. Add the `spec.master` field. However, the resulting number of replicas must be odd.
2. The new shard will be added automatically, but the slots won't be rebalanced.

#### Expand Nodes
1. Add the `spec.replicas` field. However, the resulting number of replicas must be odd.
2. The new node will join based on the shard, and the slots will be synchronized.

### Shrink

#### Shrink Shard
1. **Important:** Before shrinking the shard, manually migrate the slots to other shards.
2. Reduce the `spec.master` field. The resulting number of replicas must be an odd number and at least 3.
3. The shard with the highest ID (or number) without slots will be deleted.


#### Shrink Nodes
1. Reduce the `spec.replicas` field. The resulting number of replicas must be an odd number and at least 1.
2. Nodes with a `slave` role will be deleted.

### Migration
1. Use kubectl edit kvrocks xxxx to modify the kvrocks cluster.
2. Add the content below to the master node of the shard you wish to migrate, and then save it.
```yaml
migrate:
   - shard: 1   # the destination shard
     slots:
        - "1-2" # the slots to migrate
        - "300"
```
