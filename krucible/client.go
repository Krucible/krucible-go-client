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

func getIntPointer(i int) *int {
	return &i
}

var OneHour *int = getIntPointer(1)
var TwoHours *int = getIntPointer(2)
var ThreeHours *int = getIntPointer(3)
var FourHours *int = getIntPointer(4)
var FiveHours *int = getIntPointer(5)
var SixHours *int = getIntPointer(6)
var Permanent *int = nil

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

func (c *Client) makeJSONRequestWithBody(method, apiPath string, body interface{}, expectedStatus int, target interface{}) error {
	resp, err := c.makeRequestWithBody(method, apiPath, body)
	defer resp.Body.Close()
	if err != nil {
		return err
	}

	if resp.StatusCode != expectedStatus {
		return fmt.Errorf("Unexpected response code: %d", resp.StatusCode)
	}

	return json.NewDecoder(resp.Body).Decode(target)
}

func (c *Client) makeJSONRequest(method, apiPath string, expectedStatus int, target interface{}) error {
	return c.makeJSONRequestWithBody(method, apiPath, http.NoBody, expectedStatus, target)
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

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return &http.Response{}, err
	}
	return resp, nil
}

type CreateClusterConfig struct {
	DisplayName string `json:"displayName"`

	// DurationInHours is the number of hours the cluster should be available
	// for. If the cluster should run indefinitely then supply a nil pointer,
	// otherwise an integer between 1 and 6 should be provided.
	DurationInHours *int `json:"durationInHours"` // pointer because it could be null
}

// Cluster contains metadata about a Krucible cluster.
type Cluster struct {
	ID                string `json:"id"`
	DisplayName       string `json:"displayName"`
	State             string `json:"state"`
	ConnectionDetails struct {
		Server               string `json:"server"`
		CertificateAuthority string `json:"certificateAuthority"`
		ClusterAuthToken     string `json:"clusterAuthToken"`
	} `json:"connectionDetails"`
	CreatedAt time.Time `json:"createdAt"`
	ExpiresAt time.Time `json:"expiresAt"`
}

// GetCluster fetches metadata about the given Krucible cluster.
func (c *Client) GetCluster(id string) (result Cluster, err error) {
	if id == "" {
		return result, fmt.Errorf("Cluster ID must be non-empty")
	}

	err = c.makeJSONRequest("GET", "/clusters/"+id, 200, &result)
	return result, err
}

func (c *Client) GetClusters() (result []Cluster, err error) {
	err = c.makeJSONRequest("GET", "/clusters/", 200, &result)
	return
}

// GetClusterKubeConfig returns a cluster's kubeconfig as a byte array.
func (c *Client) GetClusterKubeConfig(id string) (result []byte, err error) {
	resp, err := c.makeRequest("GET", "/clusters/"+id+"/kube-config")
	defer resp.Body.Close()
	if err != nil {
		return result, err
	}

	if resp.StatusCode != 200 {
		return result, fmt.Errorf("Unexpected status code %d", resp.StatusCode)
	}

	return ioutil.ReadAll(resp.Body)
}

// GetClusterClientset returns a set of clients for a given Krucible cluster.
// These can be used to connect to the cluster as usual.
func (c *Client) GetClusterClientset(id string) (result *kubernetes.Clientset, err error) {
	kubeConfigBytes, err := c.GetClusterKubeConfig(id)
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
	err = c.makeJSONRequestWithBody("POST", "/clusters", createConfig, 201, &cluster)
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

// DeleteCluster deletes a Krucible cluster with the given ID.
func (c *Client) DeleteCluster(clusterID string) (err error) {
	resp, err := c.makeRequest("DELETE", "/clusters/"+clusterID)
	if err != nil {
		return err
	}

	if resp.StatusCode != 202 {
		return fmt.Errorf("Could not delete cluster %s. Error code: %s", clusterID, resp.Status)
	}

	return nil
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
