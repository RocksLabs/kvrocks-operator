# Design

## Sentinel

![avatar](/docs/images/sentinel.png)

1. Use deployment to deploy sentinel Pod
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
   - Determine whether the pods of the deployment are all in the Running state, if not, wait
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

