package k8s

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (c *Client) ListStatefulSetPVC(key types.NamespacedName) (*corev1.PersistentVolumeClaimList, error) {
	sts, err := c.GetStatefulSet(key)
	if err != nil {
		return nil, err
	}
	labels := sts.Spec.Selector.MatchLabels
	var pvcList corev1.PersistentVolumeClaimList
	if err = c.client.List(ctx, &pvcList, client.InNamespace(key.Namespace), client.MatchingLabels(labels)); err != nil {
		return nil, err
	}
	return &pvcList, nil
}

func (c *Client) DeletePVC(pvc *corev1.PersistentVolumeClaim) error {
	if err := c.client.Delete(ctx, pvc); err != nil && !errors.IsNotFound(err) {
		return err
	}
	c.logger.V(1).Info("delete pvc successfully", "pvc", pvc.Name)
	return nil
}

func (c *Client) ListPVC(namespace string, labels map[string]string) (*corev1.PersistentVolumeClaimList, error) {
	var pvcList corev1.PersistentVolumeClaimList
	if err := c.client.List(ctx, &pvcList, client.InNamespace(namespace), client.MatchingLabels(labels)); err != nil {
		return nil, err
	}
	return &pvcList, nil
}

func (c *Client) DeletePVCByPod(podName string, namespace string) error {
	var pvc corev1.PersistentVolumeClaim
	if err := c.client.Get(ctx, types.NamespacedName{
		Namespace: namespace,
		Name:      "data-" + podName,
	}, &pvc); err == nil {
		return c.DeletePVC(&pvc)
	}
	c.logger.V(1).Info("delete pvc successfully", "pvc", pvc.Name)
	return nil
}
