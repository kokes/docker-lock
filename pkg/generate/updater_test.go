package generate_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/safe-waters/docker-lock/pkg/generate"
	"github.com/safe-waters/docker-lock/pkg/generate/parse"
	"github.com/safe-waters/docker-lock/pkg/generate/registry"
	"github.com/safe-waters/docker-lock/pkg/generate/registry/contrib"
	"github.com/safe-waters/docker-lock/pkg/generate/registry/firstparty"
	"github.com/safe-waters/docker-lock/pkg/generate/update"
	"github.com/safe-waters/docker-lock/pkg/kind"
	"github.com/safe-waters/docker-lock/pkg/test_utils"
)

func TestImageDigestUpdater(t *testing.T) {
	t.Parallel()

	tests := []struct {
		Name                    string
		Images                  []parse.IImage
		ExpectedNumNetworkCalls uint64
		Expected                []parse.IImage
	}{
		{
			Name: "Dockerfiles, Composefiles, And Kubernetesfiles",
			Images: []parse.IImage{
				test_utils.MakeImage(kind.Dockerfile, "redis", "latest", "", map[string]interface{}{
					"position": 0,
					"path":     "Dockerfile",
				}),
				test_utils.MakeImage(kind.Dockerfile, "redis", "latest", test_utils.RedisLatestSHA, map[string]interface{}{
					"position": 2,
					"path":     "Dockerfile",
				}),
				test_utils.MakeImage(kind.Dockerfile, "busybox", "latest", "", map[string]interface{}{
					"position": 1,
					"path":     "Dockerfile",
				}),
				test_utils.MakeImage(kind.Composefile, "busybox", "latest", "", map[string]interface{}{
					"position":    0,
					"path":        "docker-compose.yml",
					"serviceName": "svc",
				}),
				test_utils.MakeImage(kind.Composefile, "golang", "latest", "", map[string]interface{}{
					"position":    0,
					"path":        "docker-compose.yml",
					"serviceName": "anothersvc",
				}),
				test_utils.MakeImage(kind.Kubernetesfile, "busybox", "latest", "", map[string]interface{}{
					"path":          "pod.yml",
					"containerName": "busybox",
					"docPosition":   0,
					"imagePosition": 1,
				}),
				test_utils.MakeImage(kind.Kubernetesfile, "golang", "latest", test_utils.GolangLatestSHA, map[string]interface{}{
					"path":          "pod.yml",
					"containerName": "golang",
					"docPosition":   0,
					"imagePosition": 0,
				}),
			},
			Expected: []parse.IImage{
				test_utils.MakeImage(kind.Dockerfile, "redis", "latest", test_utils.RedisLatestSHA, map[string]interface{}{
					"position": 0,
					"path":     "Dockerfile",
				}),
				test_utils.MakeImage(kind.Dockerfile, "redis", "latest", test_utils.RedisLatestSHA, map[string]interface{}{
					"position": 2,
					"path":     "Dockerfile",
				}),
				test_utils.MakeImage(kind.Dockerfile, "busybox", "latest", test_utils.BusyboxLatestSHA, map[string]interface{}{
					"position": 1,
					"path":     "Dockerfile",
				}),
				test_utils.MakeImage(kind.Composefile, "busybox", "latest", test_utils.BusyboxLatestSHA, map[string]interface{}{
					"position":    0,
					"path":        "docker-compose.yml",
					"serviceName": "svc",
				}),
				test_utils.MakeImage(kind.Composefile, "golang", "latest", test_utils.GolangLatestSHA, map[string]interface{}{
					"position":    0,
					"path":        "docker-compose.yml",
					"serviceName": "anothersvc",
				}),
				test_utils.MakeImage(kind.Kubernetesfile, "busybox", "latest", test_utils.BusyboxLatestSHA, map[string]interface{}{
					"path":          "pod.yml",
					"containerName": "busybox",
					"docPosition":   0,
					"imagePosition": 1,
				}),
				test_utils.MakeImage(kind.Kubernetesfile, "golang", "latest", test_utils.GolangLatestSHA, map[string]interface{}{
					"path":          "pod.yml",
					"containerName": "golang",
					"docPosition":   0,
					"imagePosition": 0,
				}),
			},
			ExpectedNumNetworkCalls: 3,
		},
	}

	for _, test := range tests {
		test := test

		t.Run(test.Name, func(t *testing.T) {
			t.Parallel()

			var gotNumNetworkCalls uint64

			server := test_utils.MockServer(t, &gotNumNetworkCalls)
			defer server.Close()

			client := &registry.HTTPClient{
				Client:      server.Client(),
				RegistryURL: server.URL,
				TokenURL:    server.URL + "?scope=repository%s",
			}

			wrapperManager, err := DefaultWrapperManager(
				client, DefaultConfigPath(),
			)
			if err != nil {
				t.Fatal(err)
			}

			innerUpdater, err := update.NewImageDigestUpdater(wrapperManager)
			if err != nil {
				t.Fatal(err)
			}

			updater, err := generate.NewImageDigestUpdater(innerUpdater, false)
			if err != nil {
				t.Fatal(err)
			}

			done := make(chan struct{})

			anyImages := make(chan parse.IImage, len(test.Images))

			for _, anyImage := range test.Images {
				anyImages <- anyImage
			}
			close(anyImages)

			updatedImages := updater.UpdateDigests(anyImages, done)

			var got []parse.IImage

			for updatedImage := range updatedImages {
				if updatedImage.Err() != nil {
					t.Fatal(updatedImage.Err())
				}

				got = append(got, updatedImage)
			}

			sortImages(test.Expected)
			sortImages(test.Expected)
			sortImages(test.Expected)
			sortImages(got)
			sortImages(got)
			sortImages(got)

			test_utils.AssertImagesEqual(t, test.Expected, got)

			test_utils.AssertNumNetworkCallsEqual(
				t, test.ExpectedNumNetworkCalls, gotNumNetworkCalls,
			)
		})
	}
}

// TODO: remove all below

func DefaultWrapperManager(
	client *registry.HTTPClient,
	configPath string,
) (*registry.WrapperManager, error) {
	defaultWrapper, err := firstparty.DefaultWrapper(client, configPath)
	if err != nil {
		return nil, err
	}

	wrapperManager := registry.NewWrapperManager(defaultWrapper)
	wrapperManager.Add(firstparty.AllWrappers(client, configPath)...)
	wrapperManager.Add(contrib.AllWrappers(client, configPath)...)

	return wrapperManager, nil
}

// DefaultConfigPath returns the default location of docker's config.json
// for all platforms.
func DefaultConfigPath() string {
	if homeDir, err := os.UserHomeDir(); err == nil {
		configPath := filepath.Join(homeDir, ".docker", "config.json")
		if _, err := os.Stat(configPath); err != nil {
			return ""
		}

		return configPath
	}

	return ""
}
