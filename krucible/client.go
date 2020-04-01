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

type client struct {
	accountURL string
	config     ClientConfig
	httpClient http.Client
}

func (c *client) makeRequest(method, apiPath string) (*http.Response, error) {
	return c.makeRequestWithBody(method, apiPath, http.NoBody)
}

func (c *client) makeRequestWithBody(method, apiPath string, body interface{}) (*http.Response, error) {
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

type Cluster struct {
	ID                string `json:"id"`
	DisplayName       string `json:"displayName"`
	State             string `json:"state"`
	ConnectionDetails struct {
		Server string `json:"server"`
	} `json:"connectionDetails"`
	CreatedAt string `json:"createdAt"`
	ExpiresAt string `json:"expiresAt"`
}

type CreateClusterResult struct {
	Cluster   Cluster
	Clientset *kubernetes.Clientset
}

func (c *client) GetCluster(id string) (result Cluster, err error) {
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

func (c *client) GetClusterClientset(id string) (result *kubernetes.Clientset, err error) {
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

func (c *client) CreateCluster(createConfig CreateClusterConfig) (CreateClusterResult, error) {
	resp, err := c.makeRequestWithBody("POST", "/clusters", CreateClusterConfig{
		DisplayName: createConfig.DisplayName,
	})
	if err != nil {
		return CreateClusterResult{}, err
	}

	if resp.StatusCode != 201 {
		return CreateClusterResult{}, fmt.Errorf("Unexpected status code %d", resp.StatusCode)
	}

	var result CreateClusterResult
	err = json.NewDecoder(resp.Body).Decode(&result.Cluster)
	if err != nil {
		return CreateClusterResult{}, err
	}

	for result.Cluster.State == "provisioning" {
		time.Sleep(1)
		result.Cluster, err = c.GetCluster(result.Cluster.ID)
	}

	result.Clientset, err = c.GetClusterClientset(result.Cluster.ID)
	return result, nil
}

func NewClient(config ClientConfig) *client {
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

	c := client{
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
