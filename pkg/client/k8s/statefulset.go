package k8s

import (
	kruise "github.com/openkruise/kruise-api/apps/v1beta1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (c *Client) CreateIfNotExistsStatefulSet(sts *kruise.StatefulSet) error {
	if err := c.client.Create(ctx, sts); err != nil && !errors.IsAlreadyExists(err) {
		return err
	}
	c.logger.V(1).Info("create statefulSet successfully", "statefulSet", sts.Name)
	return nil
}

func (c *Client) GetStatefulSet(key types.NamespacedName) (*kruise.StatefulSet, error) {
	var sts kruise.StatefulSet
	if err := c.client.Get(ctx, key, &sts); err != nil {
		return nil, err
	}
	return &sts, nil
}

func (c *Client) UpdateStatefulSet(sts *kruise.StatefulSet) error {
	if err := c.client.Update(ctx, sts); err != nil {
		return err
	}
	c.logger.V(1).Info("update statefulSet successfully", "statefulSet", sts.Name)
	return nil
}

func (c *Client) ListStatefulSetPods(key types.NamespacedName) (*corev1.PodList, error) {
	sts, err := c.GetStatefulSet(key)
	if err != nil {
		return nil, err
	}
	labels := make(map[string]string)
	for k, v := range sts.Spec.Selector.MatchLabels {
		labels[k] = v
	}
	var pods corev1.PodList
	if err := c.client.List(ctx, &pods, client.InNamespace(sts.Namespace), client.MatchingLabels(labels)); err != nil {
		return nil, err
	}
	return &pods, nil
}

func (c *Client) CreateOrUpdateStatefulSet(sts *kruise.StatefulSet) error {
	oldSts, err := c.GetStatefulSet(types.NamespacedName{
		Namespace: sts.Namespace,
		Name:      sts.Name,
	})
	if err != nil {
		if errors.IsNotFound(err) {
			return c.CreateIfNotExistsStatefulSet(sts)
		}
		return err
	}
	sts.ResourceVersion = oldSts.ResourceVersion
	return c.UpdateStatefulSet(sts)
}

func (c *Client) ListStatefulSets(namespace string, labels map[string]string) (*kruise.StatefulSetList, error) {
	var stsList kruise.StatefulSetList
	if err := c.client.List(ctx, &stsList, client.InNamespace(namespace), client.MatchingLabels(labels)); err != nil {
		return nil, err
	}
	return &stsList, nil
}

func (c *Client) DeleteStatefulSetIfExists(key types.NamespacedName) error {
	sts, err := c.GetStatefulSet(key)
	if err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
		return nil
	}
	return c.client.Delete(ctx, sts)
}
