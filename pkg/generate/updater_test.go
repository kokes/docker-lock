package generate_test

import (
	"testing"

	cmd_generate "github.com/safe-waters/docker-lock/cmd/generate"
	"github.com/safe-waters/docker-lock/pkg/generate"
	"github.com/safe-waters/docker-lock/pkg/generate/parse"
	"github.com/safe-waters/docker-lock/pkg/generate/registry"
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
				test_utils.MakeImage(kind.Dockerfile, "redis", "latest", redisLatestSHA, map[string]interface{}{
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
				test_utils.MakeImage(kind.Composefile, "busybox", "latest", "", map[string]interface{}{
					"path":          "pod.yml",
					"containerName": "busybox",
					"docPosition":   0,
					"imagePosition": 1,
				}),
				test_utils.MakeImage(kind.Composefile, "golang", "latest", golangLatestSHA, map[string]interface{}{
					"path":          "pod.yml",
					"containerName": "golang",
					"docPosition":   0,
					"imagePosition": 0,
				}),
			},
			Expected: []parse.IImage{
				test_utils.MakeImage(kind.Dockerfile, "redis", "latest", redisLatestSHA, map[string]interface{}{
					"position": 0,
					"path":     "Dockerfile",
				}),
				test_utils.MakeImage(kind.Dockerfile, "redis", "latest", redisLatestSHA, map[string]interface{}{
					"position": 2,
					"path":     "Dockerfile",
				}),
				test_utils.MakeImage(kind.Dockerfile, "busybox", "latest", busyboxLatestSHA, map[string]interface{}{
					"position": 1,
					"path":     "Dockerfile",
				}),
				test_utils.MakeImage(kind.Composefile, "busybox", "latest", busyboxLatestSHA, map[string]interface{}{
					"position":    0,
					"path":        "docker-compose.yml",
					"serviceName": "svc",
				}),
				test_utils.MakeImage(kind.Composefile, "golang", "latest", golangLatestSHA, map[string]interface{}{
					"position":    0,
					"path":        "docker-compose.yml",
					"serviceName": "anothersvc",
				}),
				test_utils.MakeImage(kind.Composefile, "busybox", "latest", busyboxLatestSHA, map[string]interface{}{
					"path":          "pod.yml",
					"containerName": "busybox",
					"docPosition":   0,
					"imagePosition": 1,
				}),
				test_utils.MakeImage(kind.Composefile, "golang", "latest", golangLatestSHA, map[string]interface{}{
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

			server := mockServer(t, &gotNumNetworkCalls)
			defer server.Close()

			client := &registry.HTTPClient{
				Client:      server.Client(),
				RegistryURL: server.URL,
				TokenURL:    server.URL + "?scope=repository%s",
			}

			wrapperManager, err := cmd_generate.DefaultWrapperManager(
				client, cmd_generate.DefaultConfigPath(),
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
				if updatedImage.Err != nil {
					t.Fatal(updatedImage.Err)
				}

				got = append(got, updatedImage)
			}

			sortedExpected := sortAnyImages(t, test.Expected)
			sortedGot := sortAnyImages(t, got)

			assertAnyImagesEqual(t, sortedExpected, sortedGot)

			assertNumNetworkCallsEqual(
				t, test.ExpectedNumNetworkCalls, gotNumNetworkCalls,
			)
		})
	}
}
