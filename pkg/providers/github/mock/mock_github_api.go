package mock

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

// MockGitHubAPIContainer represents a mock GitHub API container.
type MockGitHubAPIContainer struct {
	Container testcontainers.Container
	URL       string
	Port      int
}

// NewMockGitHubAPIContainer creates a new mock GitHub API container.
func NewMockGitHubAPIContainer(ctx context.Context, log *logrus.Logger) (*MockGitHubAPIContainer, error) {
	req := testcontainers.ContainerRequest{
		Image:        "nginx:alpine",
		ExposedPorts: []string{"80/tcp"},
		WaitingFor:   wait.ForHTTP("/health").WithPort("80/tcp"),
		Files: []testcontainers.ContainerFile{
			{
				HostFilePath:      "/tmp/mock-github/nginx.conf",
				ContainerFilePath: "/etc/nginx/nginx.conf",
				FileMode:          0644,
			},
			{
				HostFilePath:      "/tmp/mock-github/api/repos/ethpandaops/dencun-devnets/contents",
				ContainerFilePath: "/usr/share/nginx/html/api/repos/ethpandaops/dencun-devnets/contents",
				FileMode:          0644,
			},
			{
				HostFilePath:      "/tmp/mock-github/api/repos/ethpandaops/dencun-devnets/contents/network-configs",
				ContainerFilePath: "/usr/share/nginx/html/api/repos/ethpandaops/dencun-devnets/contents/network-configs",
				FileMode:          0644,
			},
			{
				HostFilePath:      "/tmp/mock-github/health",
				ContainerFilePath: "/usr/share/nginx/html/health",
				FileMode:          0644,
			},
		},
	}

	// Create file structure for mock GitHub API
	if err := setupMockFiles(); err != nil {
		return nil, err
	}

	// Start container
	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		return nil, err
	}

	// Get host and port
	host, err := container.Host(ctx)
	if err != nil {
		return nil, err
	}

	mappedPort, err := container.MappedPort(ctx, "80/tcp")
	if err != nil {
		return nil, err
	}

	mockAPI := &MockGitHubAPIContainer{
		Container: container,
		URL:       fmt.Sprintf("http://%s:%s", host, mappedPort.Port()),
		Port:      mappedPort.Int(),
	}

	log.WithFields(logrus.Fields{
		"url":  mockAPI.URL,
		"port": mockAPI.Port,
	}).Info("Started mock GitHub API container")

	return mockAPI, nil
}

// Close closes and removes the mock GitHub API container.
func (m *MockGitHubAPIContainer) Close(ctx context.Context) error {
	return m.Container.Terminate(ctx)
}

// setupMockFiles creates the necessary files for the mock GitHub API container.
func setupMockFiles() error {
	// Create directory structure
	directories := []string{
		"/tmp/mock-github",
		"/tmp/mock-github/api",
		"/tmp/mock-github/api/repos",
		"/tmp/mock-github/api/repos/ethpandaops",
		"/tmp/mock-github/api/repos/ethpandaops/dencun-devnets",
		"/tmp/mock-github/api/repos/ethpandaops/dencun-devnets/contents",
		"/tmp/mock-github/api/repos/ethpandaops/dencun-devnets/contents/network-configs",
	}

	for _, dir := range directories {
		if err := createDirectory(dir); err != nil {
			return err
		}
	}

	// Create nginx.conf
	nginxConfig := `
user  nginx;
worker_processes  auto;

error_log  /var/log/nginx/error.log notice;
pid        /var/run/nginx.pid;

events {
    worker_connections  1024;
}

http {
    include       /etc/nginx/mime.types;
    default_type  application/json;

    log_format  main  '$remote_addr - $remote_user [$time_local] "$request" '
                      '$status $body_bytes_sent "$http_referer" '
                      '"$http_user_agent" "$http_x_forwarded_for"';

    access_log  /var/log/nginx/access.log  main;

    sendfile        on;
    keepalive_timeout  65;

    server {
        listen       80;
        server_name  localhost;

        location / {
            root   /usr/share/nginx/html;
            index  index.html index.htm;
            add_header Content-Type application/json;
        }

        location /health {
            root   /usr/share/nginx/html;
            add_header Content-Type text/plain;
        }

        error_page   500 502 503 504  /50x.html;
        location = /50x.html {
            root   /usr/share/nginx/html;
        }
    }
}
`
	if err := writeFile("/tmp/mock-github/nginx.conf", nginxConfig); err != nil {
		return err
	}

	// Create health check file
	if err := writeFile("/tmp/mock-github/health", "OK"); err != nil {
		return err
	}

	// Create repository root contents
	rootContents := []RepoContent{
		{Name: ".editorconfig", Type: "file", HTMLURL: "https://github.com/ethpandaops/dencun-devnets/blob/main/.editorconfig"},
		{Name: ".gitattributes", Type: "file", HTMLURL: "https://github.com/ethpandaops/dencun-devnets/blob/main/.gitattributes"},
		{Name: ".github", Type: "dir", HTMLURL: "https://github.com/ethpandaops/dencun-devnets/tree/main/.github"},
		{Name: ".gitignore", Type: "file", HTMLURL: "https://github.com/ethpandaops/dencun-devnets/blob/main/.gitignore"},
		{Name: "README.md", Type: "file", HTMLURL: "https://github.com/ethpandaops/dencun-devnets/blob/main/README.md"},
		{Name: "network-configs", Type: "dir", HTMLURL: "https://github.com/ethpandaops/dencun-devnets/tree/main/network-configs"},
	}

	rootContentsJSON, err := json.MarshalIndent(rootContents, "", "  ")
	if err != nil {
		return err
	}

	if err := writeFile("/tmp/mock-github/api/repos/ethpandaops/dencun-devnets/contents", string(rootContentsJSON)); err != nil {
		return err
	}

	// Create network-configs contents
	networkConfigs := []RepoContent{
		{Name: "devnet-10", Type: "dir", HTMLURL: "https://github.com/ethpandaops/dencun-devnets/tree/main/network-configs/devnet-10"},
		{Name: "devnet-11", Type: "dir", HTMLURL: "https://github.com/ethpandaops/dencun-devnets/tree/main/network-configs/devnet-11"},
		{Name: "devnet-12", Type: "dir", HTMLURL: "https://github.com/ethpandaops/dencun-devnets/tree/main/network-configs/devnet-12"},
		{Name: "devnet-4", Type: "dir", HTMLURL: "https://github.com/ethpandaops/dencun-devnets/tree/main/network-configs/devnet-4"},
		{Name: "devnet-5", Type: "dir", HTMLURL: "https://github.com/ethpandaops/dencun-devnets/tree/main/network-configs/devnet-5"},
		{Name: "gsf-1", Type: "dir", HTMLURL: "https://github.com/ethpandaops/dencun-devnets/tree/main/network-configs/gsf-1"},
		{Name: "gsf-2", Type: "dir", HTMLURL: "https://github.com/ethpandaops/dencun-devnets/tree/main/network-configs/gsf-2"},
		{Name: "msf-1", Type: "dir", HTMLURL: "https://github.com/ethpandaops/dencun-devnets/tree/main/network-configs/msf-1"},
		{Name: "sepolia-sf1", Type: "dir", HTMLURL: "https://github.com/ethpandaops/dencun-devnets/tree/main/network-configs/sepolia-sf1"},
	}

	networkConfigsJSON, err := json.MarshalIndent(networkConfigs, "", "  ")
	if err != nil {
		return err
	}

	if err := writeFile("/tmp/mock-github/api/repos/ethpandaops/dencun-devnets/contents/network-configs", string(networkConfigsJSON)); err != nil {
		return err
	}

	return nil
}

// RepoContent represents a GitHub repository content item.
type RepoContent struct {
	Name    string `json:"name"`
	Path    string `json:"path,omitempty"`
	Type    string `json:"type"`
	HTMLURL string `json:"html_url"`
}

// Helper function to create directory
func createDirectory(path string) error {
	return execCommand("mkdir", "-p", path)
}

// Helper function to write file
func writeFile(path, content string) error {
	return execCommand("bash", "-c", fmt.Sprintf("echo '%s' > %s", strings.ReplaceAll(content, "'", "'\\''"), path))
}

// Helper function to execute command
func execCommand(command string, args ...string) error {
	allArgs := append([]string{command}, args...)
	cmd := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()
		
		output, err := exec.CommandContext(ctx, allArgs[0], allArgs[1:]...).CombinedOutput()
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, "Error executing command: %v\nOutput: %s", err, output)
			return
		}
		
		w.WriteHeader(http.StatusOK)
		w.Write(output)
	})
	
	// Start a temporary server to handle the command execution
	server := &http.Server{Addr: ":0", Handler: cmd}
	go server.ListenAndServe()
	defer server.Shutdown(context.Background())
	
	// Execute the command via HTTP request
	resp, err := http.Get("http://localhost" + server.Addr)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("command failed: %s", body)
	}
	
	return nil
}