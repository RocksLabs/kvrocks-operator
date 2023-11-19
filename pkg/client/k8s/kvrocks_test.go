package k8s

import (
	"context"
	"testing"

	kvrocksv1alpha1 "github.com/RocksLabs/kvrocks-operator/api/v1alpha1"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	k8sApiClient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestGetKVRocks(t *testing.T) {
	ns := "unit-test"
	testKVRocks := &kvrocksv1alpha1.KVRocks{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: ns,
		},
	}

	tests := []struct {
		name            string
		kvrocks         *kvrocksv1alpha1.KVRocks
		existingKVRocks *kvrocksv1alpha1.KVRocks
		expErr          bool
	}{
		{
			name:            "KVRocks should be returned.",
			kvrocks:         testKVRocks.DeepCopy(),
			existingKVRocks: testKVRocks.DeepCopy(),
			expErr:          false,
		}, {
			name:            "A non existent KVRocks should return an error.",
			kvrocks:         testKVRocks.DeepCopy(),
			existingKVRocks: nil,
			expErr:          true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert := assert.New(t)

			var objs []k8sApiClient.Object
			if test.existingKVRocks != nil {
				objs = append(objs, test.existingKVRocks)
			}
			scheme := runtime.NewScheme()
			_ = kvrocksv1alpha1.AddToScheme(scheme)
			fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(objs...).Build()
			c := NewK8sClient(fakeClient, ctrl.Log.WithName("kvrocks-test"))

			_, err := c.GetKVRocks(types.NamespacedName{
				Namespace: test.kvrocks.Namespace,
				Name:      test.kvrocks.Name,
			})
			if test.expErr {
				assert.Error(err)
				assert.True(errors.IsNotFound(err))
			} else {
				assert.NoError(err)
			}
		})
	}
}

func TestUpdateKVRocks(t *testing.T) {
	ns := "unit-test"
	testKVRocks := &kvrocksv1alpha1.KVRocks{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: ns,
		},
		Status: kvrocksv1alpha1.KVRocksStatus{
			Status: kvrocksv1alpha1.StatusNone,
		},
	}

	tests := []struct {
		name            string
		kvrocks         *kvrocksv1alpha1.KVRocks
		existingKVRocks *kvrocksv1alpha1.KVRocks
		expErr          bool
	}{
		{
			name:            "KVRocks should be updated.",
			kvrocks:         testKVRocks.DeepCopy(),
			existingKVRocks: testKVRocks.DeepCopy(),
			expErr:          false,
		}, {
			name:            "A non existent KVRocks should return an error.",
			kvrocks:         testKVRocks.DeepCopy(),
			existingKVRocks: nil,
			expErr:          true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert := assert.New(t)

			objs := make([]k8sApiClient.Object, 0)
			if test.existingKVRocks != nil {
				objs = append(objs, test.existingKVRocks)
			}
			scheme := runtime.NewScheme()
			_ = kvrocksv1alpha1.AddToScheme(scheme)
			fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(objs...).Build()
			c := NewK8sClient(fakeClient, ctrl.Log.WithName("kvrocks-test"))

			kvrocks, err := c.GetKVRocks(types.NamespacedName{
				Namespace: test.kvrocks.Namespace,
				Name:      test.kvrocks.Name,
			})

			if kvrocks != nil {
				kvrocks.Status.Status = kvrocksv1alpha1.StatusCreating
				err = c.UpdateKVRocks(kvrocks)
			}
			if test.expErr {
				assert.Error(err)
				assert.True(errors.IsNotFound(err))
			} else {
				assert.NoError(err)
				updatedKVRocks, err := c.GetKVRocks(types.NamespacedName{
					Namespace: test.kvrocks.Namespace,
					Name:      test.kvrocks.Name,
				})
				assert.NoError(err)
				assert.Equal(kvrocksv1alpha1.StatusCreating, updatedKVRocks.Status.Status)
			}
		})
	}
}

func TestListKVRocks(t *testing.T) {
	ns := "unit-test"
	labels := map[string]string{
		"app": "kvrocks",
	}

	testKVRocks1 := &kvrocksv1alpha1.KVRocks{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test1",
			Namespace: ns,
			Labels:    labels,
		},
	}

	testKVRocks2 := &kvrocksv1alpha1.KVRocks{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test2",
			Namespace: ns,
			Labels:    labels,
		},
	}

	tests := []struct {
		name      string
		kvrocks   []*kvrocksv1alpha1.KVRocks
		namespace string
		labels    map[string]string
		expErr    bool
		expNames  []string
	}{
		{
			name: "List should return all matching kvrocks.",
			kvrocks: []*kvrocksv1alpha1.KVRocks{
				testKVRocks1.DeepCopy(),
				testKVRocks2.DeepCopy(),
			},
			namespace: ns,
			labels:    labels,
			expErr:    false,
			expNames: []string{
				testKVRocks1.Name,
				testKVRocks2.Name,
			},
		}, {
			name:      "No kvrocks match.",
			kvrocks:   []*kvrocksv1alpha1.KVRocks{},
			namespace: ns,
			labels:    labels,
			expErr:    false,
			expNames:  []string{},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert := assert.New(t)

			objs := make([]k8sApiClient.Object, len(test.kvrocks))
			for i, kvrocks := range test.kvrocks {
				objs[i] = kvrocks
			}
			scheme := runtime.NewScheme()
			_ = kvrocksv1alpha1.AddToScheme(scheme)
			fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(objs...).Build()
			c := NewK8sClient(fakeClient, ctrl.Log.WithName("kvrocks-test"))

			list, err := c.ListKVRocks(test.namespace, test.labels)
			if test.expErr {
				assert.Error(err)
			} else {
				assert.NoError(err)
				assert.Equal(len(test.expNames), len(list.Items))
				for i, kvrocks := range list.Items {
					assert.Equal(test.expNames[i], kvrocks.Name)
				}
			}
		})
	}
}

func TestCreateIfNotExistsKVRocks(t *testing.T) {
	ns := "unit-test"
	testKVRocks := &kvrocksv1alpha1.KVRocks{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: ns,
		},
	}

	tests := []struct {
		name            string
		kvrocks         *kvrocksv1alpha1.KVRocks
		existingKVRocks *kvrocksv1alpha1.KVRocks
		expErr          bool
	}{
		{
			name:            "KVRocks should be created.",
			kvrocks:         testKVRocks.DeepCopy(),
			existingKVRocks: nil,
			expErr:          false,
		}, {
			name:            "Existing KVRocks should not cause an error.",
			kvrocks:         testKVRocks.DeepCopy(),
			existingKVRocks: testKVRocks.DeepCopy(),
			expErr:          false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert := assert.New(t)

			objs := make([]k8sApiClient.Object, 0)
			if test.existingKVRocks != nil {
				objs = append(objs, test.existingKVRocks)
			}
			scheme := runtime.NewScheme()
			_ = kvrocksv1alpha1.AddToScheme(scheme)
			fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(objs...).Build()
			c := NewK8sClient(fakeClient, ctrl.Log.WithName("kvrocks-test"))

			err := c.CreateIfNotExistsKVRocks(test.kvrocks)
			if test.expErr {
				assert.Error(err)
			} else {
				assert.NoError(err)
				createdKVRocks := &kvrocksv1alpha1.KVRocks{}
				err := fakeClient.Get(context.Background(), k8sApiClient.ObjectKey{
					Namespace: test.kvrocks.Namespace,
					Name:      test.kvrocks.Name,
				}, createdKVRocks)
				assert.NoError(err)
				assert.Equal(test.kvrocks.Name, createdKVRocks.Name)
			}
		})
	}
}
