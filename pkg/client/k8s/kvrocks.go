package k8s

import (
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kvrocksv1alpha1 "github.com/RocksLabs/kvrocks-operator/api/v1alpha1"
)

func (c *Client) GetKVRocks(key types.NamespacedName) (*kvrocksv1alpha1.KVRocks, error) {
	var kvrocks kvrocksv1alpha1.KVRocks
	if err := c.client.Get(ctx, key, &kvrocks); err != nil {
		return nil, err
	}
	return &kvrocks, nil
}

func (c *Client) UpdateKVRocks(instance *kvrocksv1alpha1.KVRocks) error {
	if err := c.client.Update(ctx, instance); err != nil {
		return err
	}
	c.logger.V(1).Info("update kvrocks successfully")
	return nil
}

func (c *Client) ListKVRocks(namespace string, labels map[string]string) (*kvrocksv1alpha1.KVRocksList, error) {
	var kvrockses kvrocksv1alpha1.KVRocksList
	if err := c.client.List(ctx, &kvrockses, client.InNamespace(namespace), client.MatchingLabels(labels)); err != nil {
		return nil, err
	}
	return &kvrockses, nil
}

func (c *Client) CreateIfNotExistsKVRocks(instance *kvrocksv1alpha1.KVRocks) error {
	if err := c.client.Create(ctx, instance); err != nil && !errors.IsAlreadyExists(err) {
		return err
	}
	c.logger.V(1).Info("kvrocks create successfully")
	return nil
}
