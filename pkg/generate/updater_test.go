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
	"github.com/safe-waters/docker-lock/pkg/testutils"
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
				parse.NewImage(kind.Dockerfile, "redis", "latest", "", map[string]interface{}{
					"position": 0,
					"path":     "Dockerfile",
				}, nil),
				parse.NewImage(kind.Dockerfile, "redis", "latest", testutils.RedisLatestSHA, map[string]interface{}{
					"position": 2,
					"path":     "Dockerfile",
				}, nil),
				parse.NewImage(kind.Dockerfile, "busybox", "latest", "", map[string]interface{}{
					"position": 1,
					"path":     "Dockerfile",
				}, nil),
				parse.NewImage(kind.Composefile, "busybox", "latest", "", map[string]interface{}{
					"position":    0,
					"path":        "docker-compose.yml",
					"serviceName": "svc",
				}, nil),
				parse.NewImage(kind.Composefile, "golang", "latest", "", map[string]interface{}{
					"position":    0,
					"path":        "docker-compose.yml",
					"serviceName": "anothersvc",
				}, nil),
				parse.NewImage(kind.Kubernetesfile, "busybox", "latest", "", map[string]interface{}{
					"path":          "pod.yml",
					"containerName": "busybox",
					"docPosition":   0,
					"imagePosition": 1,
				}, nil),
				parse.NewImage(kind.Kubernetesfile, "golang", "latest", testutils.GolangLatestSHA, map[string]interface{}{
					"path":          "pod.yml",
					"containerName": "golang",
					"docPosition":   0,
					"imagePosition": 0,
				}, nil),
			},
			Expected: []parse.IImage{
				parse.NewImage(kind.Dockerfile, "redis", "latest", testutils.RedisLatestSHA, map[string]interface{}{
					"position": 0,
					"path":     "Dockerfile",
				}, nil),
				parse.NewImage(kind.Dockerfile, "redis", "latest", testutils.RedisLatestSHA, map[string]interface{}{
					"position": 2,
					"path":     "Dockerfile",
				}, nil),
				parse.NewImage(kind.Dockerfile, "busybox", "latest", testutils.BusyboxLatestSHA, map[string]interface{}{
					"position": 1,
					"path":     "Dockerfile",
				}, nil),
				parse.NewImage(kind.Composefile, "busybox", "latest", testutils.BusyboxLatestSHA, map[string]interface{}{
					"position":    0,
					"path":        "docker-compose.yml",
					"serviceName": "svc",
				}, nil),
				parse.NewImage(kind.Composefile, "golang", "latest", testutils.GolangLatestSHA, map[string]interface{}{
					"position":    0,
					"path":        "docker-compose.yml",
					"serviceName": "anothersvc",
				}, nil),
				parse.NewImage(kind.Kubernetesfile, "busybox", "latest", testutils.BusyboxLatestSHA, map[string]interface{}{
					"path":          "pod.yml",
					"containerName": "busybox",
					"docPosition":   0,
					"imagePosition": 1,
				}, nil),
				parse.NewImage(kind.Kubernetesfile, "golang", "latest", testutils.GolangLatestSHA, map[string]interface{}{
					"path":          "pod.yml",
					"containerName": "golang",
					"docPosition":   0,
					"imagePosition": 0,
				}, nil),
			},
			ExpectedNumNetworkCalls: 3,
		},
	}

	for _, test := range tests {
		test := test

		t.Run(test.Name, func(t *testing.T) {
			t.Parallel()

			var gotNumNetworkCalls uint64

			server := testutils.MockServer(t, &gotNumNetworkCalls)
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
			sortImages(got)

			testutils.AssertImagesEqual(t, test.Expected, got)

			testutils.AssertNumNetworkCallsEqual(
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
