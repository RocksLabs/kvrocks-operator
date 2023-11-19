package k8s

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	k8sApiClient "sigs.k8s.io/controller-runtime/pkg/client"
)

func (c *Client) GetPod(key types.NamespacedName) (*corev1.Pod, error) {
	var pod corev1.Pod
	if err := c.client.Get(ctx, key, &pod); err != nil {
		return nil, err
	}
	return &pod, nil
}

func (c *Client) UpdatePod(pod *corev1.Pod) error {
	if err := c.client.Update(ctx, pod); err != nil {
		return err
	}
	c.logger.V(1).Info("update pod successfully", "pod", pod.Name)
	return nil
}

func (c *Client) DeletePodImmediately(podName, namespace string) error {
	pod, err := c.GetPod(types.NamespacedName{
		Namespace: namespace,
		Name:      podName,
	})
	if err != nil && !errors.IsNotFound(err) {
		return err
	}
	if err = c.client.Delete(ctx, pod, k8sApiClient.GracePeriodSeconds(0)); err != nil && !errors.IsNotFound(err) {
		return err
	}
	c.logger.V(1).Info("delete pod successfully", "pod", pod.Name)
	return nil
}
