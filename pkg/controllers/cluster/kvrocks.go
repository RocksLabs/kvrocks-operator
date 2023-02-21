package cluster

import (
	"fmt"

	"k8s.io/apimachinery/pkg/types"

	kvrocksv1alpha1 "github.com/KvrocksLabs/kvrocks-operator/api/v1alpha1"
	"github.com/KvrocksLabs/kvrocks-operator/pkg/client/kvrocks"
	"github.com/KvrocksLabs/kvrocks-operator/pkg/controllers/common"
	"github.com/KvrocksLabs/kvrocks-operator/pkg/controllers/events"
	"github.com/KvrocksLabs/kvrocks-operator/pkg/resources"
)

func (h *KVRocksClusterHandler) ensureKVRocksStatus() error {
	var err error
	if err = h.ensureKVRocksConfig(); err != nil {
		return err
	}
	if err = h.ensureSetNodeID(); err != nil {
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
	h.ensureVersion()
	if err = h.ensureStatusTopoMsg(); err != nil {
		return err
	}
	commHandler := common.NewCommandHandler(h.instance, h.k8s, h.kvrocks, h.password)
	h.requeue, err = commHandler.EnsureTopo()
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

func (h *KVRocksClusterHandler) ensureSetNodeID() error {
	for _, sts := range h.stsNodes {
		for _, node := range sts {
			if node.NodeId == "" {
				node.NodeId = resources.SetClusterNodeId()
			}
			// pod restart, reset nodeID
			if err := h.kvrocks.SetClusterID(node.IP, h.password, node.NodeId); err != nil {
				return err
			}
		}
	}
	return nil
}

func (h *KVRocksClusterHandler) initCluster() error {
	slotsPreNode := kvrocks.HashSlotCount / h.instance.Spec.Master
	slotsRem := kvrocks.HashSlotCount % h.instance.Spec.Master
	allocated := 0
	for index, sts := range h.stsNodes {
		expected := slotsPreNode
		if index < int(slotsRem) {
			expected++
		}
		slots := make([]int, expected)
		for i := 0; i < int(expected); i++ {
			slots[i] = allocated
			allocated++
		}
		sts[0].Slots = slots
		sts[0].Role = kvrocks.RoleMaster
	}
	for partition, sts := range h.stsNodes {
		for index, node := range sts {
			if index != 0 {
				node.Master = sts[0].NodeId
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
	}
	return h.ensureStatusTopoMsg()
}

func (h *KVRocksClusterHandler) updatePodLabels(key types.NamespacedName, role string) error {
	pod, err := h.k8s.GetPod(key)
	if err != nil {
		return err
	}
	if pod.Labels[resources.RedisRole] != role {
		pod.Labels[resources.RedisRole] = role
		if err = h.k8s.UpdatePod(pod); err != nil {
			return err
		}
	}
	return nil
}
