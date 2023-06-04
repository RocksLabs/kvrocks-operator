package sentinel

import (
	"github.com/RocksLabs/kvrocks-operator/pkg/resources"
)

func (h *KVRocksSentinelHandler) ensureKubernetes() error {
	cm := resources.NewSentinelConfigMap(h.instance)
	err := h.k8s.CreateOrUpdateConfigMap(cm)
	if err != nil {
		return err
	}
	service := resources.NewSentinelService(h.instance)
	if err = h.k8s.CreateIfNotExistsService(service); err != nil {
		return err
	}
	sts := resources.NewSentinelStatefulSet(h.instance)
	if err = h.k8s.CreateIfNotExistsStatefulSet(sts); err != nil {
		return err
	}
	sts, err = h.k8s.GetStatefulSet(h.key)
	if err != nil {
		return err
	}
	if sts.Status.ReadyReplicas != *sts.Spec.Replicas {
		h.log.Info("please wait statefulSet ready")
		h.requeue = true
		return nil
	}
	pods, err := h.k8s.ListStatefulSetPods(h.key)
	if err != nil {
		return err
	}

	for _, pod := range pods.Items {
		h.pods[pod.Status.PodIP] = struct{}{}
	}
	h.log.Info("kubernetes resources ok")
	return nil
}
