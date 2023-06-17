package util

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/url"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	kvrocksv1alpha1 "github.com/RocksLabs/kvrocks-operator/api/v1alpha1"
	"github.com/RocksLabs/kvrocks-operator/pkg/controllers"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	kruise "github.com/openkruise/kruise-api/apps/v1beta1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/portforward"
	"k8s.io/client-go/transport/spdy"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	DefaultClusterName          = "e2e-test"
	DefaultNamespace            = "kvrocks"
	DefaultInstallKruiseVersion = "1.4.0"
	DefaultMinikubeShell        = "start_minikube_cluster.sh"
)

var DefaultClusterConnectionConfig = ClusterConnectionConfig{
	KubeConfig:  "",
	Development: false,
}

type Config struct {
	InstallKubernetes        bool   // if true, install kubernetes cluster
	UninstallKubernetes      bool   // if true, uninstall kubernetes cluster, ignored when InstallKubernetes is false
	InstallKruise            bool   // if true, install kruise
	InstallKvrocksOperator   bool   // if true, install kvrocks operator
	UsingKvorcksOperatorHelm bool   // if true, using kvrocks operator helm chart
	InstallKruiseVersion     string // kruise version
	ClusterName              string // kubernetes cluster name
	Namespace                string // default namespace
	ClusterConnectionConfig         // kubernetes cluster connection config
}

type KubernetesEnv struct {
	Client           client.Client
	Config           *Config
	KubernetesConfig *rest.Config
}

type ClusterConnectionConfig struct {
	KubeConfig  string
	Development bool
}

func Start(config *Config) *KubernetesEnv {
	config = DefaultConfig(config)

	env := &KubernetesEnv{
		Config: config,
	}
	if config.InstallKubernetes {
		env.installKubernetes()
	}

	// scheme
	env.registerScheme()

	//config
	cfg, err := LoadKubernetesConfig(env.Config.ClusterConnectionConfig)
	Expect(err).NotTo(HaveOccurred())
	env.KubernetesConfig = cfg

	env.Client, err = client.New(env.KubernetesConfig, client.Options{Scheme: scheme.Scheme})
	Expect(err).NotTo(HaveOccurred())

	if config.InstallKruise {
		env.installKruise()
	}

	env.createNamespace()

	if config.InstallKvrocksOperator {
		env.installKvrocksOperator()
	}

	return env
}

func DefaultConfig(config *Config) *Config {
	if config == nil {
		config = &Config{
			InstallKubernetes:       true,
			UninstallKubernetes:     true,
			InstallKruise:           true,
			InstallKvrocksOperator:  true,
			InstallKruiseVersion:    DefaultInstallKruiseVersion,
			ClusterName:             DefaultClusterName,
			Namespace:               DefaultNamespace,
			ClusterConnectionConfig: DefaultClusterConnectionConfig,
		}
	}
	if config.ClusterName == "" {
		config.ClusterName = DefaultClusterName
	}
	if config.InstallKruiseVersion == "" {
		config.InstallKruiseVersion = DefaultInstallKruiseVersion
	}
	if config.Namespace == "" {
		config.Namespace = DefaultNamespace
	}
	if config.ClusterConnectionConfig == (ClusterConnectionConfig{}) {
		config.ClusterConnectionConfig = DefaultClusterConnectionConfig
	}
	return config
}

func (env *KubernetesEnv) installKubernetes() {
	fmt.Fprintf(GinkgoWriter, "install kubernetes cluster %s\n", env.Config.ClusterName)

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute*10)
	defer cancel() // This ensures resources are cleaned up.

	cmd := exec.CommandContext(ctx, getClusterInstallScriptPath(), "up", "-c", env.Config.ClusterName)
	done := make(chan error)

	go func() {
		done <- cmd.Run()
	}()

	select {
	case <-ctx.Done():
		Fail("install kubernetes cluster timeout")
	case err := <-done:
		if err != nil {
			Fail(fmt.Sprintf("Error occurred: %v", err)) // If an error occurred running the command, report it.
		}
	}
}

func (env *KubernetesEnv) registerScheme() {
	err := kvrocksv1alpha1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())
	err = kruise.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())
}

func (env *KubernetesEnv) Clear() {
	if !env.Config.InstallKubernetes || !env.Config.UninstallKubernetes {
		return
	}

	fmt.Fprintf(GinkgoWriter, "uninstall kubernetes cluster %s\n", env.Config.ClusterName)
	cmd := exec.Command(getClusterInstallScriptPath(), "down", "-c", env.Config.ClusterName)
	err := cmd.Run()
	Expect(err).NotTo(HaveOccurred())
}

func (env *KubernetesEnv) installKruise() {
	fmt.Fprintf(GinkgoWriter, "install kruise %s\n", env.Config.InstallKruiseVersion)

	// Add OpenKruise Helm repo
	_, err := HelmTool("repo", "add", "openkruise", "https://openkruise.github.io/charts")
	Expect(err).NotTo(HaveOccurred())

	// Update Helm repo
	_, err = HelmTool("repo", "update")
	Expect(err).NotTo(HaveOccurred())

	// Install OpenKruise using Helm
	if !env.isHelmInstalled("kruise") {
		_, err = HelmTool("install", "kruise", "openkruise/kruise", "--version", env.Config.InstallKruiseVersion)
		Expect(err).NotTo(HaveOccurred())
	}
}

func (env *KubernetesEnv) createNamespace() {
	fmt.Fprintf(GinkgoWriter, "create namespace %s\n", env.Config.Namespace)

	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: env.Config.Namespace,
		},
	}
	err := env.Client.Create(context.TODO(), ns)
	if err != nil && !errors.IsAlreadyExists(err) {
		Expect(err).NotTo(HaveOccurred())
	}
}

func (env *KubernetesEnv) installKvrocksOperator() {
	if !env.Config.InstallKvrocksOperator {
		return
	}
	fmt.Fprintf(GinkgoWriter, "install kvrocks operator\n")

	if env.Config.UsingKvorcksOperatorHelm {
		// TODO package the crd and operator and remove kubectlTool (jiayouxujin)
		if !env.isExistsCRD("kvrocks.kvrocks.com") {
			_, err := KubectlTool("apply", "-f", "../../../deploy/crd/templates/crd.yaml")
			Expect(err).NotTo(HaveOccurred())
		}

		if !env.isHelmInstalled("kvrocks-operator") {
			_, err := HelmTool("install", "kvrocks-operator", "../../../deploy/operator", "-n", env.Config.Namespace)
			Expect(err).NotTo(HaveOccurred())
		}
	} else {
		// Start the operator
		k8sManager, err := ctrl.NewManager(env.KubernetesConfig, ctrl.Options{
			Scheme: scheme.Scheme,
		})
		Expect(err).NotTo(HaveOccurred())
		err = (&controllers.KVRocksReconciler{
			Client: k8sManager.GetClient(),
			Scheme: k8sManager.GetScheme(),
			Log:    ctrl.Log,
		}).SetupWithManager(k8sManager, 4)
		Expect(err).NotTo(HaveOccurred())

		go func() {
			err = k8sManager.Start(ctrl.SetupSignalHandler())
		}()
		Expect(err).NotTo(HaveOccurred())
	}
}

func (env *KubernetesEnv) isExistsCRD(name string) bool {
	_, err := KubectlTool("get", "crd", name)
	if err != nil {
		if !strings.Contains(err.Error(), "not found") {
			Expect(err).NotTo(HaveOccurred())
		}
		return false
	}
	return true
}

func (env *KubernetesEnv) isHelmInstalled(name string) bool {
	_, err := HelmTool("status", name)
	if err != nil {
		if !strings.Contains(err.Error(), "release: not found") {
			Expect(err).NotTo(HaveOccurred())
		}
		return false
	}
	return true
}

func (env *KubernetesEnv) PortForward(pod *corev1.Pod, ports []string) (func(), error) {
	scheme := "https"
	path := fmt.Sprintf("/api/v1/namespaces/%s/pods/%s/portforward", pod.Namespace, pod.Name)
	hostIP := strings.TrimPrefix(env.KubernetesConfig.Host, "https://")

	serverURL := url.URL{Scheme: scheme, Path: path, Host: hostIP}

	roundTripper, upgrader, err := spdy.RoundTripperFor(env.KubernetesConfig)
	if err != nil {
		return nil, err
	}
	dialer := spdy.NewDialer(upgrader, &http.Client{Transport: roundTripper}, "POST", &serverURL)

	stopChan := make(chan struct{}, 1)
	readyChan := make(chan struct{})
	errChan := make(chan error, 1)
	out := new(bytes.Buffer)
	errOut := new(bytes.Buffer)
	go func() {
		pf, err := portforward.New(dialer, ports, stopChan, readyChan, out, errOut)
		if err != nil {
			errChan <- err
			return
		}
		err = pf.ForwardPorts()
		if err != nil {
			errChan <- err
		}
	}()

	select {
	case <-readyChan:
		if errOut.Len() > 0 {
			err = fmt.Errorf(errOut.String())
			return nil, err
		}
	case err := <-errChan:
		return nil, err
	case <-time.After(time.Second * 5):
		return nil, fmt.Errorf("timeout waiting for port-forward")
	}

	return func() {
		close(stopChan)
	}, nil
}

func LoadKubernetesConfig(connectionConfig ClusterConnectionConfig) (*rest.Config, error) {
	var cfg *rest.Config
	if connectionConfig.Development {
		config, err := clientcmd.BuildConfigFromFlags("", connectionConfig.KubeConfig)
		if err != nil {
			return nil, err
		}
		cfg = config
	} else {
		config, err := rest.InClusterConfig()
		if err != nil {
			return nil, err
		}
		cfg = config
	}
	return cfg, nil
}

func getClusterInstallScriptPath() string {
	_, currentFile, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(filepath.Dir(currentFile)), "script", DefaultMinikubeShell)
}
