package cluster

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"k8s.io/apimachinery/pkg/types"

	kvrocksv1alpha1 "github.com/RocksLabs/kvrocks-operator/api/v1alpha1"
	"github.com/RocksLabs/kvrocks-operator/pkg/client/kvrocks"
	"github.com/RocksLabs/kvrocks-operator/pkg/controllers/common"
	"github.com/RocksLabs/kvrocks-operator/pkg/controllers/events"
	"github.com/RocksLabs/kvrocks-operator/pkg/resources"
)

func (h *KVRocksClusterHandler) ensureKVRocksStatus() error {
	var err error
	if err = h.ensureKVRocksConfig(); err != nil {
		return err
	}
	if h.instance.Status.Status == kvrocksv1alpha1.StatusCreating {
		if err = h.initCluster(); err != nil {
			return err
		}
	} else {
		if err = h.ensureReplication(); err != nil {
			return err
		}
	}
	h.requeue, err = h.ensureCluster()
	if err != nil {
		return err
	}
	h.ensureVersion()
	if err = h.ensureStatusTopoMsg(); err != nil {
		return err
	}
	h.version = h.instance.Status.Version
	return err
}

func (h *KVRocksClusterHandler) ensureKVRocksConfig() error {
	commHandler := common.NewCommandHandler(h.instance, h.k8s, h.kvrocks, h.password)
	for _, sts := range h.stsNodes {
		if err := commHandler.EnsureConfig(sts); err != nil {
			return err
		}
	}
	h.password = h.instance.Spec.Password
	configMap := resources.NewKVRocksConfigMap(h.instance)
	if err := h.k8s.UpdateConfigMap(configMap); err != nil {
		return err
	}
	h.log.Info("kvrocks config ready")
	return nil
}

func (h *KVRocksClusterHandler) initCluster() error {
	for _, sts := range h.stsNodes {
		sts[0].Role = kvrocks.RoleMaster
	}
	for partition, sts := range h.stsNodes {
		for index, node := range sts {
			if index != 0 {
				node.Role = kvrocks.RoleSlaver
			}
			key := types.NamespacedName{
				Namespace: h.instance.GetNamespace(),
				Name:      fmt.Sprintf("%s-%d", resources.GetStatefulSetName(h.instance.GetName(), partition), node.PodIndex),
			}
			if err := h.updatePodLabels(key, node.Role); err != nil {
				return err
			}
		}
	}
	return nil
}

func (h *KVRocksClusterHandler) ensureReplication() error {
	for i, sts := range h.stsNodes {
		masterNodeID := ""
		for _, node := range sts {
			if node.Role == kvrocks.RoleMaster {
				masterNodeID = node.NodeId
				break
			}
		}
		if masterNodeID == "" {
			h.stsNodes[i][0].Role = kvrocks.RoleMaster
			masterNodeID = h.stsNodes[i][0].NodeId
		}
		for _, node := range sts {
			if node.NodeId != masterNodeID && node.Master != masterNodeID {
				node.Master = masterNodeID
				node.Role = kvrocks.RoleSlaver
			}
			key := types.NamespacedName{
				Namespace: h.instance.GetNamespace(),
				Name:      fmt.Sprintf("%s-%d", resources.GetStatefulSetName(h.instance.GetName(), i), node.PodIndex),
			}
			if err := h.updatePodLabels(key, node.Role); err != nil {
				return err
			}
		}
	}
	return nil
}

// check version should change
func (h *KVRocksClusterHandler) ensureVersion() {
	if h.instance.Status.Topo == nil {
		goto version
	}
	// 扩容
	if len(h.stsNodes) != len(h.instance.Status.Topo) {
		goto version
	}
	for i, partition := range h.instance.Status.Topo {
		// 扩容
		if len(partition.Topology) != len(h.stsNodes[i]) {
			goto version
		}
		for j, topo := range partition.Topology {
			if topo.Ip != h.stsNodes[i][j].IP { // ip change
				goto version
			}
			if topo.Role != h.stsNodes[i][j].Role { // role change
				goto version
			}
		}
	}
	return
version:
	h.version++
}

// if operator exists, and node down,send failover message
func (h *KVRocksClusterHandler) ensureFailover() error {
	change := false
	for partition, sts := range h.stsNodes {
		for index, node := range sts {
			if node.Failover { // delete pod
				h.requeue = true
				if h.kvrocks.Ping(node.IP, h.password) {
					node.Failover = false
					change = true
					continue
				}
				podName := fmt.Sprintf("%s-%d-%d", h.instance.Name, partition, index)
				if err := h.k8s.DeletePVCByPod(podName, h.instance.Namespace); err != nil {
					return err
				}
				if err := h.k8s.DeletePodImmediately(podName, h.instance.Namespace); err != nil {
					return err
				}
				node.Failover = false
				change = true
				continue
			}
			if !h.kvrocks.Ping(node.IP, h.password) {
				h.requeue = true
				events.SendFailoverMsg(node.IP, h.key, partition)
			}
		}
	}
	if change {
		h.version++
		return h.ensureStatusTopoMsg()
	}
	return nil
}

func (h *KVRocksClusterHandler) updatePodLabels(key types.NamespacedName, role string) error {
	pod, err := h.k8s.GetPod(key)
	if err != nil {
		return err
	}
	if pod.Labels[resources.KvrocksRole] != role {
		pod.Labels[resources.KvrocksRole] = role
		if err = h.k8s.UpdatePod(pod); err != nil {
			return err
		}
	}
	return nil
}

func (h *KVRocksClusterHandler) ensureCluster() (bool, error) {
	nodes := make([]string, 0)
	for _, sts := range h.stsNodes {
		for _, node := range sts {
			nodes = append(nodes, node.IP+":"+strconv.Itoa(kvrocks.KVRocksPort))
		}
	}
	if h.instance.Status.Status == kvrocksv1alpha1.StatusCreating {
		err := h.controllerClient.CreateCluster(int(h.instance.Spec.Replicas), nodes, h.password)
		if err != nil {
			return false, err
		}
	} else {
		err := h.updateCluster()
		if err != nil {
			return false, err
		}
	}
	err := h.ensureSetNodeID()
	if err != nil {
		return false, err
	}
	if h.instance.Status.Status != kvrocksv1alpha1.StatusRunning {
		h.instance.Status.Status = kvrocksv1alpha1.StatusRunning
	}
	if err := h.k8s.UpdateKVRocks(h.instance); err != nil {
		return true, err
	}
	return false, nil
}

func (h *KVRocksClusterHandler) ensureSetNodeID() error {
	for index, sts := range h.stsNodes {
		shardData, err := h.controllerClient.GetNodes(index)
		if err != nil {
			return err
		}
		masterID := ""
		for _, shard := range shardData.Nodes {
			if shard.Role == "master" {
				masterID = shard.ID
				break
			}
		}
		if masterID == "" {
			h.log.V(1).Error(errors.New("master error"), "no master node in shard", "shard", index)
			return fmt.Errorf("no master node in shard %d", index)
		}
		for _, node := range sts {
			for _, shard := range shardData.Nodes {
				if simplyIp(shard.Addr) == node.IP {
					node.NodeId = shard.ID
					if shard.Role == "master" {
						node.Role = kvrocks.RoleMaster
						node.Master = ""
						node.Slots = kvrocks.SlotsToInt(shardData.SlotRanges)
					} else {
						node.Role = kvrocks.RoleSlaver
						node.Master = masterID
						node.Slots = kvrocks.SlotsToInt(shardData.SlotRanges)
					}
					break
				}
			}
			key := types.NamespacedName{
				Namespace: h.instance.GetNamespace(),
				Name:      fmt.Sprintf("%s-%d", resources.GetStatefulSetName(h.instance.GetName(), index), node.PodIndex),
			}
			if err := h.updatePodLabels(key, node.Role); err != nil {
				return err
			}
		}
	}
	return nil
}

func (h *KVRocksClusterHandler) updateCluster() error {
	//remove node
	for index, sts := range h.stsNodes {
		shardData, err := h.controllerClient.GetNodes(index)
		if err != nil {
			return err
		}
		// the shard need to be created
		if shardData == nil {
			nodes := make([]string, 0)
			for _, node := range sts {
				nodes = append(nodes, node.IP+":"+strconv.Itoa(kvrocks.KVRocksPort))
			}
			err := h.controllerClient.CreateShard(nodes, h.password)
			if err != nil {
				return err
			}
			continue
		}
		for _, shard := range shardData.Nodes {
			needDeleted := true
			for _, node := range sts {
				if simplyIp(shard.Addr) == node.IP {
					needDeleted = false
					break
				}
			}
			if needDeleted {
				if err := h.controllerClient.DeleteNode(index, shard.ID); err != nil {
					return err
				}
			}
		}
	}

	// add node
	for index, sts := range h.stsNodes {
		shardData, err := h.controllerClient.GetNodes(index)
		if err != nil {
			return err
		}
		for _, node := range sts {
			needAdded := true
			for _, shard := range shardData.Nodes {
				if simplyIp(shard.Addr) == node.IP {
					needAdded = false
					break
				}
			}
			if needAdded {
				err = h.controllerClient.AddNode(index, node.IP+":"+strconv.Itoa(kvrocks.KVRocksPort), node.Role, h.password)
				if err != nil {
					return err
				}
			}
		}
	}

	// delete shard
	shards, err := h.controllerClient.GetShards()
	if err != nil {
		return err
	}
	// TODO delete shard by any index
	for i := len(shards) - 1; i >= len(h.stsNodes); i-- {
		if err := h.controllerClient.DeleteShard(i); err != nil {
			return err
		}
	}
	return nil
}

func simplyIp(ip string) string {
	return strings.Split(ip, ":")[0]
}
