package common

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"

	kvrocksv1alpha1 "github.com/KvrocksLabs/kvrocks-operator/api/v1alpha1"
	"github.com/KvrocksLabs/kvrocks-operator/pkg/resources"
)

func (h *CommandHandler) EnsureSentinel(masterIP string, index ...int) (bool, error) {
	sentinelPods, sentinelPassword, requeue, err := h.GetSentinel()
	if err != nil || requeue {
		return requeue, err
	}
	_, masterName := resources.ParseRedisName(h.instance.Name)
	if len(index) != 0 {
		masterName = fmt.Sprintf("%s-%d", masterName, index[0])
	}
	for _, sentinel := range sentinelPods.Items {
		master, err := h.kvrocks.GetMasterFromSentinel(sentinel.Status.PodIP, *sentinelPassword, masterName)
		if err != nil || master != masterIP {
			h.kvrocks.RemoveMonitor(sentinel.Status.PodIP, *sentinelPassword, masterName)
			if err = h.kvrocks.CreateMonitor(sentinel.Status.PodIP, *sentinelPassword, masterName, masterIP, h.password); err != nil {
				return false, err
			}
		} else { // if password is changed
			if err = h.kvrocks.ResetMonitor(sentinel.Status.PodIP, *sentinelPassword, masterName, h.password); err != nil {
				return false, err
			}
		}
	}
	return false, nil
}

func (h *CommandHandler) RemoveMonitor(masterName string, index ...int) (bool, error) {
	sentinelPods, sentinelPassword, requeue, err := h.GetSentinel()
	if err != nil || requeue {
		return requeue, err
	}
	if len(index) != 0 {
		masterName = fmt.Sprintf("%s-%d", masterName, index[0])
	}
	for _, sentinel := range sentinelPods.Items {
		_, err = h.kvrocks.GetMasterFromSentinel(sentinel.Status.PodIP, *sentinelPassword, masterName)
		if err == nil {
			if err = h.kvrocks.RemoveMonitor(sentinel.Status.PodIP, *sentinelPassword, masterName); err != nil {
				return false, err
			}
		}
	}
	return false, nil
}

func (h *CommandHandler) GetSentinel() (*corev1.PodList, *string, bool, error) {
	sentinel := resources.GetSentinelInstance(h.instance)
	if err := h.k8s.CreateIfNotExistsKVRocks(sentinel); err != nil {
		return nil, nil, false, err
	}
	key := types.NamespacedName{
		Namespace: sentinel.Namespace,
		Name:      sentinel.Name,
	}
	sentinel, err := h.k8s.GetKVRocks(key)
	if err != nil || sentinel.Status.Status != kvrocksv1alpha1.StatusRunning {
		return nil, nil, true, err
	}
	sentinelPods, err := h.k8s.ListStatefulSetPods(key)
	if err != nil {
		return nil, nil, false, err
	}
	return sentinelPods, &sentinel.Spec.Password, false, nil
}
