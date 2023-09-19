package cluster

import (
	"github.com/RocksLabs/kvrocks-operator/pkg/client/kvrocks"
	"github.com/RocksLabs/kvrocks-operator/pkg/resources"
	"k8s.io/apimachinery/pkg/types"
)

// etcd-> controller
// TODO owner reference
func (h *KVRocksClusterHandler) ensureController() error {
	etcdService := resources.NewEtcdService(h.instance)
	if err := h.k8s.CreateIfNotExistsService(etcdService); err != nil {
		return err
	}
	etcd := resources.NewEtcdStatefulSet(h.instance)
	if err := h.k8s.CreateIfNotExistsNativeStatefulSet(etcd); err != nil {
		return err
	}
	// ensure etcd
	etcd, err := h.k8s.GetNativeStatefulSet(types.NamespacedName{
		Namespace: h.instance.Namespace,
		Name:      kvrocks.EtcdStatefulName,
	})
	if err != nil {
		return err
	}
	if etcd.Status.ReadyReplicas != *etcd.Spec.Replicas {
		h.log.Info("waiting for etcd ready")
		h.requeue = true
		return nil
	}

	controllerConfigmap := resources.NewKVRocksControllerConfigmap(h.instance)
	if err := h.k8s.CreateIfNotExistsConfigMap(controllerConfigmap); err != nil {
		return err
	}
	controllerService := resources.NewKVRocksControllerService(h.instance)
	if err := h.k8s.CreateIfNotExistsService(controllerService); err != nil {
		return err
	}
	controllerDep := resources.NewKVRocksControllerDeployment(h.instance)
	if err := h.k8s.CreateIfNotExistsDeployment(controllerDep); err != nil {
		return err
	}
	// ensure controller
	controllerDep, err = h.k8s.GetDeployment(types.NamespacedName{
		Namespace: h.instance.Namespace,
		Name:      kvrocks.ControllerDeploymentName,
	})
	if err != nil {
		return err
	}
	if controllerDep.Status.ReadyReplicas != *controllerDep.Spec.Replicas {
		h.log.Info("waiting for controller ready")
		h.requeue = true
		return nil
	}

	err = h.controllerClient.SetEndPoint(h.instance.Namespace, h.k8s)
	if err != nil {
		return err
	}

	err = h.createControllerNamespace()
	if err != nil {
		return err
	}
	return nil
}

func (h *KVRocksClusterHandler) removeController() (bool, error) {
	// Remove KVRocks Controller resources
	if err := h.k8s.DeleteDeployment(types.NamespacedName{
		Namespace: h.instance.Namespace,
		Name:      kvrocks.ControllerDeploymentName,
	}); err != nil {
		return false, err
	}

	if err := h.k8s.DeleteService(types.NamespacedName{
		Namespace: h.instance.Namespace,
		Name:      kvrocks.ControllerServiceName,
	}); err != nil {
		return false, err
	}

	if err := h.k8s.DeleteConfigMap(types.NamespacedName{
		Namespace: h.instance.Namespace,
		Name:      "controller-config",
	}); err != nil {
		return false, err
	}

	// Remove Etcd resources
	if err := h.k8s.DeleteNativeStatefulSet(types.NamespacedName{
		Namespace: h.instance.Namespace,
		Name:      kvrocks.EtcdStatefulName,
	}); err != nil {
		return false, err
	}

	if err := h.k8s.DeleteService(types.NamespacedName{
		Namespace: h.instance.Namespace,
		Name:      kvrocks.EtcdServiceName,
	}); err != nil {
		return false, err
	}

	return false, nil
}

func (h *KVRocksClusterHandler) createControllerNamespace() error {
	err := h.controllerClient.CreateIfNotExistsNamespace()
	if err != nil {
		return err
	}
	return nil
}
