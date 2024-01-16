package sentinel

import (
	"strconv"

	"github.com/RocksLabs/kvrocks-operator/pkg/client/k8s"
	"github.com/RocksLabs/kvrocks-operator/pkg/resources"
	"k8s.io/apimachinery/pkg/types"
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
	dep := resources.NewSentinelDeployment(h.instance)
	if err = h.k8s.CreateIfNotExistsDeployment(dep); err != nil {
		return err
	}
	dep, err = h.k8s.GetDeployment(h.key)
	if err != nil {
		return err
	}
	if dep.Status.ReadyReplicas != *dep.Spec.Replicas {
		h.log.Info("please wait deployment ready")
		h.requeue = true
		return nil
	}
	pods, err := h.k8s.ListDeploymentPods(h.key)
	if err != nil {
		return err
	}

	h.pods = []string{}
	for _, pod := range pods.Items {
		h.pods = append(h.pods, pod.Status.PodIP)
	}
	h.log.Info("kubernetes resources ok")
	return nil
}

func UpdateSentinelAnnotationCount(k8s *k8s.Client, namespace, sentinelName string) error {
	sentinel, err := k8s.GetKVRocks(types.NamespacedName{
		Namespace: namespace,
		Name:      sentinelName,
	})
	if err != nil {
		return err
	}
	annotations := sentinel.GetAnnotations()
	if annotations == nil {
		annotations = make(map[string]string)
	}
	count, ok := annotations["change-count"]
	if !ok {
		count = "0"
	}
	countInt, err := strconv.Atoi(count)
	if err != nil {
		return err
	}
	countInt++
	annotations["change-count"] = strconv.Itoa(countInt)
	sentinel.SetAnnotations(annotations)
	if err := k8s.UpdateKVRocks(sentinel); err != nil {
		return err
	}
	return nil
}
