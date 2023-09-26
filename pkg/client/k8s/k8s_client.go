package k8s

import (
	"context"

	"github.com/go-logr/logr"
	k8sApiClient "sigs.k8s.io/controller-runtime/pkg/client"
)

var ctx = context.TODO()

// Client for k8s
type Client struct {
	client k8sApiClient.Client
	logger logr.Logger
}

func NewK8sClient(client k8sApiClient.Client, logger logr.Logger) *Client {
	return &Client{
		client: client,
		logger: logger,
	}
}
