package k8s

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
)

func (c *Client) CreateIfNotExistsService(service *corev1.Service) error {
	if err := c.client.Create(ctx, service); err != nil && !errors.IsAlreadyExists(err) {
		return err
	}
	c.logger.V(1).Info("service create successfully", "service", service.Name)
	return nil
}

func (c *Client) GetService(key types.NamespacedName) (*corev1.Service, error) {
	var service corev1.Service
	if err := c.client.Get(ctx, key, &service); err != nil {
		return nil, err
	}
	return &service, nil
}

func (c *Client) DeleteService(key types.NamespacedName) error {
	var service corev1.Service
	if err := c.client.Get(ctx, key, &service); err != nil {
		return err
	}
	if err := c.client.Delete(ctx, &service); err != nil {
		return err
	}
	c.logger.V(1).Info("service delete successfully", "service", service.Name)
	return nil
}
