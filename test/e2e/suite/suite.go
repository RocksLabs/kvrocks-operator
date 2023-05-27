package suite

import (
	"context"
	"flag"
	"path/filepath"

	"github.com/RocksLabs/kvrocks-operator/pkg/controllers"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	kruise "github.com/openkruise/kruise-api/apps/v1beta1"
	"go.uber.org/zap/zapcore"
	"k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	kvrocksv1alpha1 "github.com/RocksLabs/kvrocks-operator/api/v1alpha1"
	"github.com/RocksLabs/kvrocks-operator/pkg/client/kvrocks"
)

var K8sClient client.Client
var KVRocksClient *kvrocks.Client
var CTX = context.Background()
var useExistingCluster = true
var testEnv *envtest.Environment

var _ = BeforeSuite(func() {
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))

	By("bootstrapping test environment")
	testEnv = &envtest.Environment{
		CRDDirectoryPaths:     []string{filepath.Join("..", "..", "config", "crd", "bases")},
		ErrorIfCRDPathMissing: true,
		UseExistingCluster:    &useExistingCluster,
	}

	cfg, err := testEnv.Start()
	Expect(err).NotTo(HaveOccurred())
	Expect(cfg).NotTo(BeNil())

	err = kvrocksv1alpha1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())
	err = kruise.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	//+kubebuilder:scaffold:scheme
	K8sClient, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})
	Expect(err).NotTo(HaveOccurred())
	Expect(K8sClient).NotTo(BeNil())
	opts := zap.Options{
		Development: true,
		Level:       zapcore.ErrorLevel,
	}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()
	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))
	// Start the reconciler
	k8sManager, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme: scheme.Scheme,
	})
	Expect(err).ToNot(HaveOccurred())

	KVRocksClient = kvrocks.NewKVRocksClient(ctrl.Log)

	err = (&controllers.KVRocksReconciler{
		Client: k8sManager.GetClient(),
		Scheme: k8sManager.GetScheme(),
		Log:    ctrl.Log,
	}).SetupWithManager(k8sManager, 4)
	Expect(err).ToNot(HaveOccurred())

	go func() {
		err = k8sManager.Start(ctrl.SetupSignalHandler())
		Expect(err).ToNot(HaveOccurred())
	}()
}, 60)

var _ = AfterSuite(func() {
	By("tearing down the test environment")
	err := testEnv.Stop()
	Expect(err).NotTo(HaveOccurred())
})

const (
	KVRocksStandard = `
apiVersion: kvrocks.com/v1alpha1
kind: KVRocks
metadata:
  name: kvrocks-standard-1-test
  namespace: kvrocks
  labels:
    redis/system: xxxxx
spec:
  image: xxxx
  imagePullPolicy: IfNotPresent
  master: 1
  replicas: 2
  type: standard
  enableSentinel: true
  password: "123456"
  kvrocksConfig:
    bind: "0.0.0.0"
    port: "6379"
    timeout: "0" # 客户端空闲 N 秒后关闭连接 （0 禁用）
    workers: "8"
    daemonize: "no"
    maxclients: "10000"
    db-name: "change.me.db"
    slave-read-only: "yes"
    slave-priority: "100"
    tcp-backlog: "512"
    slave-serve-stale-data: "yes"
    slave-empty-db-before-fullsync: "no"
    purge-backup-on-fullsync: "no"
    max-io-mb: "500"
    max-db-size: "0"
    max-backup-to-keep: "1"
    max-backup-keep-hours: "24"
    slowlog-log-slower-than: "200000"
    profiling-sample-ratio: "0"
    profiling-sample-record-max-len: "256"
    profiling-sample-record-threshold-ms: "100"
    supervised: "no"
    auto-resize-block-and-sst: "yes"
    rocksdb.compression: "no"
    rocksdb.wal_ttl_seconds: "0"
    rocksdb.wal_size_limit_mb: "0"
  storage:
    size: 32Gi
    class: lightning
  resources:
    limits:
      cpu: 2
      memory: 8Gi
    requests:
      cpu: 1
      memory: 4Gi

`
	KVRocksCluster = `
apiVersion: kvrocks.com/v1alpha1
kind: KVRocks
metadata:
  name: kvrocks-cluster-1-test
  namespace: kvrocks
  labels:
    redis/system: xxxx
spec:
  image: xxxx
  imagePullPolicy: IfNotPresent
  master: 3
  replicas: 2
  type: cluster
  enableSentinel: true
  password: "123456"
  kvrocksConfig:
    bind: "0.0.0.0"
    port: "6379"
    timeout: "0" # 客户端空闲 N 秒后关闭连接 （0 禁用）
    workers: "8"
    daemonize: "no"
    maxclients: "10000"
    db-name: "change.me.db"
    slave-read-only: "yes"
    slave-priority: "100"
    tcp-backlog: "512"
    slave-serve-stale-data: "yes"
    slave-empty-db-before-fullsync: "no"
    purge-backup-on-fullsync: "no"
    max-io-mb: "500"
    max-db-size: "0"
    max-backup-to-keep: "1"
    max-backup-keep-hours: "24"
    slowlog-log-slower-than: "200000"
    profiling-sample-ratio: "0"
    profiling-sample-record-max-len: "256"
    profiling-sample-record-threshold-ms: "100"
    supervised: "no"
    auto-resize-block-and-sst: "yes"
    rocksdb.compression: "no"
    rocksdb.wal_ttl_seconds: "0"
    rocksdb.wal_size_limit_mb: "0"
  storage:
    size: 32Gi
    class: lightning
  resources:
    limits:
      cpu: 2
      memory: 8Gi
    requests:
      cpu: 1
      memory: 4Gi
`
)
