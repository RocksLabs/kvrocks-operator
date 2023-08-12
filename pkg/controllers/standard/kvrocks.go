package standard

import (
	"errors"
	"fmt"
	"strconv"

	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	kvrocksv1alpha1 "github.com/RocksLabs/kvrocks-operator/api/v1alpha1"
	"github.com/RocksLabs/kvrocks-operator/pkg/client/kvrocks"
	"github.com/RocksLabs/kvrocks-operator/pkg/controllers/common"
	"github.com/RocksLabs/kvrocks-operator/pkg/resources"
)

func (h *KVRocksStandardHandler) ensureKVRocksStatus() error {
	err := h.ensureKVRocksConfig()
	if err != nil || h.requeue {
		return err
	}
	err = h.ensureKVRocksReplication()
	if err != nil || h.requeue {
		return err
	}
	h.log.Info("kvrocks status ok")
	return nil
}

func (h *KVRocksStandardHandler) ensureKVRocksConfig() error {
	commHandler := common.NewCommandHandler(h.instance, h.k8s, h.kvrocks, h.password)
	if err := commHandler.EnsureConfig(h.stsNodes); err != nil {
		return err
	}
	h.password = h.instance.Spec.Password
	cm := resources.NewKVRocksConfigMap(h.instance)
	if err := h.k8s.UpdateConfigMap(cm); err != nil {
		return err
	}
	h.log.Info("kvrocks config ok")
	return nil
}

func (h *KVRocksStandardHandler) ensureKVRocksReplication() error {
	masterIP := ""
	if h.instance.Status.Status == kvrocksv1alpha1.StatusCreating {
		for i, node := range h.stsNodes {
			if i == 0 {
				masterIP = node.IP
				if err := h.kvrocks.ChangeMyselfToMaster(node.IP, h.password); err != nil {
					return err
				}
				if err := h.updateKVRocksRole(node.PodIndex, kvrocks.RoleMaster); err != nil {
					return err
				}
				node.Role = kvrocks.RoleMaster
				continue
			}
			if err := h.SlaveOfMaster(node, masterIP); err != nil {
				return err
			}
		}
		h.instance.Status.Status = kvrocksv1alpha1.StatusRunning
		return h.k8s.UpdateKVRocks(h.instance)
	} else {
		for _, node := range h.stsNodes {
			if node.Role == kvrocks.RoleMaster {
				if masterIP == "" {
					masterIP = node.IP
				} else if masterIP != node.IP {
					err := errors.New("more than one master exist")
					h.log.Error(err, "ensure redis replication failed", "master1", masterIP, "master2", node.IP)
					return err
				}
			}
		}
		if masterIP == "" {
			err := errors.New("no master")
			h.log.Error(err, "ensure redis replication failed")
			h.requeue = true
			return nil
		}
		for _, node := range h.stsNodes {
			if node.IP != masterIP {
				if err := h.SlaveOfMaster(node, masterIP); err != nil {
					return err
				}
			} else {
				if err := h.updateKVRocksRole(node.PodIndex, kvrocks.RoleMaster); err != nil {
					return err
				}
			}
		}
	}
	h.log.V(1).Info("kvrocks replication ok")
	// add Finalizer
	if !controllerutil.ContainsFinalizer(h.instance, kvrocksv1alpha1.KVRocksFinalizer) {
		controllerutil.AddFinalizer(h.instance, kvrocksv1alpha1.KVRocksFinalizer)
		if err := h.k8s.UpdateKVRocks(h.instance); err != nil {
			return err
		}
	}
	// notify sentinel to update
	if v, ok := h.instance.Labels[resources.MonitoredBy]; ok {
		return h.updateSentinelAnnotationCount(v)
	}
	return nil
}

func (h *KVRocksStandardHandler) updateSentinelAnnotationCount(sentinelName string) error {
	sentinel, err := h.k8s.GetKVRocks(types.NamespacedName{
		Namespace: h.instance.Namespace,
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
	if err := h.k8s.UpdateKVRocks(sentinel); err != nil {
		return err
	}
	h.log.V(1).Info("sentinel monitor ready")
	return nil
}

func (h *KVRocksStandardHandler) updateKVRocksRole(podID int, role string) error {
	podName := fmt.Sprintf("%s-%d", h.instance.Name, podID)
	pod, err := h.k8s.GetPod(types.NamespacedName{
		Namespace: h.instance.Namespace,
		Name:      podName,
	})
	if err != nil {
		return err
	}
	if pod.Labels[resources.KvrocksRole] != role {
		pod.Labels[resources.KvrocksRole] = role
		return h.k8s.UpdatePod(pod)
	}
	return nil
}

func (h *KVRocksStandardHandler) SlaveOfMaster(node *kvrocks.Node, masterIP string) error {
	curMasterIP, err := h.kvrocks.GetMaster(node.IP, h.password)
	if err != nil {
		return err
	}
	if curMasterIP != masterIP {
		if err = h.kvrocks.SlaveOf(node.IP, masterIP, h.password); err != nil {
			return err
		}
	}
	if err = h.updateKVRocksRole(node.PodIndex, kvrocks.RoleSlaver); err != nil {
		return err
	}
	node.Role = kvrocks.RoleSlaver
	return nil
}
