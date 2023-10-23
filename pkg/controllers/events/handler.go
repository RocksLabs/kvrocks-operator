package events

import (
	"strconv"
	"strings"
	"time"

	"github.com/go-redis/redis/v8"
	"k8s.io/apimachinery/pkg/types"

	kvrocksv1alpha1 "github.com/RocksLabs/kvrocks-operator/api/v1alpha1"
	"github.com/RocksLabs/kvrocks-operator/pkg/client/kvrocks"
	"github.com/RocksLabs/kvrocks-operator/pkg/controllers/common"
	"github.com/RocksLabs/kvrocks-operator/pkg/resources"
)

const ErrNoSuitableSlaver = "ErrNoSuitableSlaver"
const ErrNoMaster = "ErrNoMaster"

func (e *event) sentDownMessage(msg *produceMessage) {
	pubsub, finalize := e.kvrocks.SubOdownMsg(msg.ip, msg.password)
	go func() {
		defer finalize()

		e.listen(pubsub, msg.systemId, msg.key)
	}()
}

func (e *event) listen(pubsub *redis.PubSub, systemId, namespaceName string) {
	ch := pubsub.Channel()
	namespace := strings.Split(namespaceName, "/")[0]
	for msg := range ch {
		fileds := strings.Split(msg.Payload, " ")
		// demo-0 demo
		masterMsg := strings.Split(fileds[1], "-")
		if len(masterMsg) == 1 { // standard type
			continue
		}
		partition, _ := strconv.Atoi(masterMsg[1])
		key := types.NamespacedName{
			Namespace: namespace,
			Name:      resources.GetClusterName(systemId, masterMsg[0]),
		}
		e.lock.Lock()
		e.messages.add(&eventMessage{ip: fileds[2], port: fileds[3], key: key, partition: partition, timeout: time.Now().Add(time.Second * 30)})
		e.log.Info("receive failover message", "master-name", masterMsg)
		e.lock.Unlock()
	}
	// chan done, pusub exits
	e.lock.Lock()
	delete(e.producerSentinels, namespaceName)
	defer e.lock.Unlock()
}

func (e *event) consumer() {
	for msg := range e.messages.message {
		go e.handleFailover(msg)
	}
}

func (e *event) handleFailover(msg *eventMessage) {
	e.lock.Lock()
	defer e.lock.Unlock()
	var err error
	var instance *kvrocksv1alpha1.KVRocks
	requeue := true

	// if handle failover error, msg requeue
	defer func() {
		if err != nil && time.Since(msg.timeout) < 0 && requeue {
			e.messages.message <- msg
			return
		}
		delete(e.messages.keys, msg.ip)
		e.log.Info("failover successfully", "instance", msg.key, "partition", msg.partition)
	}()
	instance, err = e.k8s.GetKVRocks(msg.key)
	if err != nil || (err == nil && instance.Spec.Type != kvrocksv1alpha1.ClusterType) {
		requeue = false
		return
	}
	if instance.DeletionTimestamp != nil || instance.Status.Status == kvrocksv1alpha1.StatusFailed {
		requeue = false
		return
	}
	commHandler := common.NewCommandHandler(instance, e.k8s, e.kvrocks, instance.Spec.Password)

	err = e.controller.SetEndPoint(instance.Namespace, e.k8s)
	if err != nil {
		e.log.Error(err, "set endpoint failed", "instance", msg.key, "partition", msg.partition)
		return
	}

	// handle failover shard
	err = e.controller.FailoverShard(msg.partition)
	if err != nil {
		e.log.Error(err, "failover shard failed", "instance", msg.key, "partition", msg.partition)
		instance.Status = kvrocksv1alpha1.KVRocksStatus{
			Status: kvrocksv1alpha1.StatusFailed,
			Reason: ErrNoSuitableSlaver,
		}
		if err = e.k8s.UpdateKVRocks(instance); err == nil {
			requeue = false
		}
		return
	}

	isMasterFailover := false
	for _, topo := range instance.Status.Topo[msg.partition].Topology {
		if topo.Failover {
			continue
		}
		if topo.Ip == msg.ip {
			if topo.Role == kvrocks.RoleMaster {
				isMasterFailover = true
			}
			break
		}
	}
	if isMasterFailover {
		// sentinel remove monitor
		_, masterName := resources.ParseRedisName(msg.key.Name)
		commHandler.RemoveMonitor(masterName, msg.partition)
	}

	// update topology
	shardData, err := e.controller.GetNodes(msg.partition)
	if err != nil {
		return
	}

	// find new masterID
	masterID := ""
	for _, node := range shardData.Nodes {
		if node.Role == kvrocks.RoleMaster {
			masterID = node.ID
			break
		}
	}
	if masterID == "" {
		e.log.Error(err, "no master found in instance", msg.key, "partition", msg.partition)
		instance.Status = kvrocksv1alpha1.KVRocksStatus{
			Status: kvrocksv1alpha1.StatusFailed,
			Reason: ErrNoMaster,
		}
		if err = e.k8s.UpdateKVRocks(instance); err == nil {
			requeue = false
		}
		return
	}

	for index, topo := range instance.Status.Topo[msg.partition].Topology {
		for _, node := range shardData.Nodes {
			if topo.NodeId == node.ID {
				instance.Status.Topo[msg.partition].Topology[index].Failover = false
				instance.Status.Topo[msg.partition].Topology[index].Role = node.Role
				if node.Role == kvrocks.RoleSlaver {
					instance.Status.Topo[msg.partition].Topology[index].MasterId = masterID
				}
			}
			if topo.Ip == msg.ip {
				instance.Status.Topo[msg.partition].Topology[index].Failover = true
			}
		}
	}
	instance.Status.Version++
	err = e.k8s.UpdateKVRocks(instance)
	if err != nil {
		return
	}
}
