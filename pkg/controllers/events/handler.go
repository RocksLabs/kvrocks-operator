package events

import (
	"errors"
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

func (e *event) sentDownMessage(msg *produceMessage) {
	subpub := e.kvrocks.SubOdownMsg(msg.ip, msg.password)
	go e.listen(subpub, msg.systemId, msg.key)
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

	// only one node, set instance fail
	if len(instance.Status.Topo[msg.partition].Topology) == 1 {
		e.log.Error(errors.New(ErrNoSuitableSlaver), "this instance doesn't have slaves")
		instance.Status = kvrocksv1alpha1.KVRocksStatus{
			Status: kvrocksv1alpha1.StatusFailed,
			Reason: ErrNoSuitableSlaver,
		}
		if err = e.k8s.UpdateKVRocks(instance); err == nil {
			requeue = false
		}
		return
	}
	// handle failover
	failover := false
	// slaves ip
	nodeIps := map[string]int{}
	oldMasterIndex := 0
	var newMasterIP *string
	for index, topo := range instance.Status.Topo[msg.partition].Topology {
		if topo.Failover {
			continue
		}
		if topo.Ip == msg.ip {
			if topo.Role == kvrocks.RoleSlaver {
				instance.Status.Topo[msg.partition].Topology[index].Failover = true
				err = e.k8s.UpdateKVRocks(instance)
				return
			}
			failover = true
			oldMasterIndex = index
			continue
		}
		nodeIps[topo.Ip] = index
	}
	if failover {
		// sentinel remove monitor
		_, masterName := resources.ParseRedisName(msg.key.Name)
		commHandler.RemoveMonitor(masterName, msg.partition)
		newMasterIP = e.findNewMaster(nodeIps, instance.Spec.Password)
		// can't find suitable slave
		if newMasterIP == nil {
			//filover timeout
			if time.Since(msg.timeout) >= 0 {
				e.log.Error(errors.New(ErrNoSuitableSlaver), "this instance doesn't have slaves")
				instance.Status = kvrocksv1alpha1.KVRocksStatus{
					Status: kvrocksv1alpha1.StatusFailed,
					Reason: ErrNoSuitableSlaver,
				}
				err = e.k8s.UpdateKVRocks(instance)
			}
			return
		}
		newMasterIndex := nodeIps[*newMasterIP]
		oldMaster := instance.Status.Topo[msg.partition].Topology[oldMasterIndex]
		newMaster := instance.Status.Topo[msg.partition].Topology[newMasterIndex]
		for index, topo := range instance.Status.Topo[msg.partition].Topology {
			if topo.NodeId == newMaster.NodeId {
				instance.Status.Topo[msg.partition].Topology[index] = kvrocksv1alpha1.KVRocksTopology{
					Pod:      newMaster.Pod,
					Role:     kvrocks.RoleMaster,
					NodeId:   newMaster.NodeId,
					Ip:       newMaster.Ip,
					Port:     kvrocks.KVRocksPort,
					Slots:    oldMaster.Slots,
					Migrate:  oldMaster.Migrate,
					Import:   oldMaster.Import,
					Failover: false,
				}
			} else {
				instance.Status.Topo[msg.partition].Topology[index] = kvrocksv1alpha1.KVRocksTopology{
					Role:     kvrocks.RoleSlaver,
					MasterId: newMaster.NodeId,
					Port:     kvrocks.KVRocksPort,
					Pod:      topo.Pod,
					Failover: false,
					NodeId:   topo.NodeId,
					Ip:       topo.Ip,
				}
				if topo.NodeId == oldMaster.NodeId {
					instance.Status.Topo[msg.partition].Topology[index].Failover = true
				}
			}
		}
		instance.Status.Version++
		err = e.k8s.UpdateKVRocks(instance)
		if err != nil {
			return
		}
		_, err = commHandler.EnsureTopo()
	}
}

func (e *event) findNewMaster(ips map[string]int, password string) *string {
	max := 0
	var slaveIp *string
	for ip := range ips {
		offset, err := e.kvrocks.GetOffset(ip, password)
		if err != nil || offset == -1 {
			return nil
		}
		if offset > max {
			max = offset
			slaveIp = &ip
		}
	}
	return slaveIp
}
