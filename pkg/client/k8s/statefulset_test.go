package k8s

import (
	"context"
	"testing"

	kruise "github.com/openkruise/kruise-api/apps/v1beta1"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	k8sApiClient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestCreateIfNotExistsStatefulSet(t *testing.T) {
	ns := "unit-test"
	testSTS := &kruise.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-sts",
			Namespace: ns,
		},
	}

	tests := []struct {
		name        string
		sts         *kruise.StatefulSet
		existingSTS *kruise.StatefulSet
		expErr      bool
	}{
		{
			name:        "create sts successfully",
			sts:         testSTS.DeepCopy(),
			existingSTS: nil,
			expErr:      false,
		}, {
			name:        "existing sts should not cause error",
			sts:         testSTS.DeepCopy(),
			existingSTS: testSTS.DeepCopy(),
			expErr:      false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert := assert.New(t)

			objs := make([]k8sApiClient.Object, 0)
			if test.existingSTS != nil {
				objs = append(objs, test.existingSTS)
			}
			scheme := runtime.NewScheme()
			_ = kruise.AddToScheme(scheme)
			fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(objs...).Build()
			c := NewK8sClient(fakeClient, ctrl.Log.WithName("statefulset-test"))

			err := c.CreateIfNotExistsStatefulSet(test.sts)
			if test.expErr {
				assert.Error(err)
			} else {
				assert.NoError(err)
				sts := &kruise.StatefulSet{}
				err = fakeClient.Get(context.TODO(), types.NamespacedName{
					Namespace: test.sts.Namespace,
					Name:      test.sts.Name,
				}, sts)
				assert.NoError(err)
				assert.Equal(test.sts.Name, sts.Name)
			}
		})
	}
}

func TestGetStatefulSet(t *testing.T) {
	ns := "unit-test"
	testSTS := &kruise.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: ns,
		},
	}

	tests := []struct {
		name        string
		sts         *kruise.StatefulSet
		existingSTS *kruise.StatefulSet
		expErr      bool
	}{
		{
			name:        "get sts successfully",
			sts:         testSTS.DeepCopy(),
			existingSTS: testSTS.DeepCopy(),
			expErr:      false,
		}, {
			name:        "get sts failed",
			sts:         testSTS.DeepCopy(),
			existingSTS: nil,
			expErr:      true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert := assert.New(t)

			objs := make([]k8sApiClient.Object, 0)
			if test.existingSTS != nil {
				objs = append(objs, test.existingSTS)
			}
			scheme := runtime.NewScheme()
			_ = kruise.AddToScheme(scheme)
			fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(objs...).Build()
			c := NewK8sClient(fakeClient, ctrl.Log.WithName("statefulset-test"))

			sts, err := c.GetStatefulSet(types.NamespacedName{
				Namespace: test.sts.Namespace,
				Name:      test.sts.Name,
			})
			if test.expErr {
				assert.Error(err)
				assert.True(errors.IsNotFound(err))
			} else {
				assert.NoError(err)
				assert.Equal(test.existingSTS.Name, sts.Name)
			}
		})
	}
}

func TestUpdateStatefulSet(t *testing.T) {
	ns := "unit-test"
	testSTS := &kruise.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: ns,
		},
		Status: kruise.StatefulSetStatus{
			Replicas: 1,
		},
	}

	tests := []struct {
		name       string
		sts        *kruise.StatefulSet
		exitingSTS *kruise.StatefulSet
		expErr     bool
	}{
		{
			name:       "update sts successfully",
			sts:        testSTS.DeepCopy(),
			exitingSTS: testSTS.DeepCopy(),
			expErr:     false,
		}, {
			name:       "update sts failed",
			sts:        testSTS.DeepCopy(),
			exitingSTS: nil,
			expErr:     true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert := assert.New(t)

			objs := make([]k8sApiClient.Object, 0)
			if test.exitingSTS != nil {
				objs = append(objs, test.exitingSTS)
			}
			scheme := runtime.NewScheme()
			_ = kruise.AddToScheme(scheme)
			fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(objs...).Build()
			c := NewK8sClient(fakeClient, ctrl.Log.WithName("statefulset-test"))

			sts, err := c.GetStatefulSet(types.NamespacedName{
				Namespace: test.sts.Namespace,
				Name:      test.sts.Name,
			})
			if sts != nil {
				sts.Status.Replicas = 2
				err = c.UpdateStatefulSet(sts)
			}
			if test.expErr {
				assert.Error(err)
				assert.True(errors.IsNotFound(err))
			} else {
				assert.NoError(err)

				sts := &kruise.StatefulSet{}
				err = fakeClient.Get(context.TODO(), types.NamespacedName{
					Namespace: test.sts.Namespace,
					Name:      test.sts.Name,
				}, sts)
				assert.NoError(err)
				assert.Equal(int32(2), sts.Status.Replicas)
			}
		})
	}
}

func TestListStatefulSetPods(t *testing.T) {
	ns := "unit-test"
	testSTS := &kruise.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: ns,
		},
		Spec: kruise.StatefulSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": "test",
				},
			},
		},
	}

	testPod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pod",
			Namespace: ns,
			Labels: map[string]string{
				"app": "test",
			},
		},
	}

	tests := []struct {
		name    string
		sts     *kruise.StatefulSet
		pod     *corev1.Pod
		expPods int
		expErr  bool
	}{
		{
			name:    "list sts pods successfully",
			sts:     testSTS.DeepCopy(),
			pod:     testPod.DeepCopy(),
			expPods: 1,
			expErr:  false,
		}, {
			name:    "no pods",
			sts:     testSTS.DeepCopy(),
			pod:     nil,
			expPods: 0,
			expErr:  false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert := assert.New(t)

			objs := make([]k8sApiClient.Object, 0)
			objs = append(objs, test.sts)
			if test.pod != nil {
				objs = append(objs, test.pod)
			}

			scheme := runtime.NewScheme()
			_ = kruise.AddToScheme(scheme)
			_ = corev1.AddToScheme(scheme)
			fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(objs...).Build()
			c := NewK8sClient(fakeClient, ctrl.Log.WithName("statefulset-test"))

			pods, err := c.ListStatefulSetPods(types.NamespacedName{
				Namespace: test.sts.Namespace,
				Name:      test.sts.Name,
			})
			if test.expErr {
				assert.Error(err)
				assert.True(errors.IsNotFound(err))
			} else {
				assert.NoError(err)
				assert.Equal(test.expPods, len(pods.Items))
			}
		})
	}
}

func TestCreateOrUpdateStatefulSet(t *testing.T) {
	ns := "unit-test"
	testSTS := &kruise.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: ns,
		},
		Status: kruise.StatefulSetStatus{
			Replicas: 1,
		},
	}

	tests := []struct {
		name       string
		sts        *kruise.StatefulSet
		exitingSTS *kruise.StatefulSet
		expErr     bool
	}{
		{
			name:       "create sts successfully",
			sts:        testSTS.DeepCopy(),
			exitingSTS: nil,
			expErr:     false,
		}, {
			name:       "update sts successfully",
			sts:        testSTS.DeepCopy(),
			exitingSTS: testSTS.DeepCopy(),
			expErr:     false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert := assert.New(t)

			objs := make([]k8sApiClient.Object, 0)
			if test.exitingSTS != nil {
				objs = append(objs, test.exitingSTS)
			}
			scheme := runtime.NewScheme()
			_ = kruise.AddToScheme(scheme)
			fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(objs...).Build()
			c := NewK8sClient(fakeClient, ctrl.Log.WithName("statefulset-test"))

			err := c.CreateOrUpdateStatefulSet(test.sts)
			if test.expErr {
				assert.Error(err)
			} else {
				assert.NoError(err)

				sts := &kruise.StatefulSet{}
				err = fakeClient.Get(context.TODO(), types.NamespacedName{
					Namespace: test.sts.Namespace,
					Name:      test.sts.Name,
				}, sts)
				assert.NoError(err)
				assert.Equal(int32(1), sts.Status.Replicas)
			}
		})
	}
}

func TestListStatefulSets(t *testing.T) {
	ns := "unit-test"
	labels := map[string]string{
		"app": "test",
	}
	testSTS := &kruise.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: ns,
			Labels:    labels,
		},
	}

	tests := []struct {
		name        string
		namespace   string
		labels      map[string]string
		existingSTS *kruise.StatefulSet
		expItems    int
		expErr      bool
	}{
		{
			name:        "list sts successfully",
			namespace:   ns,
			labels:      labels,
			existingSTS: testSTS.DeepCopy(),
			expItems:    1,
			expErr:      false,
		}, {
			name:        "no sts found",
			namespace:   ns,
			labels:      map[string]string{"app": "not-found"},
			existingSTS: nil,
			expItems:    0,
			expErr:      false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert := assert.New(t)

			objs := make([]k8sApiClient.Object, 0)
			if test.existingSTS != nil {
				objs = append(objs, test.existingSTS)
			}
			scheme := runtime.NewScheme()
			_ = kruise.AddToScheme(scheme)
			fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(objs...).Build()
			c := NewK8sClient(fakeClient, ctrl.Log.WithName("statefulset-test"))

			stsList, err := c.ListStatefulSets(test.namespace, test.labels)
			if test.expErr {
				assert.Error(err)
			} else {
				assert.NoError(err)
				assert.Equal(test.expItems, len(stsList.Items))
			}
		})
	}
}

func TestDeleteStatefulSetIfExists(t *testing.T) {
	ns := "unit-test"
	testSTS := &kruise.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: ns,
		},
	}

	tests := []struct {
		name        string
		sts         *kruise.StatefulSet
		existingSTS *kruise.StatefulSet
		expErr      bool
	}{
		{
			name:        "delete sts successfully",
			sts:         testSTS.DeepCopy(),
			existingSTS: testSTS.DeepCopy(),
			expErr:      false,
		}, {
			name:        "no sts found",
			sts:         testSTS.DeepCopy(),
			existingSTS: nil,
			expErr:      false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert := assert.New(t)

			objs := make([]k8sApiClient.Object, 0)
			if test.existingSTS != nil {
				objs = append(objs, test.existingSTS)
			}
			scheme := runtime.NewScheme()
			_ = kruise.AddToScheme(scheme)
			fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(objs...).Build()
			c := NewK8sClient(fakeClient, ctrl.Log.WithName("statefulset-test"))

			err := c.DeleteStatefulSetIfExists(types.NamespacedName{
				Namespace: test.sts.Namespace,
				Name:      test.sts.Name,
			})
			if test.expErr {
				assert.Error(err)
			} else {
				assert.NoError(err)

				sts := &kruise.StatefulSet{}
				err = fakeClient.Get(context.TODO(), types.NamespacedName{
					Namespace: test.sts.Namespace,
					Name:      test.sts.Name,
				}, sts)
				assert.Error(err)
				assert.True(errors.IsNotFound(err))
			}
		})
	}
}
