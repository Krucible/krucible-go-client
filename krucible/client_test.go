package krucible

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestNewClientDefaultURL(t *testing.T) {
	c := NewClient(ClientConfig{
		AccountID:    "",
		APIKeyId:     "",
		APIKeySecret: "",
	})
	assert.Equal(t, "https://usekrucible.com/api", c.config.BaseURL)
}

func TestNewClientUnauthorised(t *testing.T) {
	c := NewClient(ClientConfig{
		BaseURL:      "http://127.0.0.1:3000/api",
		AccountID:    "d3ff9b5d-d753-4997-9879-e29eaf46a489",
		APIKeyId:     "",
		APIKeySecret: "",
	})
	_, err := c.CreateCluster(CreateClusterConfig{
		DisplayName: "stuff",
	})
	assert.Equal(t, err.Error(), "Unexpected status code 404")
}

func TestNewClientAuthorised(t *testing.T) {
	c := NewClient(ClientConfig{
		BaseURL:      "http://127.0.0.1:3000/api",
		AccountID:    "4ad69a63-bb6c-49a8-9c6f-6a166fb5acff",
		APIKeyId:     "146d98c4-327d-4a2d-b85a-49590246e136",
		APIKeySecret: "0da8afd2911904aa9bad8862a3e7478a",
	})
	clusterResult, err := c.CreateCluster(CreateClusterConfig{
		DisplayName: "stuff",
	})
	assert.Nil(t, err)
	pods, err := clusterResult.Clientset.CoreV1().Pods("kube-system").List(context.Background(), metav1.ListOptions{})
	assert.Nil(t, err)
	assert.NotEmpty(t, pods.Items)
}
