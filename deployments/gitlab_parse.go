package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"text/template"
	"time"

	"gopkg.in/yaml.v3"
)

// Constants for GitLab API URL, deployment template, and default timeout.
const (
	gitlabPathURL      = "https://gitlab.com/api/v4/projects/1nce-tech%2Fplatform%2Fresearch%2Fbong-meta-proxy-pipeline%2Fdeployments%2F"
	deploymentTemplate = "deployment.yml.j2"
	dstestRepoTag      = "v0.6.2"
	regSvcRepoTag      = "v5.3.4"
	defaultTimeout     = 10 * time.Second
)

// Config represents the structure of the YAML configuration file.
type Config struct {
	Deployments map[string]Deployment `yaml:"deployments"` // Map of deployments
	Terraform   string                `yaml:"terraform"`   // Terraform version
}

// Deployment represents the details of a single deployment.
type Deployment struct {
	Version string `yaml:"version"` // Version of the deployment
	Account string `yaml:"account"` // AWS account ID
	Region  string `yaml:"region"`  // AWS region
}

// DeploymentTemplate represents the data structure used to render the deployment template.
type DeploymentTemplate struct {
	DEPLOYMENT_REPO_NAME string // Name of the deployment repository
	TERRAFORM_REPO_TAG   string // Terraform repository tag
	PROXY_REPO_TAG       string // GTP Proxy repository tag
	DEPLOYMENT_REPO_TAG  string // Deployment repository tag
	CONFIG_PATH          string // Path to the configuration
	TA_REPO_TAG          string // Traffic Accumulator repository tag
	ALERTS_REPO_TAG      string // Alerts repository tag
	DEPLOYMENT_REGION    string // AWS region for the deployment
	GTP_PROXY_ACCOUNT_ID string // AWS account ID for GTP Proxy
	DSTEST_REPO_TAG      string // DSTest repository tag
	REG_SVC_REPO_TAG     string // Registration Service repository tag
}

// Gtpproxy represents the structure of the proxy configuration fetched from GitLab.
type Gtpproxy struct {
	Gtpproxy      string `yaml:"gtp-proxy"`                // GTP Proxy version
	Trafficaccum  string `yaml:"traffic-accumulator"`      // Traffic Accumulator version
	Alerts        string `yaml:"alerts"`                   // Alerts version
	Gtpregservice string `yaml:"gtp-registration-service"` // Registration Service version
}

// loadConfig reads and parses the YAML configuration file.
//
// Args:
//   - filePath: Path to the YAML configuration file.
//
// Returns:
//   - Config: Parsed configuration.
//   - error: Error if any occurred during reading or parsing.
func loadConfig(filePath string) (Config, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return Config{}, fmt.Errorf("failed to read YAML file %s: %w", filePath, err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return Config{}, fmt.Errorf("failed to parse YAML from %s: %w", filePath, err)
	}

	return config, nil
}

// getEnvType determines the environment type (dev, stg, prd) based on the deployment key.
//
// Args:
//   - key: Deployment key.
//
// Returns:
//   - string: Environment type (dev, stg, prd, or unknown).
func getEnvType(key string) string {
	switch {
	case strings.Contains(key, "dev"):
		return "dev"
	case strings.Contains(key, "stg"):
		return "stg"
	case strings.Contains(key, "prd"):
		return "prd"
	default:
		return "unknown"
	}
}

// fetchProxyConfig fetches the proxy configuration from GitLab.
//
// Args:
//   - url: URL to fetch the proxy configuration.
//   - token: GitLab access token.
//
// Returns:
//   - *Gtpproxy: Parsed proxy configuration.
//   - error: Error if any occurred during the HTTP request or parsing.
func fetchProxyConfig(url, token string) (*Gtpproxy, error) {
	client := &http.Client{Timeout: defaultTimeout}
	resp, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to make HTTP request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var gtpproxy Gtpproxy
	if err := yaml.Unmarshal(body, &gtpproxy); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	return &gtpproxy, nil
}

// createDeploymentFile creates a deployment file from a template.
//
// Args:
//   - envType: Environment type (dev, stg, prd).
//   - key: Deployment key.
//   - config: Deployment configuration data.
//
// Returns:
//   - error: Error if any occurred during file creation or template execution.
func createDeploymentFile(envType, key string, config DeploymentTemplate) error {
	if err := os.MkdirAll(envType, 0750); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	path := fmt.Sprintf("%s/%s.yml", envType, key)
	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	t, err := template.ParseFiles(deploymentTemplate)
	if err != nil {
		return fmt.Errorf("failed to parse template: %w", err)
	}

	if err := t.Execute(file, config); err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}

	return nil
}

// main is the entry point of the program.
func main() {
	if len(os.Args) != 3 {
		log.Fatalf("Error: Invalid number of arguments.\nUsage: go run gitlab_parse.go <config-file> <gitlab-token>\nExample: go run gitlab_parse.go deployments.yml $GITLAB_TOKEN")
	}

	configFile := os.Args[1]
	gitlabToken := os.Args[2]

	config, err := loadConfig(configFile)
	if err != nil {
		log.Fatal(err)
	}

	for key, value := range config.Deployments {
		fmt.Printf("Key: %s, Ver: %s, Acc: %s, Reg: %s\n", key, value.Version, value.Account, value.Region)

		envType := getEnvType(key)
		url := fmt.Sprintf("%s%s%%2F%s/repository/files/proxy.yml/raw?ref=%s&private_token=%s", gitlabPathURL, envType, key, value.Version, gitlabToken)

		gtpproxy, err := fetchProxyConfig(url, gitlabToken)
		if err != nil {
			log.Printf("Failed to fetch proxy config: %v\n", err)
			continue
		}

		deploymentConfig := DeploymentTemplate{
			DEPLOYMENT_REPO_NAME: key,
			TERRAFORM_REPO_TAG:   config.Terraform,
			PROXY_REPO_TAG:       gtpproxy.Gtpproxy,
			DEPLOYMENT_REPO_TAG:  value.Version,
			CONFIG_PATH:          envType,
			TA_REPO_TAG:          gtpproxy.Trafficaccum,
			ALERTS_REPO_TAG:      gtpproxy.Alerts,
			DEPLOYMENT_REGION:    value.Region,
			GTP_PROXY_ACCOUNT_ID: value.Account,
			DSTEST_REPO_TAG:      dstestRepoTag,
			REG_SVC_REPO_TAG:     regSvcRepoTag,
		}

		if err := createDeploymentFile(envType, key, deploymentConfig); err != nil {
			log.Fatalf("Failed to create deployment file: %v", err)
		}
	}
}
