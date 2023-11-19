package k8s

import (
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

func TestListStatefulSetPVC(t *testing.T) {
	ns := "unit-test"
	testStatefulSet := &kruise.StatefulSet{
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
	testPVC := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pvc",
			Namespace: ns,
			Labels: map[string]string{
				"app": "test",
			},
		},
	}

	tests := []struct {
		name        string
		sts         *kruise.StatefulSet
		pvc         *corev1.PersistentVolumeClaim
		existingSTS *kruise.StatefulSet
		existingPVC *corev1.PersistentVolumeClaim
		expErr      bool
	}{
		{
			name:        "PVCs should be returned.",
			sts:         testStatefulSet.DeepCopy(),
			pvc:         testPVC.DeepCopy(),
			existingSTS: testStatefulSet.DeepCopy(),
			existingPVC: testPVC.DeepCopy(),
			expErr:      false,
		}, {
			name:        "A non existent PVC should return an error.",
			sts:         testStatefulSet.DeepCopy(),
			pvc:         testPVC.DeepCopy(),
			existingSTS: nil,
			existingPVC: testPVC.DeepCopy(),
			expErr:      true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert := assert.New(t)

			var objs []k8sApiClient.Object
			if test.existingSTS != nil {
				objs = append(objs, test.existingSTS)
			}
			if test.existingPVC != nil {
				objs = append(objs, test.existingPVC)
			}
			scheme := runtime.NewScheme()
			_ = corev1.AddToScheme(scheme)
			_ = kruise.AddToScheme(scheme)
			fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(objs...).Build()
			c := NewK8sClient(fakeClient, ctrl.Log.WithName("pvc-test"))

			pvcList, err := c.ListStatefulSetPVC(types.NamespacedName{
				Namespace: test.sts.Namespace,
				Name:      test.sts.Name,
			})
			if test.expErr {
				assert.Error(err)
				assert.Nil(pvcList)
			} else {
				assert.NoError(err)
				assert.Equal(1, len(pvcList.Items))
				assert.Equal(test.pvc.Name, pvcList.Items[0].Name)
			}
		})
	}
}

func TestDeletePVC(t *testing.T) {
	ns := "unit-test"
	testPVC := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pvc",
			Namespace: ns,
		},
	}

	tests := []struct {
		name        string
		pvc         *corev1.PersistentVolumeClaim
		existingPVC *corev1.PersistentVolumeClaim
		expErr      bool
	}{
		{
			name:        "PVC should be deleted.",
			pvc:         testPVC.DeepCopy(),
			existingPVC: testPVC.DeepCopy(),
			expErr:      false,
		}, {
			name:        "A non existent PVC should return an error.",
			pvc:         testPVC.DeepCopy(),
			existingPVC: nil,
			expErr:      false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert := assert.New(t)

			var objs []k8sApiClient.Object
			if test.existingPVC != nil {
				objs = append(objs, test.existingPVC)
			}
			scheme := runtime.NewScheme()
			_ = corev1.AddToScheme(scheme)
			fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(objs...).Build()
			c := NewK8sClient(fakeClient, ctrl.Log.WithName("pvc-test"))

			err := c.DeletePVC(test.pvc)
			if test.expErr {
				t.Log(err)
				assert.Error(err)
			} else {
				assert.NoError(err)

				pvc := &corev1.PersistentVolumeClaim{}
				err = fakeClient.Get(ctx, types.NamespacedName{
					Namespace: test.pvc.Namespace,
					Name:      test.pvc.Name,
				}, pvc)
				assert.True(errors.IsNotFound(err))
			}
		})
	}
}

func TestListPVC(t *testing.T) {
	ns := "unit-test"
	testPVC1 := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pvc1",
			Namespace: ns,
			Labels: map[string]string{
				"app": "test",
			},
		},
	}

	testPVC2 := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pvc2",
			Namespace: ns,
			Labels: map[string]string{
				"app": "test",
			},
		},
	}

	tests := []struct {
		name          string
		ns            string
		labelSelector map[string]string
		existingPVCs  []*corev1.PersistentVolumeClaim
		expPVCNames   []string
		expErr        bool
	}{
		{
			name: "PVCs with matching labels should be returned.",
			ns:   ns,
			labelSelector: map[string]string{
				"app": "test",
			},
			existingPVCs: []*corev1.PersistentVolumeClaim{
				testPVC1.DeepCopy(),
				testPVC2.DeepCopy(),
			},
			expPVCNames: []string{"test-pvc1", "test-pvc2"},
			expErr:      false,
		}, {
			name: "PVCs with non matching labels should not be returned.",
			ns:   ns,
			labelSelector: map[string]string{
				"app": "no-match",
			},
			existingPVCs: []*corev1.PersistentVolumeClaim{
				testPVC1.DeepCopy(),
				testPVC2.DeepCopy(),
			},
			expPVCNames: []string{},
			expErr:      false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert := assert.New(t)

			objs := make([]k8sApiClient.Object, len(test.existingPVCs))
			for i, pvc := range test.existingPVCs {
				objs[i] = pvc
			}
			scheme := runtime.NewScheme()
			_ = corev1.AddToScheme(scheme)
			fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(objs...).Build()
			c := NewK8sClient(fakeClient, ctrl.Log.WithName("pvc-test"))

			pvcList, err := c.ListPVC(test.ns, test.labelSelector)
			if test.expErr {
				assert.Error(err)
			} else {
				assert.NoError(err)
				assert.Equal(len(test.expPVCNames), len(pvcList.Items))
				for _, pvc := range pvcList.Items {
					assert.Contains(test.expPVCNames, pvc.Name)
				}
			}
		})
	}
}

func TestDeletePVCByPod(t *testing.T) {
	ns := "unit-test"
	podName := "test-pod"
	testPVC := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "data-" + podName,
			Namespace: ns,
		},
	}

	tests := []struct {
		name        string
		podName     string
		ns          string
		existingPVC *corev1.PersistentVolumeClaim
		expErr      bool
	}{
		{
			name:        "PVC should be deleted.",
			podName:     podName,
			ns:          ns,
			existingPVC: testPVC.DeepCopy(),
			expErr:      false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert := assert.New(t)

			objs := []k8sApiClient.Object{}
			if test.existingPVC != nil {
				objs = append(objs, test.existingPVC)
			}
			scheme := runtime.NewScheme()
			_ = corev1.AddToScheme(scheme)
			fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(objs...).Build()
			c := NewK8sClient(fakeClient, ctrl.Log.WithName("pvc-test"))

			err := c.DeletePVCByPod(test.ns, test.podName)
			if test.expErr {
				assert.Error(err)
			} else {
				assert.NoError(err)
				pvc := &corev1.PersistentVolumeClaim{}
				err = fakeClient.Get(ctx, types.NamespacedName{
					Namespace: test.ns,
					Name:      "data-" + test.podName,
				}, pvc)
				assert.Nil(err)
			}
		})
	}
}
