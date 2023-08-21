# Design

## Sentinel

![avatar](/docs/images/sentinel.png)

1. Use statefulSet to deploy sentinel Pod
2. For scaling, just modify the spec.replicas field

### Expansion

1. Add the spec.replicas field, but the number of replicas must be an odd number after the increase
2. The new sentinel copy will automatically add the master information that the current sentinel has monitored
3. Modify the number of quorum to (number of copies/2)+1

### Shrink

1. Reduce the spec.replicas field, but the number of replicas after shrinking must be an odd number and must be greater than or equal to 3
2. Modify the number of quorum to (number of copies/2)+1

### Delete

1. If sentinel still has a master to monitor, it is not allowed to delete the sentinel cluster.

### Fault Detection Recovery

1. When an event of sentinel type is received, the following steps will be performed
   - Determine whether the pods of the statefulSet are all in the Running state, if not, wait
   - The operator starts a coroutine subscription +odown message for each sentinel for cluster mode failure detection and recovery
   - Detect all pods with the label sentinel=xxx (xxx is the name of the current sentinel cluster), and add monitoring if the master ip changes or is not monitored.

## Standard

![avatar](/docs/images/standard.png)

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
   - sentinel Check whether the monitoring information is correct, delete the old monitoring information incorrectly, and create a new one

## Cluster

![avatar](/docs/images/cluster.png)

1. Use statefulSet to control pods
2. Sentinel is used to monitor the cluster
   - Sentinel monitors all master nodes in the cluster
   - operator subscribes to +odown messages, responding to failover
   - Refresh the cluster topology after receiving the message
3. storageClass is used for persistent storage of cluster data.

### Expansion

1. You can expand the master by increasing the partition, or you can modify spec.replicas to increase the number of replicas.
2. The expansion steps are as follows
   - matser increase
      - Refresh the cluster topo structure for each node of the cluster (the new master slot is initialized to null)
      - Perform slot migration
         - Run clusterx migrate $slot $node_id for slot data migration
         - Run clustex setslot $slot NODE $node_id $new_version to refresh the cluster topo
   - spec.replicas increase
      - Refresh the new topo structure for each node of the cluster

### Shrink

1. The operation steps are the same as the expansion operation

### Delete
1. Before deleting the kvrocks cluster, clear sentinl's monitoring of all masters of the cluster
2. Delete the kvrocks cluster instance

### Fault Detection Recovery

1. After receiving an event of cluster type, perform the following steps
   - Detect whether all pods of the statefulSet attached to the cluster mode are in the Running state, and wait if not.
   - Sentinel monitors the master node of each partition
   - Check the cluster information topo relationship for each pod, if it is incorrect, refresh the topo organization
   - If +odown receives the message, it will refresh the cluster topo structure (note that slaveof no one operation is not allowed in cluster mode, that is to say, sentinel will not fail over the partition, but will only detect it)
