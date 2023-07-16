package util

import (
	"fmt"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"path/filepath"

	kvrocksv1alpha1 "github.com/RocksLabs/kvrocks-operator/api/v1alpha1"
	k8sYaml "sigs.k8s.io/yaml"
)

const (
	DefaultClusterName   = "e2e-test"
	DefaultNamespace     = "kvrocks"
	DefaultKruiseVersion = "1.4.0"
	DefaultMinikubeShell = "start_minikube_cluster.sh"
	DefaulManifestDir    = "../../../examples/"
)

type Config struct {
	KruiseVersion string `yaml:"kruiseVersion"`
	ClusterName   string `yaml:"clusterName"`
	Namespace     string `yaml:"namespace"`
	KubeConfig    string `yaml:"kubeConfig"`
	ManifestDir   string `yaml:"manifestDir"`
}

func NewConfig(configFilePath string) (*Config, error) {
	config := &Config{}

	configData, err := ioutil.ReadFile(configFilePath)
	if err != nil {
		return nil, err
	}

	err = yaml.Unmarshal(configData, config)
	if err != nil {
		return nil, err
	}

	if config.ClusterName == "" {
		config.ClusterName = DefaultClusterName
	}
	if config.KruiseVersion == "" {
		config.KruiseVersion = DefaultKruiseVersion
	}
	if config.Namespace == "" {
		config.Namespace = DefaultNamespace
	}
	if config.ManifestDir == "" {
		config.ManifestDir = DefaulManifestDir
	}
	return config, nil
}

func (c *Config) ParseManifest(t kvrocksv1alpha1.KVRocksType) (*kvrocksv1alpha1.KVRocks, error) {
	path := filepath.Join(c.ManifestDir, string(t)+".yaml")
	instanceYamlFile, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	instance := &kvrocksv1alpha1.KVRocks{}
	err = k8sYaml.Unmarshal(instanceYamlFile, instance)
	if err != nil {
		return nil, err
	}
	if instance.GetNamespace() != c.Namespace {
		return nil, fmt.Errorf("namespace does not match the config")
	}
	return instance, nil
}
