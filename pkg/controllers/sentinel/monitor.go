package sentinel

import (
	"errors"
	"fmt"

	"k8s.io/apimachinery/pkg/types"

	kvrocksv1alpha1 "github.com/RocksLabs/kvrocks-operator/api/v1alpha1"
	kv "github.com/RocksLabs/kvrocks-operator/pkg/client/kvrocks"
	"github.com/RocksLabs/kvrocks-operator/pkg/resources"
)

func (h *KVRocksSentinelHandler) ensureSentinel() error {
	kvrockses, err := h.k8s.ListKVRocks(h.key.Namespace, resources.MonitorLabels(h.key.Name))
	if err != nil {
		return err
	}
	for _, kvrocks := range kvrockses.Items {
		if _, ok := h.instance.Labels[resources.MonitoredBy]; !ok {
			continue
		}
		if kvrocks.Status.Status != kvrocksv1alpha1.StatusRunning {
			h.requeue = true
			continue
		}
		password := kvrocks.Spec.Password
		_, name := resources.ParseRedisName(kvrocks.Name)
		if kvrocks.Spec.Type == kvrocksv1alpha1.StandardType {
			key := types.NamespacedName{
				Namespace: kvrocks.Namespace,
				Name:      kvrocks.Name,
			}
			node, err := h.getMasterMsg(key, password)
			if err != nil {
				return err
			}
			return h.ensureMonitor(node.IP, name, password)
		} else { // cluster type
			for index := 0; index < int(kvrocks.Spec.Master); index++ {
				key := types.NamespacedName{
					Namespace: kvrocks.Namespace,
					Name:      fmt.Sprintf("%s-%d", kvrocks.Name, index),
				}
				masterName := fmt.Sprintf("%s-%d", name, index)
				node, err := h.getMasterMsg(key, password)
				if err != nil {
					return err
				}
				if err = h.ensureMonitor(node.IP, masterName, password); err != nil {
					return err
				}
			}
		}
	}
	h.log.Info("sentinel status ok")
	return nil
}

func (h *KVRocksSentinelHandler) getMasterMsg(key types.NamespacedName, password string) (*kv.Node, error) {
	pods, err := h.k8s.ListStatefulSetPods(key)
	if err != nil {
		return nil, err
	}
	for _, pod := range pods.Items {
		node, err := h.kvrocks.NodeInfo(pod.Status.PodIP, password)
		if err != nil {
			return nil, err
		}
		if node.Role == kv.RoleMaster {
			return &node, nil
		}
	}
	return nil, errors.New("master not found")
}

func (h *KVRocksSentinelHandler) ensureMonitor(masterIP, masterName, password string) error {
	sentinelPassword := h.instance.Spec.Password
	for _, sentinelIP := range h.pods {
		master, err := h.kvrocks.GetMasterFromSentinel(sentinelIP, sentinelPassword, masterName)
		if err != nil || master != masterIP {
			h.kvrocks.RemoveMonitor(sentinelIP, sentinelPassword, masterName)
			if err := h.kvrocks.CreateMonitor(sentinelIP, sentinelPassword, masterName, masterIP, password); err != nil {
				return err
			}
		}
	}
	h.log.Info("sentinel monitor ok", "master", masterName)
	return nil
}
