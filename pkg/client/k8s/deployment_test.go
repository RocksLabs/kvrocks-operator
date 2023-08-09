package k8s

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestCreateIfNotExistsDeployment(t *testing.T) {
	ns := "unit-test"
	testDeployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: ns,
		},
	}

	tests := []struct {
		name           string
		deployment     *appsv1.Deployment
		existingDeploy *appsv1.Deployment
		expErr         bool
	}{
		{
			name:           "create new deployment",
			deployment:     testDeployment.DeepCopy(),
			existingDeploy: nil,
			expErr:         false,
		}, {
			name:           "create existing deployment",
			deployment:     testDeployment.DeepCopy(),
			existingDeploy: testDeployment.DeepCopy(),
			expErr:         false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert := assert.New(t)

			objs := make([]client.Object, 0)
			if test.existingDeploy != nil {
				objs = append(objs, test.existingDeploy)
			}
			scheme := runtime.NewScheme()
			_ = appsv1.AddToScheme(scheme)
			fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(objs...).Build()
			c := NewK8sClient(fakeClient, ctrl.Log.WithName("deployment-test"))

			err := c.CreateIfNotExistsDeployment(test.deployment)
			if test.expErr {
				assert.Error(err)
			} else {
				assert.NoError(err)
				dep := &appsv1.Deployment{}
				err := fakeClient.Get(context.TODO(), types.NamespacedName{
					Name:      test.deployment.Name,
					Namespace: test.deployment.Namespace,
				}, dep)
				assert.NoError(err)
				assert.Equal(test.deployment.Name, dep.Name)
			}
		})
	}
}

func TestGetDeployment(t *testing.T) {
	ns := "unit-test"
	testDeployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: ns,
		},
	}
	tests := []struct {
		name           string
		deployment     *appsv1.Deployment
		existingDeploy *appsv1.Deployment
		expErr         bool
	}{
		{
			name:           "get existing deployment",
			deployment:     testDeployment.DeepCopy(),
			existingDeploy: testDeployment.DeepCopy(),
			expErr:         false,
		}, {
			name:           "get non-existing deployment",
			deployment:     testDeployment.DeepCopy(),
			existingDeploy: nil,
			expErr:         true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert := assert.New(t)

			objs := make([]client.Object, 0)
			if test.existingDeploy != nil {
				objs = append(objs, test.existingDeploy)
			}
			scheme := runtime.NewScheme()
			_ = appsv1.AddToScheme(scheme)
			fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(objs...).Build()
			c := NewK8sClient(fakeClient, ctrl.Log.WithName("deployment-test"))

			dep, err := c.GetDeployment(types.NamespacedName{
				Name:      test.deployment.Name,
				Namespace: test.deployment.Namespace,
			})
			if test.expErr {
				assert.Error(err)
				assert.True(errors.IsNotFound(err))
			} else {
				assert.NoError(err)
				assert.Equal(test.deployment.Name, dep.Name)
			}
		})
	}
}

func TestUpdateDeployment(t *testing.T) {
	ns := "unit-test"
	testDeployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: ns,
		},
		Status: appsv1.DeploymentStatus{
			Replicas: 1,
		},
	}

	tests := []struct {
		name           string
		deploy         *appsv1.Deployment
		existingDeploy *appsv1.Deployment
		expErr         bool
	}{
		{
			name:           "update existing deployment",
			deploy:         testDeployment.DeepCopy(),
			existingDeploy: testDeployment.DeepCopy(),
			expErr:         false,
		}, {
			name:           "update non-existing deployment",
			deploy:         testDeployment.DeepCopy(),
			existingDeploy: nil,
			expErr:         true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert := assert.New(t)

			objs := make([]client.Object, 0)
			if test.existingDeploy != nil {
				objs = append(objs, test.existingDeploy)
			}
			scheme := runtime.NewScheme()
			_ = appsv1.AddToScheme(scheme)
			fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(objs...).Build()
			c := NewK8sClient(fakeClient, ctrl.Log.WithName("deployment-test"))

			dep, err := c.GetDeployment(types.NamespacedName{
				Name:      test.deploy.Name,
				Namespace: test.deploy.Namespace,
			})
			if dep != nil {
				dep.Status.Replicas = 2
				err = c.UpdateDeployment(dep)
			}
			if test.expErr {
				assert.Error(err)
				assert.True(errors.IsNotFound(err))
			} else {
				assert.NoError(err)

				dep := &appsv1.Deployment{}
				err := fakeClient.Get(context.TODO(), types.NamespacedName{
					Name:      test.deploy.Name,
					Namespace: test.deploy.Namespace,
				}, dep)
				assert.NoError(err)
				assert.Equal(int32(2), dep.Status.Replicas)
			}
		})
	}
}
func TestListDeploymentPods(t *testing.T) {
	ns := "unit-test"
	testDeployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: ns,
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": "test"},
			},
		},
	}

	testPod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pod",
			Namespace: ns,
			Labels:    map[string]string{"app": "test"},
		},
	}

	tests := []struct {
		name    string
		dep     *appsv1.Deployment
		pod     *corev1.Pod
		expPods int
		expErr  bool
	}{
		{
			name:    "list pods for deployment",
			dep:     testDeployment.DeepCopy(),
			pod:     testPod.DeepCopy(),
			expPods: 1,
			expErr:  false,
		}, {
			name:    "no pods",
			dep:     testDeployment.DeepCopy(),
			pod:     nil,
			expPods: 0,
			expErr:  false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert := assert.New(t)

			objs := make([]client.Object, 0)
			objs = append(objs, test.dep)
			if test.pod != nil {
				objs = append(objs, test.pod)
			}
			scheme := runtime.NewScheme()
			_ = appsv1.AddToScheme(scheme)
			_ = corev1.AddToScheme(scheme)
			fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(objs...).Build()
			c := NewK8sClient(fakeClient, ctrl.Log.WithName("deployment-test"))

			pods, err := c.ListDeploymentPods(types.NamespacedName{
				Name:      test.dep.Name,
				Namespace: test.dep.Namespace,
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
