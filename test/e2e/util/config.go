package util

import (
	"gopkg.in/yaml.v2"
	"io/ioutil"
)

const (
	DefaultClusterName   = "e2e-test"
	DefaultNamespace     = "kvrocks"
	DefaultKruiseVersion = "1.4.0"
	DefaultMinikubeShell = "start_minikube_cluster.sh"
)

type Config struct {
	KruiseVersion string `yaml:"kruiseVersion"`
	ClusterName   string `yaml:"clusterName"`
	Namespace     string `yaml:"namespace"`
	KubeConfig    string `yaml:"kubeConfig"`
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
	return config, nil
}
