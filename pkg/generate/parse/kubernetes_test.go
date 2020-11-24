package parse_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/safe-waters/docker-lock/pkg/generate/collect"
	"github.com/safe-waters/docker-lock/pkg/generate/parse"
	"github.com/safe-waters/docker-lock/pkg/kind"
	"github.com/safe-waters/docker-lock/pkg/test_utils"
)

const kubernetesfileImageParserTestDir = "kubernetesfileParser-tests"

func TestKubernetesfileImageParser(t *testing.T) {
	t.Parallel()

	tests := []struct {
		Name                   string
		KubernetesfilePaths    []string
		KubernetesfileContents [][]byte
		Expected               []parse.IImage
		ShouldFail             bool
	}{
		{
			Name:                "Image Position",
			KubernetesfilePaths: []string{"pod.yaml"},
			KubernetesfileContents: [][]byte{
				[]byte(`apiVersion: v1
kind: Pod
metadata:
  name: test
  labels:
    app: test
spec:
  containers:
  - name: busybox
    image: busybox
    ports:
    - containerPort: 80
  - name: golang
    image: golang
    ports:
    - containerPort: 88
`),
			},
			Expected: []parse.IImage{
				test_utils.MakeImage(kind.Kubernetesfile, "busybox", "latest", "", map[string]interface{}{
					"path":          "pod.yaml",
					"docPosition":   0,
					"imagePosition": 0,
					"containerName": "busybox",
				}),
				test_utils.MakeImage(kind.Kubernetesfile, "golang", "latest", "", map[string]interface{}{
					"path":          "pod.yaml",
					"docPosition":   0,
					"imagePosition": 1,
					"containerName": "golang",
				}),
			},
		},
		{
			Name:                "Doc Position",
			KubernetesfilePaths: []string{"pod.yaml"},
			KubernetesfileContents: [][]byte{
				[]byte(`apiVersion: v1
kind: Pod
metadata:
  name: test
  labels:
    app: test
spec:
  containers:
  - name: busybox
    image: busybox
    ports:
    - containerPort: 80
  - name: golang
    image: golang
    ports:
    - containerPort: 88
---
apiVersion: v1
kind: Pod
metadata:
  name: test
  labels:
    app: test
spec:
  containers:
  - name: redis
    image: redis:1.0@sha256:123
    ports:
    - containerPort: 80
  - name: bash
    image: bash:v1
    ports:
    - containerPort: 88
`),
			},
			Expected: []parse.IImage{
				test_utils.MakeImage(kind.Kubernetesfile, "busybox", "latest", "", map[string]interface{}{
					"path":          "pod.yaml",
					"docPosition":   0,
					"imagePosition": 0,
					"containerName": "busybox",
				}),
				test_utils.MakeImage(kind.Kubernetesfile, "golang", "latest", "", map[string]interface{}{
					"path":          "pod.yaml",
					"docPosition":   0,
					"imagePosition": 1,
					"containerName": "golang",
				}),
				test_utils.MakeImage(kind.Kubernetesfile, "redis", "1.0", "123", map[string]interface{}{
					"path":          "pod.yaml",
					"docPosition":   1,
					"imagePosition": 0,
					"containerName": "redis",
				}),
				test_utils.MakeImage(kind.Kubernetesfile, "bash", "v1", "", map[string]interface{}{
					"path":          "pod.yaml",
					"docPosition":   1,
					"imagePosition": 1,
					"containerName": "bash",
				}),
			},
		},
		{
			Name:                "Multiple Files",
			KubernetesfilePaths: []string{"deployment.yaml", "pod.yaml"},
			KubernetesfileContents: [][]byte{
				[]byte(`apiVersion: apps/v1
kind: Deployment
metadata:
  labels:
    app: nginx
  name: nginx-deployment
spec:
  replicas: 3
  selector:
    matchLabels:
      app: nginx
  template:
    metadata:
      labels:
        app: nginx
    spec:
      containers:
      - image: nginx
        name: nginx
        ports:
        - containerPort: 80
`),
				[]byte(`---
apiVersion: v1
kind: Pod
metadata:
  name: test
  labels:
    app: test
spec:
  containers:
  - name: busybox
    image: busybox
    ports:
    - containerPort: 80
`),
			},
			Expected: []parse.IImage{
				test_utils.MakeImage(kind.Kubernetesfile, "nginx", "latest", "", map[string]interface{}{
					"path":          "deployment.yaml",
					"docPosition":   0,
					"imagePosition": 0,
					"containerName": "nginx",
				}),
				test_utils.MakeImage(kind.Kubernetesfile, "busybox", "latest", "", map[string]interface{}{
					"path":          "pod.yaml",
					"docPosition":   0,
					"imagePosition": 0,
					"containerName": "busybox",
				}),
			},
		},
	}

	for _, test := range tests { // nolint: dupl
		test := test

		t.Run(test.Name, func(t *testing.T) {
			t.Parallel()

			tempDir := test_utils.MakeTempDir(t, kubernetesfileImageParserTestDir)
			defer os.RemoveAll(tempDir)

			test_utils.MakeParentDirsInTempDirFromFilePaths(
				t, tempDir, test.KubernetesfilePaths,
			)
			pathsToParse := test_utils.WriteFilesToTempDir(
				t, tempDir, test.KubernetesfilePaths,
				test.KubernetesfileContents,
			)

			pathsToParseCh := make(chan collect.IPath, len(pathsToParse))
			for _, path := range pathsToParse {
				pathsToParseCh <- collect.NewPath(kind.Kubernetesfile, path, nil)
			}
			close(pathsToParseCh)

			done := make(chan struct{})

			kubernetesfileParser := parse.NewKubernetesfileImageParser()
			kubernetesfileImages := kubernetesfileParser.ParseFiles(
				pathsToParseCh, done,
			)

			var got []parse.IImage

			for kubernetesfileImage := range kubernetesfileImages {
				if test.ShouldFail {
					if kubernetesfileImage.Err() == nil {
						t.Fatal("expected error but did not get one")
					}

					return
				}

				if kubernetesfileImage.Err() != nil {
					close(done)
					t.Fatal(kubernetesfileImage.Err())
				}

				got = append(got, kubernetesfileImage)
			}

			test_utils.SortKubernetesfileImageParserResults(t, got)

			for _, kubernetesfileImage := range test.Expected {
				metadata := kubernetesfileImage.Metadata()
				metadata["path"] = filepath.Join(tempDir, metadata["path"].(string))
				kubernetesfileImage.SetMetadata(metadata)
			}

			test_utils.AssertImagesEqual(t, test.Expected, got)
		})
	}
}
