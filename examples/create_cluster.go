package examples

import (
	"context"
	"fmt"

	"github.com/Krucible/krucible-go-client/krucible"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func main() {
	client := krucible.NewClient(krucible.ClientConfig{
		AccountID:    "4ad69a63-bb6c-49a8-9c6f-6a166fb5acff",
		APIKeyId:     "146d98c4-327d-4a2d-b85a-49590246e136",
		APIKeySecret: "0da8afd2911904aa9bad8862a3e7478a",
	})

	// create a cluster
	createClusterResult, err := client.CreateCluster(krucible.CreateClusterConfig{
		DisplayName: "my-krucible-cluster",
	})
	if err != nil {
		panic(err)
	}

	// print expiry date
	fmt.Println(createClusterResult.Cluster.ExpiresAt)

	// get pods in default namespace
	core := createClusterResult.Clientset.CoreV1()
	pods, err := core.Pods("default").List(
		context.Background(), metav1.ListOptions{},
	)
	fmt.Println(pods.Items)
}
