package k8s

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
)

func (c *Client) CreateIfNotExistsService(service *corev1.Service) error {
	if err := c.client.Create(ctx, service); err != nil && !errors.IsAlreadyExists(err) {
		return err
	}
	c.logger.V(1).Info("service create successfully", "service", service.Name)
	return nil
}
