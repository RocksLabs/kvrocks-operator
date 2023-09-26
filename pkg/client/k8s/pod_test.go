package k8s

import (
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	kubeerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	k8sApiClient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestGetPod(t *testing.T) {
	ns := "unit-test"
	testPod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: ns,
		},
	}

	tests := []struct {
		name        string
		pod         *corev1.Pod
		existingPod *corev1.Pod
		expErr      bool
	}{
		{
			name:        "A pod should be returned.",
			pod:         testPod.DeepCopy(),
			existingPod: testPod.DeepCopy(),
			expErr:      false,
		}, {
			name:        "A non existent pod should return an error.",
			pod:         testPod.DeepCopy(),
			existingPod: nil,
			expErr:      true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert := assert.New(t)

			var objs []k8sApiClient.Object
			if test.existingPod != nil {
				objs = append(objs, test.existingPod)
			}
			fakeClient := fake.NewClientBuilder().WithScheme(scheme.Scheme).WithObjects(objs...).Build()
			c := NewK8sClient(fakeClient, ctrl.Log.WithName("pod-test"))

			pod, err := c.GetPod(types.NamespacedName{
				Namespace: test.pod.Namespace,
				Name:      test.pod.Name,
			})
			if test.expErr {
				assert.Error(err)
				assert.True(kubeerrors.IsNotFound(err))
			} else {
				assert.NoError(err)
				assert.Equal(test.pod.Name, pod.Name)
			}
		})
	}
}

func TestUpdatePod(t *testing.T) {
	ns := "unit-test"
	testPod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: ns,
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "container1",
					Image: "image1",
				},
			},
		},
	}

	tests := []struct {
		name        string
		pod         *corev1.Pod
		existingPod *corev1.Pod
		updateImage string
		expErr      bool
	}{
		{
			name:        "A pod should be updated.",
			pod:         testPod.DeepCopy(),
			existingPod: testPod.DeepCopy(),
			updateImage: "image2",
			expErr:      false,
		}, {
			name:        "A non existent pod should return an error.",
			pod:         testPod.DeepCopy(),
			existingPod: nil,
			updateImage: "image2",
			expErr:      true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert := assert.New(t)

			var objs []k8sApiClient.Object
			if test.existingPod != nil {
				objs = append(objs, test.existingPod)
			}
			fakeClient := fake.NewClientBuilder().WithScheme(scheme.Scheme).WithObjects(objs...).Build()
			c := NewK8sClient(fakeClient, ctrl.Log.WithName("pod-test"))

			// update the image of the pod's container
			test.pod.Spec.Containers[0].Image = test.updateImage

			err := c.UpdatePod(test.pod)
			if test.expErr {
				assert.Error(err)
				assert.True(kubeerrors.IsNotFound(err))
			} else {
				assert.NoError(err)

				// Get the updated pod and check if the image of the container has been updated
				updatedPod, err := c.GetPod(types.NamespacedName{
					Namespace: test.pod.Namespace,
					Name:      test.pod.Name,
				})
				assert.NoError(err)
				assert.Equal(test.updateImage, updatedPod.Spec.Containers[0].Image)
			}
		})
	}
}

func TestDeletePodImmediately(t *testing.T) {
	ns := "unit-test"
	testPod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: ns,
		},
	}

	tests := []struct {
		name        string
		pod         *corev1.Pod
		existingPod *corev1.Pod
		expErr      bool
	}{
		{
			name:        "A pod should be deleted.",
			pod:         testPod.DeepCopy(),
			existingPod: testPod.DeepCopy(),
			expErr:      false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert := assert.New(t)

			var objs []k8sApiClient.Object
			if test.existingPod != nil {
				objs = append(objs, test.existingPod)
			}
			fakeClient := fake.NewClientBuilder().WithScheme(scheme.Scheme).WithObjects(objs...).Build()
			c := NewK8sClient(fakeClient, ctrl.Log.WithName("pod-test"))

			err := c.DeletePodImmediately(test.pod.Name, test.pod.Namespace)
			if test.expErr {
				assert.Error(err)
				assert.True(kubeerrors.IsNotFound(err))
			} else {
				assert.NoError(err)

				// Get the deleted pod and check if it has been deleted
				pod, err := c.GetPod(types.NamespacedName{
					Namespace: test.pod.Namespace,
					Name:      test.pod.Name,
				})
				assert.Error(err)
				assert.True(kubeerrors.IsNotFound(err))
				assert.Nil(pod)
			}
		})
	}
}
