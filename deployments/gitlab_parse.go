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

const GITLAB_PATH_URL string = "https://gitlab.com/api/v4/projects/1nce-tech%2Fplatform%2Fresearch%2Fbong-meta-proxy-pipeline%2Fdeployments%2F"
const DEPLOYMENT_TEMPLATE string = "deployment.yml.j2"

// Temporary solution for dstest and regsvc
const DSTEST_REPO_TAG string = "v0.6.2"
const REG_SVC_REPO_TAG string = "v5.3.4"

// Define a struct to match the YAML structure
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
		return Config{}, fmt.Errorf("failed to read YAML file %s: %v", filePath, err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return Config{}, fmt.Errorf("failed to parse YAML from %s: %v", filePath, err)
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

func main() {
	if len(os.Args) != 3 {
		log.Fatalf("Error: Invalid number of arguments.\nUsage: go run gitlab_parse.go <config-file> <gitlab-token>\nExample: go run gitlab_parse.go deployments.yml $GITLAB_TOKEN")
	}

	// Read the YAML file
	config, err := loadConfig(os.Args[1])
	if err != nil {
		log.Fatal(err)
	}

	// Access the parsed data
	for key, value := range config.Deployments {
		fmt.Printf("Key: %s, Ver: %s, Acc: %s, Reg: %s\n", key, value.Version, value.Account, value.Region)

		var envType string
		envType = getEnvType(key)
		// Compose the URL using fmt.Sprintf
		url := fmt.Sprintf("%s%s%%2F%s/repository/files/proxy.yml/raw?ref=%s&private_token=%s", GITLAB_PATH_URL, envType, key, value.Version, os.Args[2])

		// Make the HTTP GET request
		client := &http.Client{Timeout: 10 * time.Second}
		resp, err := client.Get(url)
		if err != nil {
			log.Printf("Failed to make HTTP request: %v\n", err)
			continue
		}
		defer resp.Body.Close()

		// Handle the response (e.g., read the body)
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Printf("Failed to read response body: %v\n", err)
			continue
		}
		var gtpproxy Gtpproxy
		err = yaml.Unmarshal(body, &gtpproxy)
		if err != nil {
			log.Fatalf("Failed to parse YAML: %v", err)
		}

		config := DeploymentTemplate{
			DEPLOYMENT_REPO_NAME: key,
			TERRAFORM_REPO_TAG:   config.Terraform,
			PROXY_REPO_TAG:       gtpproxy.Gtpproxy,
			DEPLOYMENT_REPO_TAG:  value.Version,
			CONFIG_PATH:          envType,
			TA_REPO_TAG:          gtpproxy.Trafficaccum,
			ALERTS_REPO_TAG:      gtpproxy.Alerts,
			DEPLOYMENT_REGION:    value.Region,
			GTP_PROXY_ACCOUNT_ID: value.Account,
			DSTEST_REPO_TAG:      DSTEST_REPO_TAG,
			REG_SVC_REPO_TAG:     REG_SVC_REPO_TAG,
		}

		// Load and execute the template
		t, err := template.ParseFiles(DEPLOYMENT_TEMPLATE)
		if err != nil {
			log.Fatalf("Failed to parse template: %v", err)
		}

		err = os.MkdirAll(envType, 0750)
		if err != nil {
			log.Fatal(err)
		}

		// Create the file path
		path := fmt.Sprintf("%s/%s.yml", envType, key)

		// Create the file
		file, err := os.Create(path)
		if err != nil {
			log.Fatalf("Failed to create file: %v", err)
		}
		defer file.Close()

		err = t.Execute(file, config)
		if err != nil {
			log.Fatalf("Failed to execute template: %v", err)
		}
	}
}
