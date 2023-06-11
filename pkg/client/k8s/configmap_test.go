package k8s

import (
	"context"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	kubeerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"testing"
)

func TestGetConfigMap(t *testing.T) {
	ns := "unit-test"
	testConfigMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: ns,
		},
	}

	tests := []struct {
		name       string
		configMap  *corev1.ConfigMap
		existingCM *corev1.ConfigMap
		expErr     bool
	}{
		{
			name:       "A configmap should be returned.",
			configMap:  testConfigMap.DeepCopy(),
			existingCM: testConfigMap.DeepCopy(),
			expErr:     false,
		}, {
			name:       "A non existent configmap should return an error.",
			configMap:  testConfigMap.DeepCopy(),
			existingCM: nil,
			expErr:     true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert := assert.New(t)

			var objs []client.Object
			if test.existingCM != nil {
				objs = append(objs, test.existingCM)
			}
			fakeClient := fake.NewClientBuilder().WithScheme(scheme.Scheme).WithObjects(objs...).Build()

			c := NewK8sClient(fakeClient, ctrl.Log.WithName("configmap-test"))

			cm, err := c.GetConfigMap(types.NamespacedName{
				Namespace: test.configMap.Namespace,
				Name:      test.configMap.Name,
			})

			if test.expErr {
				assert.Error(err)
				assert.True(kubeerrors.IsNotFound(err))
			} else {
				assert.NoError(err)
				assert.Equal(test.existingCM.Name, cm.Name)
				assert.Equal(test.existingCM.Namespace, cm.Namespace)
			}
		})
	}
}
func TestUpdateConfigMap(t *testing.T) {
	ns, updatedKey, updatedValue := "unit-test", "key1", "value2"
	testConfigMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: ns,
		},
		Data: map[string]string{
			updatedKey: "value1",
		},
	}

	updatedConfigMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: ns,
		},
		Data: map[string]string{
			updatedKey: updatedValue,
		},
	}

	tests := []struct {
		name       string
		configMap  *corev1.ConfigMap
		existingCM *corev1.ConfigMap
		expErr     bool
	}{
		{
			name:       "A configmap should be updated.",
			configMap:  updatedConfigMap.DeepCopy(),
			existingCM: testConfigMap.DeepCopy(),
			expErr:     false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert := assert.New(t)

			var objs []client.Object
			if test.existingCM != nil {
				objs = append(objs, test.existingCM)
			}
			fakeClient := fake.NewClientBuilder().WithScheme(scheme.Scheme).WithObjects(objs...).Build()

			c := NewK8sClient(fakeClient, ctrl.Log.WithName("configmap-test"))

			err := c.UpdateConfigMap(test.configMap)
			assert.NoError(err)

			cm, err := c.GetConfigMap(types.NamespacedName{
				Namespace: test.configMap.Namespace,
				Name:      test.configMap.Name,
			})
			if test.expErr {
				assert.Error(err)
			} else {
				assert.NoError(err)
				assert.Equal(test.configMap.Data[updatedKey], cm.Data[updatedKey])
			}
		})
	}

}
func TestCreateOrUpdateConfigMap(t *testing.T) {
	ns := "unit-test"

	testConfigMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: ns,
		},
	}

	tests := []struct {
		name       string
		configMap  *corev1.ConfigMap
		existingCM *corev1.ConfigMap
		expErr     bool
	}{
		{
			name:       "A new configmap should create a new configmap.",
			configMap:  testConfigMap.DeepCopy(),
			existingCM: nil,
			expErr:     false,
		}, {
			name:       "An existent configmap should update the configmap.",
			configMap:  testConfigMap.DeepCopy(),
			existingCM: testConfigMap.DeepCopy(),
			expErr:     false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert := assert.New(t)

			var objs []client.Object
			if test.existingCM != nil {
				objs = append(objs, test.existingCM)
			}
			fakeClient := fake.NewClientBuilder().WithScheme(scheme.Scheme).WithObjects(objs...).Build()

			c := NewK8sClient(fakeClient, ctrl.Log.WithName("configmap-test"))

			err := c.CreateOrUpdateConfigMap(test.configMap)

			if test.expErr {
				assert.Error(err)
			} else {
				assert.NoError(err)

				cm := &corev1.ConfigMap{}
				err = fakeClient.Get(context.TODO(), client.ObjectKey{Namespace: ns, Name: test.configMap.Name}, cm)
				assert.NoError(err)
				assert.Equal(test.configMap.ResourceVersion, cm.ResourceVersion)
			}
		})
	}
}
func TestCreateIfNotExistsConfigMap(t *testing.T) {
	ns := "unit-test"
	testConfigMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: ns,
		},
	}

	tests := []struct {
		name       string
		configMap  *corev1.ConfigMap
		existingCM *corev1.ConfigMap
		expErr     bool
	}{
		{
			name:       "Creating a new configmap should succeed.",
			configMap:  testConfigMap.DeepCopy(),
			existingCM: nil,
			expErr:     false,
		}, {
			name:       "Creating an existing configmap should not return an error.",
			configMap:  testConfigMap.DeepCopy(),
			existingCM: testConfigMap.DeepCopy(),
			expErr:     false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert := assert.New(t)

			var objs []client.Object
			if test.existingCM != nil {
				objs = append(objs, test.existingCM)
			}
			fakeClient := fake.NewClientBuilder().WithScheme(scheme.Scheme).WithObjects(objs...).Build()
			c := NewK8sClient(fakeClient, ctrl.Log.WithName("configmap-test"))

			err := c.CreateIfNotExistsConfigMap(test.configMap)

			if test.expErr {
				assert.Error(err)
			} else {
				assert.NoError(err)

				cm := &corev1.ConfigMap{}
				err = fakeClient.Get(context.TODO(), client.ObjectKey{Namespace: ns, Name: test.configMap.Name}, cm)
				assert.NoError(err)
				assert.Equal(test.configMap.Name, cm.Name)
			}
		})
	}
}
