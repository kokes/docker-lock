package update_test

import (
	"os"
	"path/filepath"
	"testing"

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
		ExpectedImages          []parse.IImage
	}{
		{
			Name: "Image Without Digest",
			Images: []parse.IImage{
				parse.NewImage(
					kind.Dockerfile, "busybox", "latest", "", nil, nil,
				),
			},
			ExpectedNumNetworkCalls: 1,
			ExpectedImages: []parse.IImage{
				parse.NewImage(
					kind.Dockerfile, "busybox", "latest",
					test_utils.BusyboxLatestSHA, nil, nil,
				),
			},
		},
		{
			Name: "Image With Digest",
			Images: []parse.IImage{
				parse.NewImage(
					kind.Dockerfile, "busybox", "latest",
					test_utils.BusyboxLatestSHA, nil, nil,
				),
			},
			ExpectedNumNetworkCalls: 0,
			ExpectedImages: []parse.IImage{
				parse.NewImage(
					kind.Dockerfile, "busybox", "latest",
					test_utils.BusyboxLatestSHA, nil, nil,
				),
			},
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

			// wrapperManager, err := cmd_generate.DefaultWrapperManager(
			// 	client, cmd_generate.DefaultConfigPath(),
			// )
			// TODO: add back
			wrapperManager, err := DefaultWrapperManager(
				client, DefaultConfigPath(),
			)
			if err != nil {
				t.Fatal(err)
			}

			updater, err := update.NewImageDigestUpdater(wrapperManager)
			if err != nil {
				t.Fatal(err)
			}

			done := make(chan struct{})

			images := make(chan parse.IImage, len(test.Images))

			for _, image := range test.Images {
				images <- image
			}
			close(images)

			updatedImages := updater.UpdateDigests(images, done)

			var gotImages []parse.IImage

			for updatedImage := range updatedImages {
				if updatedImage.Err() != nil {
					t.Fatal(updatedImage.Err())
				}
				gotImages = append(gotImages, updatedImage)
			}

			test_utils.AssertImagesEqual(
				t, test.ExpectedImages, gotImages,
			)

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
