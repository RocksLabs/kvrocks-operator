/*
Copyright 2022.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"
	"fmt"
	"reflect"
	"sync"
	"time"

	"github.com/go-logr/logr"
	kruise "github.com/openkruise/kruise-api/apps/v1beta1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	kvrocksv1alpha1 "github.com/RocksLabs/kvrocks-operator/api/v1alpha1"
	k8s "github.com/RocksLabs/kvrocks-operator/pkg/client/k8s"
	kv "github.com/RocksLabs/kvrocks-operator/pkg/client/kvrocks"
	"github.com/RocksLabs/kvrocks-operator/pkg/controllers/cluster"
	"github.com/RocksLabs/kvrocks-operator/pkg/controllers/events"
	"github.com/RocksLabs/kvrocks-operator/pkg/controllers/sentinel"
	"github.com/RocksLabs/kvrocks-operator/pkg/controllers/standard"
	"github.com/RocksLabs/kvrocks-operator/pkg/resources"
)

type KVRocksHandler interface {
	Handle() (error, bool)
	Finializer() error
	Requeue() bool
}

// KVRocksReconciler reconciles a KVRocks object
type KVRocksReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
	once   sync.Once
}

//+kubebuilder:rbac:groups=kvrocks.apache.org,resources=kvrocks,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=kvrocks.apache.org,resources=kvrocks/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=kvrocks.apache.org,resources=kvrocks/finalizers,verbs=update
//+kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=nodes,verbs=get;list;watch
//+kubebuilder:rbac:groups=core,resources=configmaps,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=secret,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=services,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=events,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=persistentvolumeclaims,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=apps.kruise.io,resources=statefulsets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the KVRocks object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.11.0/pkg/reconcile
func (r *KVRocksReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithName(req.NamespacedName.String())
	k8sClient := k8s.NewK8sClient(r.Client, log)
	kvClient := kv.NewKVRocksClient(log)
	instance, err := k8sClient.GetKVRocks(req.NamespacedName)
	if err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}
	r.once.Do(func() {
		event := events.NewEvent(k8sClient, kvClient, log)
		go event.Run()
	})
	var handler KVRocksHandler
	switch instance.Spec.Type {
	case kvrocksv1alpha1.SentinelType:
		handler = sentinel.NewKVRocksSentinelHandler(k8sClient, kvClient, log, req.NamespacedName, instance)
	case kvrocksv1alpha1.StandardType:
		handler = standard.NewKVRocksStandardHandler(k8sClient, kvClient, log, req.NamespacedName, instance)
	case kvrocksv1alpha1.ClusterType:
		handler = cluster.NewKVRocksClusterHandler(k8sClient, kvClient, log, req.NamespacedName, instance)
	}
	// delete
	if instance.GetDeletionTimestamp() != nil {
		log.Info("begin delete kvrocks")
		if !controllerutil.ContainsFinalizer(instance, kvrocksv1alpha1.KVRocksFinalizer) {
			return ctrl.Result{}, nil
		}
		err = handler.Finializer()
		if handler.Requeue() || shouldRetry(err) {
			return ctrl.Result{RequeueAfter: time.Second * 10}, nil
		}
		if err == nil {
			controllerutil.RemoveFinalizer(instance, kvrocksv1alpha1.KVRocksFinalizer)
			err = k8sClient.UpdateKVRocks(instance)
			if shouldRetry(err) {
				return ctrl.Result{RequeueAfter: time.Second * 10}, nil
			}
		}
		log.Info("delete kvrocks successfully")
		return ctrl.Result{}, nil
	}
	// if kvrocks status failed ,do nothing
	if instance.Status.Status == kvrocksv1alpha1.StatusFailed {
		return ctrl.Result{}, nil
	}
	// check kvrocks spec is reasonable
	if err = checkSpecification(instance, log, k8sClient); err != nil {
		return ctrl.Result{}, nil
	}
	// add labels and init status
	err = runIfInitialize(instance, log, k8sClient)
	if err != nil {
		return ctrl.Result{}, nil
	}
	log.Info("reconcile begin")
	err, _ = handler.Handle()
	if handler.Requeue() || shouldRetry(err) {
		return ctrl.Result{RequeueAfter: time.Second * 10}, nil
	}
	if shouldNotRetry(err) {
		return ctrl.Result{}, nil
	}
	if err == nil {
		log.Info("reconcile end")
	}
	return ctrl.Result{}, err
}

// SetupWithManager sets up the controller with the Manager.
func (r *KVRocksReconciler) SetupWithManager(mgr ctrl.Manager, maxConcurrentReconciles int) error {
	mgr.GetFieldIndexer().IndexField(context.Background(), &corev1.Pod{}, "spec.nodeName", func(o client.Object) []string {
		return []string{o.(*corev1.Pod).Spec.NodeName}
	})
	return ctrl.NewControllerManagedBy(mgr).
		WithOptions(controller.Options{MaxConcurrentReconciles: maxConcurrentReconciles}).
		For(&kvrocksv1alpha1.KVRocks{}).
		Owns(&kruise.StatefulSet{}).
		Owns(&corev1.Pod{}).
		Owns(&appsv1.Deployment{}).
		Complete(r)
}

// checkSpecification Check if the field is correct
// 1. Password must be set
// 2. Replicas must be greater than 0, must be greater than 2 in sentinel mode, and must be odd
func checkSpecification(instance *kvrocksv1alpha1.KVRocks, log logr.Logger, k8sClient *k8s.Client) error {
	ok, reason := resources.ValidateKVRocks(instance, log)
	if !ok {
		instance.Status.Status = kvrocksv1alpha1.StatusFailed
		instance.Status.Reason = *reason
		_ = k8sClient.UpdateKVRocks(instance)
		return fmt.Errorf("field is unreasonable error: %s", *reason)
	}
	return nil
}

// if err is NotFound error or update conflicts error, requeue
func shouldRetry(err error) bool {
	return err != nil && (errors.IsNotFound(err) || errors.IsConflict(err))
}

// if err is forbidden error or invalid error, ignore
func shouldNotRetry(err error) bool {
	return err != nil && (errors.IsForbidden(err) || errors.IsInvalid(err))
}

func runIfInitialize(instance *kvrocksv1alpha1.KVRocks, log logr.Logger, k8sClient *k8s.Client) error {
	labels := resources.MergeLabels(instance.Labels, resources.SelectorLabels(instance))
	if instance.Spec.Type == kvrocksv1alpha1.ClusterType {
		sysId, _ := resources.ParseRedisName(instance.Name)
		labels = resources.MergeLabels(labels, resources.MonitorLabels(resources.GetSentinelName(sysId)))
	}
	if instance.Spec.Type == kvrocksv1alpha1.SentinelType {
		labels = resources.MergeLabels(labels, resources.SentinelLabels())
	}
	if !reflect.DeepEqual(labels, instance.Labels) {
		instance.Labels = labels
		if err := k8sClient.UpdateKVRocks(instance); err != nil {
			return err
		}
	}
	// TODO wait for the relase of cluster mode
	// if instance.Spec.Type == kvrocksv1alpha1.ClusterType && !instance.Spec.SentinelConfig.EnableSentinel {
	// 	instance.Spec.SentinelConfig.EnableSentinel = true
	// 	return k8sClient.UpdateKVRocks(instance)
	// }
	if instance.Status.Status == kvrocksv1alpha1.StatusNone {
		log.Info("kvrocks is creating")
		instance.Status.Status = kvrocksv1alpha1.StatusCreating
		return k8sClient.UpdateKVRocks(instance)
	}
	return nil
}
