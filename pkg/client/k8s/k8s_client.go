package k8s

import (
	"context"

	"github.com/go-logr/logr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var ctx = context.TODO()

// Client for k8s
type Client struct {
	client client.Client
	logger logr.Logger
}

func NewK8sClient(client client.Client, logger logr.Logger) *Client {
	return &Client{
		client: client,
		logger: logger,
	}
}
