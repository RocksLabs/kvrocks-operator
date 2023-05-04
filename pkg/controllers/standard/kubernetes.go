package standard

import (
	"sort"

	"k8s.io/apimachinery/pkg/api/errors"

	"github.com/RocksLabs/kvrocks-operator/pkg/client/kvrocks"
	"github.com/RocksLabs/kvrocks-operator/pkg/resources"
)

func (h *KVRocksStandardHandler) ensureKubernetes() error {
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
	sts := resources.NewReplicationStatefulSet(h.instance)
	if err = h.k8s.CreateIfNotExistsStatefulSet(sts); err != nil {
		return err
	}
	oldSts, err := h.k8s.GetStatefulSet(h.key)
	if err != nil {
		if errors.IsNotFound(err) {
			h.requeue = true
			return nil
		}
		return err
	}
	sts.ResourceVersion = oldSts.ResourceVersion
	// we can resize statefulSet directly if scaling up
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
		oldSts = sts
	}
	if oldSts.Status.ReadyReplicas != *oldSts.Spec.Replicas {
		h.log.Info("waiting for statefulSet ready")
		h.requeue = true
		return nil
	}
	pods, err := h.k8s.ListStatefulSetPods(h.key)
	if err != nil {
		return err
	}
	for _, pod := range pods.Items {
		node, err := h.kvrocks.NodeInfo(pod.Status.PodIP, h.password)
		if err != nil {
			return err
		}
		index, err := resources.GetPVCOrPodIndex(pod.Name)
		node.PodIndex = index
		h.stsNodes = append(h.stsNodes, &node)
	}
	sort.Slice(h.stsNodes, func(i, j int) bool {
		return h.stsNodes[i].PodIndex < h.stsNodes[j].PodIndex
	})
	h.log.Info("kubernetes resources ok")
	return nil
}

func (h *KVRocksStandardHandler) resizeStatefulSet() error {
	delta := len(h.stsNodes) - int(h.instance.Spec.Replicas)
	// scaling down,delete slave node
	if delta > 0 {
		sts := resources.NewReplicationStatefulSet(h.instance)
		reserve := make([]int, 0)
		masterID := 0
		for i := len(h.stsNodes) - 1; i >= 0; i-- {
			if delta > 0 {
				if h.stsNodes[i].Role != kvrocks.RoleMaster {
					reserve = append(reserve, h.stsNodes[i].PodIndex)
					delta--
				} else {
					masterID = h.stsNodes[i].PodIndex
				}
			}
			if delta == 0 {
				break
			}
		}
		for i := len(reserve) - 1; i >= 0; i-- {
			id := reserve[i]
			// no need to reserve ordinal larger than masterID
			if id > masterID {
				break
			}
			sts.Spec.ReserveOrdinals = append(sts.Spec.ReserveOrdinals, id)
		}
		h.log.WithValues("from", len(h.stsNodes), "to", h.instance.Spec.Replicas, "reserve", sts.Spec.ReserveOrdinals).Info("scaling down")
		if err := h.k8s.CreateOrUpdateStatefulSet(sts); err != nil {
			return err
		}
		h.requeue = true
	}
	return nil
}

func (h *KVRocksStandardHandler) cleanPersistentVolumeClaim() error {
	pvcList, err := h.k8s.ListStatefulSetPVC(h.key)
	if err != nil {
		return err
	}
	exitsPod := map[int]struct{}{}
	for _, node := range h.stsNodes {
		exitsPod[node.PodIndex] = struct{}{}
	}
	for _, pvc := range pvcList.Items {
		index, err := resources.GetPVCOrPodIndex(pvc.Name)
		if err != nil {
			return err
		}
		if _, ok := exitsPod[index]; !ok {
			if err = h.k8s.DeletePVC(&pvc); err != nil {
				return err
			}
		}
	}
	return nil
}
