package events

import (
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"

	kvrocksv1alpha1 "github.com/RocksLabs/kvrocks-operator/api/v1alpha1"
	"github.com/RocksLabs/kvrocks-operator/pkg/resources"
)

func (e *event) producer() {
	sentinels, err := e.k8s.ListKVRocks(corev1.NamespaceAll, resources.SentinelLabels())
	if err != nil {
		return
	}
	for _, sentinel := range sentinels.Items {
		if sentinel.Status.Status != kvrocksv1alpha1.StatusRunning {
			continue
		}
		systemId := strings.Split(sentinel.Name, "-")[1]
		pods, err := e.k8s.ListStatefulSetPods(types.NamespacedName{
			Namespace: sentinel.Namespace,
			Name:      sentinel.Name,
		})
		if err != nil {
			return
		}
		for _, pod := range pods.Items {
			key := types.NamespacedName{
				Namespace: pod.Namespace,
				Name:      pod.Name,
			}
			e.lock.Lock()
			if _, ok := e.producerSentinels[key.String()]; !ok {
				e.producerSentinels[key.String()] = e.sentDownMessage
				e.producerSentinels[key.String()](&produceMessage{
					ip:       pod.Status.PodIP,
					password: sentinel.Spec.Password,
					key:      key.String(),
					systemId: systemId,
				})
			}
			e.lock.Unlock()
		}
	}
}
