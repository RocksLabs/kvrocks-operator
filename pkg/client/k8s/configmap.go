package k8s

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
)

func (c *Client) GetConfigMap(key types.NamespacedName) (*corev1.ConfigMap, error) {
	var cm corev1.ConfigMap
	if err := c.client.Get(ctx, key, &cm); err != nil {
		return nil, err
	}
	return &cm, nil
}

func (c *Client) UpdateConfigMap(cm *corev1.ConfigMap) error {
	if err := c.client.Update(ctx, cm); err != nil {
		return err
	}
	c.logger.V(1).Info("configMap update successfully", "configMap", cm.Name)
	return nil
}

func (c *Client) CreateOrUpdateConfigMap(cm *corev1.ConfigMap) error {
	oldCM, err := c.GetConfigMap(types.NamespacedName{
		Namespace: cm.Namespace,
		Name:      cm.Name,
	})
	if err != nil {
		if errors.IsNotFound(err) {
			return c.CreateIfNotExistsConfigMap(cm)
		}
		return err
	}
	// Already exists, need to Update.
	cm.ResourceVersion = oldCM.ResourceVersion
	return c.UpdateConfigMap(cm)
}

func (c *Client) CreateIfNotExistsConfigMap(cm *corev1.ConfigMap) error {
	if err := c.client.Create(ctx, cm); err != nil && !errors.IsAlreadyExists(err) {
		return err
	}
	c.logger.V(1).Info("configMap create successfully", "configMap", cm.Name)
	return nil
}
