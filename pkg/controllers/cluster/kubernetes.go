package cluster

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"

	kvrocksv1alpha1 "github.com/KvrocksLabs/kvrocks-operator/api/v1alpha1"
	"github.com/KvrocksLabs/kvrocks-operator/pkg/client/kvrocks"
	"github.com/KvrocksLabs/kvrocks-operator/pkg/controllers/common"
	"github.com/KvrocksLabs/kvrocks-operator/pkg/resources"
)

func (h *KVRocksClusterHandler) ensureKubernetes() error {
	cm := resources.NewKVRocksConfigMap(h.instance)
	if err := h.k8s.CreateIfNotExistsConfigMap(cm); err != nil {
		return err
	}
	service := resources.NewKVRocksService(h.instance)
	if err := h.k8s.CreateIfNotExistsService(service); err != nil {
		return err
	}
	oldCM, err := h.k8s.GetConfigMap(h.key)
	if err != nil {
		if errors.IsNotFound(err) {
			h.requeue = true
			return nil
		}
		return err
	}
	h.password = oldCM.Data["password"]
	for i := 0; i < int(h.instance.Spec.Master); i++ {
		sts := resources.NewClusterStatefulSet(h.instance, i)
		if err = h.k8s.CreateIfNotExistsStatefulSet(sts); err != nil {
			return err
		}
	}
	curStsList, err := h.k8s.ListStatefulSets(h.instance.Namespace, resources.SelectorLabels(h.instance))
	if err != nil {
		return err
	}
	if len(curStsList.Items) < int(h.instance.Spec.Master) {
		h.requeue = true
		return nil
	}
	h.stsNodes = make([][]*kvrocks.Node, len(curStsList.Items))
	// scaling up
	for i := 0; i < len(curStsList.Items); i++ {
		sts := resources.NewClusterStatefulSet(h.instance, i)
		key := types.NamespacedName{
			Namespace: h.instance.Namespace,
			Name:      sts.Name,
		}
		oldSts, err := h.k8s.GetStatefulSet(key)
		if err != nil {
			return err
		}
		sts.ResourceVersion = oldSts.ResourceVersion
		delta := *sts.Spec.Replicas - *oldSts.Spec.Replicas
		if delta > 0 {
			reserve := oldSts.Spec.ReserveOrdinals
			for delta > 0 && len(reserve) > 0 {
				reserve = reserve[1:]
				delta--
			}
			sts.Spec.ReserveOrdinals = reserve
			if err = h.k8s.UpdateStatefulSet(sts); err != nil {
				return err
			}
		}
	}
	// init h.stsNode
	for i := 0; i < len(curStsList.Items); i++ {
		key := types.NamespacedName{
			Namespace: h.instance.Namespace,
			Name:      resources.GetStatefulSetName(h.instance.Name, i),
		}
		sts, err := h.k8s.GetStatefulSet(key)
		if err != nil {
			return err
		}
		if sts.Status.ReadyReplicas != *sts.Spec.Replicas {
			h.log.Info("waiting for statefulSet ready", "statefulSet", key.Name)
			h.requeue = true
			return nil
		}
		pods, err := h.k8s.ListStatefulSetPods(key)
		if err != nil {
			return err
		}
		for _, pod := range pods.Items {
			if pod.DeletionTimestamp != nil {
				h.log.Info("pod is deleting,please wait")
				h.requeue = true
				return nil
			}
			podIndex, err := resources.GetPVCOrPodIndex(pod.Name)
			if err != nil {
				return err
			}
			// init topo message
			h.stsNodes[i] = append(h.stsNodes[i], &kvrocks.Node{
				IP:       pod.Status.PodIP,
				PodIndex: podIndex,
			})
		}
		sort.Slice(h.stsNodes[i], func(k, j int) bool {
			return h.stsNodes[i][k].PodIndex < h.stsNodes[i][j].PodIndex
		})
	}
	curInstance, err := h.k8s.GetKVRocks(h.key)
	if err != nil {
		return err
	}
	if h.instance.ResourceVersion != curInstance.ResourceVersion { // wait for topo mesage flush
		h.requeue = true
		return nil
	}
	// fix topo message
	for i, replicationTopo := range h.instance.Status.Topo {
		for j, topo := range replicationTopo.Topology {
			h.stsNodes[i][j].NodeId = topo.NodeId
			h.stsNodes[i][j].Role = topo.Role
			h.stsNodes[i][j].Master = topo.MasterId
			h.stsNodes[i][j].Slots = kvrocks.SlotsToInt(topo.Slots)
			h.stsNodes[i][j].Failover = topo.Failover
			if topo.Migrate != nil {
				for _, migrate := range topo.Migrate {
					h.stsNodes[i][j].Migrate = append(h.stsNodes[i][j].Migrate, kvrocks.MigrateMsg{
						DstNodeID: migrate.DstNode,
						Slots:     kvrocks.SlotsToInt(migrate.Slots),
					})
				}
			}
			if topo.Import != nil {
				for _, im := range topo.Import {
					h.stsNodes[i][j].Import = append(h.stsNodes[i][j].Import, kvrocks.ImportMsg{
						SrcNodeId: im.SrcNode,
						Slots:     kvrocks.SlotsToInt(im.Slots),
					})
				}
			}
		}
	}
	h.version = h.instance.Status.Version
	h.log.Info("kubernetes resources ok")
	return nil
}

func (h *KVRocksClusterHandler) ensureStatusTopoMsg() error {
	h.instance.Status.Topo = nil
	for i, sts := range h.stsNodes {
		if sts == nil {
			continue
		}
		var topoes []kvrocksv1alpha1.KVRocksTopology
		partitionName := resources.GetStatefulSetName(h.instance.Name, i)
		for j, node := range sts {
			if node == nil {
				continue
			}
			topo := kvrocksv1alpha1.KVRocksTopology{
				Pod:      fmt.Sprintf("%s-%d", partitionName, j),
				Role:     node.Role,
				NodeId:   node.NodeId,
				Ip:       node.IP,
				Port:     kvrocks.KVRocksPort,
				Slots:    kvrocks.SlotsToString(node.Slots),
				MasterId: node.Master,
			}
			if node.Migrate != nil {
				for _, migrate := range node.Migrate {
					topo.Migrate = append(topo.Migrate, kvrocksv1alpha1.MigrateMsg{
						DstNode: migrate.DstNodeID,
						Slots:   kvrocks.SlotsToString(migrate.Slots),
					})
				}
			}
			if node.Import != nil {
				for _, im := range node.Import {
					topo.Import = append(topo.Import, kvrocksv1alpha1.ImportMsg{
						SrcNode: im.SrcNodeId,
						Slots:   kvrocks.SlotsToString(im.Slots),
					})
				}
			}
			topoes = append(topoes, topo)
		}
		h.instance.Status.Topo = append(h.instance.Status.Topo, kvrocksv1alpha1.KVRocksTopoPartitions{
			PartitionName: partitionName,
			Topology:      topoes,
		})
	}
	h.instance.Status.Version = h.version
	if err := h.k8s.UpdateKVRocks(h.instance); err != nil {
		h.requeue = true
		return err
	}
	return nil
}

// shrink
// 1 ensure topo message
// delete statefulSet

func (h *KVRocksClusterHandler) ensureShrink() error {
	if h.instance.Status.Rebalance {
		return nil
	}
	var shrinkIndex []int
	var err error
	commHandler := common.NewCommandHandler(h.instance, h.k8s, h.kvrocks, h.password)
	_, masterName := resources.ParseRedisName(h.instance.Name)
	for i := int(h.instance.Spec.Master); i < len(h.stsNodes); i++ {
		// first remove sentinel monitor
		h.requeue, err = commHandler.RemoveMonitor(masterName, i)
		if err != nil {
			return err
		}
		h.stsNodes[i] = nil
		shrinkIndex = append(shrinkIndex, i)
	}
	reserves := make(map[string][]int)

	for i := 0; i < int(h.instance.Spec.Master); i++ {
		reserve := h.getReserveIndex(h.stsNodes[i])
		if reserve != nil {
			reserves[resources.GetStatefulSetName(h.instance.Name, i)] = reserve
		}
	}
	if len(shrinkIndex) == 0 && len(reserves) == 0 {
		return nil
	}
	h.instance.Status.Shrink = &kvrocksv1alpha1.KVRocksShrinkMsg{Partition: shrinkIndex, ReserveMsg: reserves}
	return h.ensureStatusTopoMsg()
}

func (h *KVRocksClusterHandler) cleanStatefulSet() error {
	for _, index := range h.instance.Status.Shrink.Partition {
		if err := h.k8s.DeleteStatefulSetIfExists(types.NamespacedName{
			Namespace: h.instance.Namespace,
			Name:      resources.GetStatefulSetName(h.instance.Name, index),
		}); err != nil {
			return err
		}
	}
	for stsName, reserve := range h.instance.Status.Shrink.ReserveMsg {
		sts, err := h.k8s.GetStatefulSet(types.NamespacedName{
			Namespace: h.instance.Namespace,
			Name:      stsName,
		})
		if err != nil {
			return err
		}
		sts.Spec.Replicas = &h.instance.Spec.Replicas
		if len(reserve) != 0 {
			sts.Spec.ReserveOrdinals = append(sts.Spec.ReserveOrdinals, reserve...)
		}
		if err := h.k8s.UpdateStatefulSet(sts); err != nil {
			return err
		}
	}
	h.instance.Status.Shrink = nil
	h.instance.Status.Version++
	return h.k8s.UpdateKVRocks(h.instance)
}

func (h *KVRocksClusterHandler) getReserveIndex(nodes []*kvrocks.Node) []int {
	delta := len(nodes) - int(h.instance.Spec.Replicas)
	var result []int
	if delta > 0 {
		result = []int{}
		var reserve []int
		var masterID int
		for j := len(nodes) - 1; j >= 0; j-- {
			if delta > 0 {
				if nodes[j].Role != kvrocks.RoleMaster {
					reserve = append(reserve, nodes[j].PodIndex)
					delta--
				} else {
					masterID = nodes[j].PodIndex
				}
			}
			if delta == 0 {
				break
			}
		}
		for j := len(reserve) - 1; j >= 0; j-- {
			id := reserve[j]
			nodes[id] = nil
			if id > masterID {
				continue
			}
			result = append(result, id)
		}
	}
	return result
}

func (h *KVRocksClusterHandler) cleanPersistentVolumeClaim() error {
	if h.instance.Status.Shrink != nil {
		return nil
	}
	pvcList, err := h.k8s.ListPVC(h.instance.Namespace, resources.SelectorLabels(h.instance))
	if err != nil {
		return err
	}
	for _, pvc := range pvcList.Items {
		remove := false
		fields := strings.Split(pvc.Name, "-")
		stsIdx, _ := strconv.Atoi(fields[len(fields)-2])
		podIdx, _ := strconv.Atoi(fields[len(fields)-1])
		if stsIdx >= int(h.instance.Spec.Master) {
			remove = true
		} else {
			pos := sort.Search(len(h.stsNodes[stsIdx]), func(i int) bool {
				return h.stsNodes[stsIdx][i].PodIndex >= podIdx
			})
			if pos >= len(h.stsNodes[stsIdx]) || h.stsNodes[stsIdx][pos].PodIndex != podIdx {
				remove = true
			}
		}
		if remove {
			if err = h.k8s.DeletePVC(&pvc); err != nil {
				return err
			}
		}
	}
	return nil
}
