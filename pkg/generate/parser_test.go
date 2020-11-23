package generate_test

import (
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/safe-waters/docker-lock/pkg/generate"
	"github.com/safe-waters/docker-lock/pkg/generate/collect"
	"github.com/safe-waters/docker-lock/pkg/generate/parse"
	"github.com/safe-waters/docker-lock/pkg/kind"
	"github.com/safe-waters/docker-lock/pkg/test_utils"
)

func TestImageParser(t *testing.T) {
	t.Parallel()

	tests := []struct {
		Name                   string
		DockerfilePaths        []string
		ComposefilePaths       []string
		KubernetesfilePaths    []string
		ComposefileContents    [][]byte
		DockerfileContents     [][]byte
		KubernetesfileContents [][]byte
		Expected               []parse.IImage
	}{
		{
			Name: "Dockerfiles, Composefiles, And Kubernetesfiles", // nolint: lll

			DockerfilePaths:     []string{"Dockerfile"},
			ComposefilePaths:    []string{"docker-compose.yml"},
			KubernetesfilePaths: []string{"pod.yml"},
			DockerfileContents: [][]byte{
				[]byte(`
FROM ubuntu:bionic
FROM busybox
`),
			},
			ComposefileContents: [][]byte{
				[]byte(`
version: '3'
services:
  svc:
    image: busybox
  anothersvc:
    image: golang
`),
			},
			KubernetesfileContents: [][]byte{
				[]byte(`
apiVersion: v1
kind: Pod
metadata:
  name: test
  labels:
    app: test
spec:
  containers:
  - name: redis
    image: redis
    ports:
    - containerPort: 80
`),
			},
			Expected: []parse.IImage{
				test_utils.MakeImage(kind.Dockerfile, "ubuntu", "bionic", "", map[string]interface{}{
					"position": 0,
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
				test_utils.MakeImage(kind.Kubernetesfile, "redis", "latest", "", map[string]interface{}{
					"path":          "pod.yml",
					"imagePosition": 0,
					"docPosition":   0,
					"containerName": "redis",
				}),
			},
		},
	}

	for _, test := range tests {
		test := test

		t.Run(test.Name, func(t *testing.T) {
			t.Parallel()

			tempDir := test_utils.MakeTempDir(t, "")
			defer os.RemoveAll(tempDir)

			dockerfilePaths := test_utils.WriteFilesToTempDir(
				t, tempDir, test.DockerfilePaths, test.DockerfileContents,
			)
			composefilePaths := test_utils.WriteFilesToTempDir(
				t, tempDir, test.ComposefilePaths, test.ComposefileContents,
			)
			kubernetesfilePaths := test_utils.WriteFilesToTempDir(
				t, tempDir, test.KubernetesfilePaths,
				test.KubernetesfileContents,
			)

			anyPaths := make(
				chan collect.IPath,
				len(dockerfilePaths)+
					len(composefilePaths)+
					len(kubernetesfilePaths),
			)
			for _, path := range dockerfilePaths {
				anyPaths <- collect.NewPath(kind.Dockerfile, path, nil)
			}
			for _, path := range composefilePaths {
				anyPaths <- collect.NewPath(kind.Composefile, path, nil)
			}
			for _, path := range kubernetesfilePaths {
				anyPaths <- collect.NewPath(kind.Kubernetesfile, path, nil)
			}

			close(anyPaths)

			dockerfileImageParser := parse.NewDockerfileImageParser(kind.Dockerfile)
			composefileImageParser, err := parse.NewComposefileImageParser(kind.Composefile, dockerfileImageParser)
			if err != nil {
				t.Fatal(err)
			}
			kubernetesfileImageParser := parse.NewKubernetesfileImageParser(kind.Kubernetesfile)

			imageParser, err := generate.NewImageParser(dockerfileImageParser, composefileImageParser, kubernetesfileImageParser)

			if err != nil {
				t.Fatal(err)
			}

			done := make(chan struct{})
			defer close(done)

			anyImages := imageParser.ParseFiles(anyPaths, done)

			var got []parse.IImage

			for anyImage := range anyImages {
				if anyImage.Err() != nil {
					t.Fatal(anyImage.Err())
				}
				got = append(got, anyImage)
			}

			for _, anyImage := range test.Expected {
				metadata := anyImage.Metadata()
				metadata["path"] = filepath.Join(
					tempDir, anyImage.Metadata()["path"].(string),
				)

				anyImage.SetMetadata(metadata)
			}

			sortImages(test.Expected)
			sortImages(got)

			test_utils.AssertImagesEqual(t, test.Expected, got)
		})
	}
}

func sortImages(images []parse.IImage) {
	sort.Slice(images, func(i int, j int) bool {
		switch {
		case images[i].Kind() != images[j].Kind():
			return images[i].Kind() < images[j].Kind()
		case images[i].Metadata()["path"].(string) != images[j].Metadata()["path"].(string):
			return images[i].Metadata()["path"].(string) < images[j].Metadata()["path"].(string)
		}

		_, ok1 := images[i].Metadata()["serviceName"].(string)
		_, ok2 := images[j].Metadata()["serviceName"].(string)

		if ok1 && ok2 {
			return images[i].Metadata()["serviceName"].(string) < images[j].Metadata()["serviceName"].(string)
		}

		_, ok1 = images[i].Metadata()["position"].(int)
		_, ok2 = images[j].Metadata()["position"].(int)

		if ok1 && ok2 {
			return images[i].Metadata()["position"].(int) < images[j].Metadata()["position"].(int)
		}

		switch {
		case images[i].Metadata()["docPosition"].(int) != images[j].Metadata()["docPosition"].(int):
			return images[i].Metadata()["docPosition"].(int) <= images[j].Metadata()["docPosition"].(int)
		default:
			return images[i].Metadata()["imagePosition"].(int) < images[j].Metadata()["imagePosition"].(int)
		}
	})
}
