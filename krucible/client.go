package krucible

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"path"
	"time"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

type ClientConfig struct {
	BaseURL      string
	AccountID    string
	APIKeyId     string
	APIKeySecret string
}

type Client struct {
	accountURL string
	config     ClientConfig
	httpClient http.Client
}

func (c *Client) makeRequest(method, apiPath string) (*http.Response, error) {
	return c.makeRequestWithBody(method, apiPath, http.NoBody)
}

func (c *Client) makeRequestWithBody(method, apiPath string, body interface{}) (*http.Response, error) {
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return &http.Response{}, err
	}

	absUrl := c.accountURL + apiPath
	req, err := http.NewRequest(method, absUrl, bytes.NewBuffer(jsonBody))
	if err != nil {
		return &http.Response{}, err
	}

	req.Header.Add("Api-Key-Id", c.config.APIKeyId)
	req.Header.Add("Api-Key-Secret", c.config.APIKeySecret)
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Accept-Encoding", "gzip, deflate, br")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return &http.Response{}, err
	}
	return resp, nil
}

type CreateClusterConfig struct {
	DisplayName string `json:"displayName"`
}

// Cluster contains metadata about a Krucible cluster.
type Cluster struct {
	ID                string `json:"id"`
	DisplayName       string `json:"displayName"`
	State             string `json:"state"`
	ConnectionDetails struct {
		Server string `json:"server"`
	} `json:"connectionDetails"`
	CreatedAt time.Time `json:"createdAt"`
	ExpiresAt time.Time `json:"expiresAt"`
}

// GetCluster fetches metadata about the given Krucible cluster.
func (c *Client) GetCluster(id string) (result Cluster, err error) {
	resp, err := c.makeRequest("GET", "/clusters/"+id)
	if err != nil {
		return result, err
	}

	if resp.StatusCode != 200 {
		return result, fmt.Errorf("Unexpected status code %d", resp.StatusCode)
	}

	err = json.NewDecoder(resp.Body).Decode(&result)
	return result, err
}

// GetClusterClientset returns a set of clients for a given Krucible cluster.
// These can be used to connect to the cluster as usual.
func (c *Client) GetClusterClientset(id string) (result *kubernetes.Clientset, err error) {
	resp, err := c.makeRequest("GET", "/clusters/"+id+"/kube-config")
	if err != nil {
		return result, err
	}

	if resp.StatusCode != 200 {
		return result, fmt.Errorf("Unexpected status code %d", resp.StatusCode)
	}

	kubeConfigBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return result, err
	}

	kubeConfig, err := clientcmd.RESTConfigFromKubeConfig(kubeConfigBytes)
	if err != nil {
		return result, err
	}

	return kubernetes.NewForConfig(kubeConfig)
}

// CreateCluster creates a Krucible cluster with the given configuration. Both
// a cluster, containing metadata about the created cluster, and a client,
// configured for connectivity to the cluster, are returned, both of which
// should be valid providing that the returned error is nil.
func (c *Client) CreateCluster(createConfig CreateClusterConfig) (cluster Cluster, clientset *kubernetes.Clientset, err error) {
	resp, err := c.makeRequestWithBody("POST", "/clusters", CreateClusterConfig{
		DisplayName: createConfig.DisplayName,
	})
	if err != nil {
		return
	}

	if resp.StatusCode != 201 {
		return Cluster{}, nil, fmt.Errorf("Unexpected status code %d", resp.StatusCode)
	}

	err = json.NewDecoder(resp.Body).Decode(&cluster)
	if err != nil {
		return
	}

	for cluster.State == "provisioning" {
		time.Sleep(1)
		cluster, err = c.GetCluster(cluster.ID)
	}

	clientset, err = c.GetClusterClientset(cluster.ID)
	return
}

// NewClient creates a new Krucible client with the given connection
// information.
func NewClient(config ClientConfig) *Client {
	baseURL := config.BaseURL
	if baseURL == "" {
		baseURL = "https://usekrucible.com/api"
	}
	parsedURL, err := url.Parse(baseURL)
	if err != nil || !parsedURL.IsAbs() {
		parsedURL, _ = url.Parse("https://usekrucible.com/api")
	}

	accountURL := parsedURL.String() + path.Join(
		"/accounts",
		config.AccountID,
	)

	c := Client{
		accountURL: accountURL,
		config: ClientConfig{
			BaseURL:      parsedURL.String(),
			AccountID:    config.AccountID,
			APIKeyId:     config.APIKeyId,
			APIKeySecret: config.APIKeySecret,
		},
		httpClient: http.Client{},
	}
	return &c
}
