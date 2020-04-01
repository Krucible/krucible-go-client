Krucible Go Client
==================

This is the Go client for [Krucible](https://usekrucible.com), the platform for
creating ephemeral Kubernetes clusters optimised for testing and development.

Installation
------------

The Krucible Go client is built using Go modules and thus that is the
recommended way of including the Krucible client into your project.

```
go get github.com/Krucible/krucible-go-client
```

Usage
-----

To get started, import the `krucible` package and create a client:
```
import "github.com/Krucible/krucible-go-client/krucible"

client := krucible.NewClient(krucible.ClientConfig{
	AccountID:    "4ad69a63-bb6c-49a8-9c6f-6a166fb5acff",
	APIKeyId:     "146d98c4-327d-4a2d-b85a-49590246e136",
	APIKeySecret: "0da8afd2911904aa9bad8862a3e7478a",
})
```
### Creating a cluster
You should then be able to create new Krucible clusters with `CreateCluster`:
```
createClusterResult, err := client.CreateCluster(krucible.CreateClusterConfig{
	DisplayName: "my-krucible-cluster",
})
```

The createClusterResult is a `krucible.CreateClusterResult` struct. This contains both a
`krucible.Cluster` struct, containing metadata about your cluster, and a
[`kubernetes.Clientset`](https://godoc.org/k8s.io/client-go/kubernetes#Clientset)
that is set up to connect to the new cluster.

Getting the cluster expiry time:
```
fmt.Println(createClusterResult.Cluster.ExpiresAt)
```

Listing the pods in the default namespace:
```
pods, err := createClusterResult.Clientset.CoreV1().
	Pods("default").
	List(
		context.Background(),
		metav1.ListOptions{},
	)
```

### Getting a cluster
Get existing clusters with `GetCluster`:
```
cluster, err = client.GetCluster("51b831d4-a9d6-4489-913e-6df70fcc8ea8")
```

Here `cluster` is a `krucible.Cluster` struct, containing metadata about the
cluster.

### Getting a cluster clientset
Get a Kubernetes [client-go](https://github.com/kubernetes/client-go)
[Clientset](https://godoc.org/k8s.io/client-go/kubernetes#Clientset),
configured to connect to a given cluster:
```
cs, err := client.GetClusterClientset("51b831d4-a9d6-4489-913e-6df70fcc8ea8")
```
