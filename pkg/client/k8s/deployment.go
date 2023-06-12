package k8s

import (
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (c *Client) CreateIfNotExistsDeployment(deployment *appsv1.Deployment) error {
	if err := c.client.Create(ctx, deployment); err != nil && !errors.IsAlreadyExists(err) {
		return err
	}
	c.logger.V(1).Info("create deployment successfully", "deployment", deployment.Name)
	return nil
}

func (c *Client) GetDeployment(key types.NamespacedName) (*appsv1.Deployment, error) {
	var deployment appsv1.Deployment
	if err := c.client.Get(ctx, key, &deployment); err != nil {
		return nil, err
	}
	return &deployment, nil
}

func (c *Client) UpdateDeployment(deployment *appsv1.Deployment) error {
	if err := c.client.Update(ctx, deployment); err != nil {
		return err
	}
	c.logger.V(1).Info("update deployment successfully", "deployment", deployment.Name)
	return nil
}

func (c *Client) ListDeploymentPods(key types.NamespacedName) (*corev1.PodList, error) {
	deployment, err := c.GetDeployment(key)
	if err != nil {
		return nil, err
	}
	labels := make(map[string]string)
	for k, v := range deployment.Spec.Selector.MatchLabels {
		labels[k] = v
	}
	var pods corev1.PodList
	if err := c.client.List(ctx, &pods, client.InNamespace(deployment.Namespace), client.MatchingLabels(labels)); err != nil {
		return nil, err
	}
	return &pods, nil
}
