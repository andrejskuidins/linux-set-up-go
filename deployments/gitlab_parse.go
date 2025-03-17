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

const (
	gitlabPathURL      = "https://gitlab.com/api/v4/projects/1nce-tech%2Fplatform%2Fresearch%2Fbong-meta-proxy-pipeline%2Fdeployments%2F"
	deploymentTemplate = "deployment.yml.j2"
	dstestRepoTag      = "v0.6.2"
	regSvcRepoTag      = "v5.3.4"
	defaultTimeout     = 10 * time.Second
)

type Config struct {
	Deployments map[string]Deployment `yaml:"deployments"`
	Terraform   string                `yaml:"terraform"`
}

type Deployment struct {
	Version string `yaml:"version"`
	Account string `yaml:"account"`
	Region  string `yaml:"region"`
}

type DeploymentTemplate struct {
	DEPLOYMENT_REPO_NAME string
	TERRAFORM_REPO_TAG   string
	PROXY_REPO_TAG       string
	DEPLOYMENT_REPO_TAG  string
	CONFIG_PATH          string
	TA_REPO_TAG          string
	ALERTS_REPO_TAG      string
	DEPLOYMENT_REGION    string
	GTP_PROXY_ACCOUNT_ID string
	DSTEST_REPO_TAG      string
	REG_SVC_REPO_TAG     string
}

type Gtpproxy struct {
	Gtpproxy      string `yaml:"gtp-proxy"`
	Trafficaccum  string `yaml:"traffic-accumulator"`
	Alerts        string `yaml:"alerts"`
	Gtpregservice string `yaml:"gtp-registration-service"`
}

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
