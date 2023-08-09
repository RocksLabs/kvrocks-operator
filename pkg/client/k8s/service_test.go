package k8s

import (
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"testing"
)

func TestCreateIfNotExistsService(t *testing.T) {
	ns := "unit-test"
	serviceName := "test-service"
	testService := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      serviceName,
			Namespace: ns,
		},
	}

	tests := []struct {
		name            string
		service         *corev1.Service
		existingService *corev1.Service
		expErr          bool
	}{
		{
			name:            "Service does not exist, should be created successfully.",
			service:         testService.DeepCopy(),
			existingService: nil,
			expErr:          false,
		}, {
			name:            "Service already exists, should not be created.",
			service:         testService.DeepCopy(),
			existingService: testService.DeepCopy(),
			expErr:          false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert := assert.New(t)

			objs := []client.Object{}
			if test.existingService != nil {
				objs = append(objs, test.existingService)
			}
			scheme := runtime.NewScheme()
			_ = corev1.AddToScheme(scheme)
			fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(objs...).Build()
			c := NewK8sClient(fakeClient, ctrl.Log.WithName("service-test"))

			err := c.CreateIfNotExistsService(test.service)
			if test.expErr {
				assert.Error(err)
			} else {
				assert.NoError(err)
				service := &corev1.Service{}
				err = fakeClient.Get(ctx, types.NamespacedName{
					Namespace: test.service.Namespace,
					Name:      test.service.Name,
				}, service)
				assert.Nil(err)
			}
		})
	}
}
