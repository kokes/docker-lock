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

const composefileImageParserTestDir = "composefileParser-tests"

func TestComposefileImageParser(t *testing.T) {
	t.Parallel()

	tests := []struct {
		Name                 string
		EnvironmentVariables map[string]string
		DotEnvContents       [][]byte
		ComposefilePaths     []string
		ComposefileContents  [][]byte
		DockerfilePaths      []string
		DockerfileContents   [][]byte
		Expected             []parse.IImage
	}{
		{
			Name:             "Image",
			ComposefilePaths: []string{"docker-compose.yml"},
			ComposefileContents: [][]byte{
				[]byte(`
version: '3'
services:
  svc:
    image: busybox
`),
			},
			Expected: []parse.IImage{
				test_utils.MakeImage(kind.Composefile, "busybox", "latest", "", map[string]interface{}{
					"path":            "docker-compose.yml",
					"servicePosition": 0,
					"serviceName":     "svc",
				}),
			},
		},
		{
			Name:             "Scratch",
			ComposefilePaths: []string{"docker-compose.yml"},
			ComposefileContents: [][]byte{
				[]byte(`
version: '3'
services:
  svc:
    image: scratch
`),
			},
			Expected: []parse.IImage{
				test_utils.MakeImage(kind.Composefile, "scratch", "", "", map[string]interface{}{
					"path":            "docker-compose.yml",
					"servicePosition": 0,
					"serviceName":     "svc",
				}),
			},
		},
		{
			Name:             "Build",
			ComposefilePaths: []string{"docker-compose.yml"},
			ComposefileContents: [][]byte{
				[]byte(`
version: '3'
services:
  svc:
    image: unused
    build: ./build
`),
			},
			DockerfilePaths:    []string{filepath.Join("build", "Dockerfile")},
			DockerfileContents: [][]byte{[]byte(`FROM busybox`)},
			Expected: []parse.IImage{
				test_utils.MakeImage(kind.Composefile, "busybox", "latest", "", map[string]interface{}{
					"path":            "docker-compose.yml",
					"servicePosition": 0,
					"serviceName":     "svc",
					"dockerfilePath":  filepath.Join("build", "Dockerfile"),
				}),
			},
		},
		{
			Name:             "Context",
			ComposefilePaths: []string{"docker-compose.yml"},
			ComposefileContents: [][]byte{
				[]byte(`
version: '3'
services:
  svc:
    image: unused
    build:
      context: ./dockerfile
`),
			},
			DockerfilePaths: []string{
				filepath.Join("dockerfile", "Dockerfile"),
			},
			DockerfileContents: [][]byte{[]byte(`FROM busybox`)},
			Expected: []parse.IImage{
				test_utils.MakeImage(kind.Composefile, "busybox", "latest", "", map[string]interface{}{
					"path":            "docker-compose.yml",
					"servicePosition": 0,
					"serviceName":     "svc",
					"dockerfilePath":  filepath.Join("dockerfile", "Dockerfile"),
				}),
			},
		},
		{
			Name:             "Context Dockerfile",
			ComposefilePaths: []string{"docker-compose.yml"},
			ComposefileContents: [][]byte{
				[]byte(`
version: '3'
services:
  svc:
    image: unused
    build:
      context: ./dockerfile
      dockerfile: Dockerfile
`),
			},
			DockerfilePaths: []string{
				filepath.Join("dockerfile", "Dockerfile"),
			},
			DockerfileContents: [][]byte{[]byte(`FROM busybox`)},
			Expected: []parse.IImage{
				test_utils.MakeImage(kind.Composefile, "busybox", "latest", "", map[string]interface{}{
					"path":            "docker-compose.yml",
					"servicePosition": 0,
					"serviceName":     "svc",
					"dockerfilePath":  filepath.Join("dockerfile", "Dockerfile"),
				}),
			},
		},
		{
			Name: "Env",
			EnvironmentVariables: map[string]string{
				"ENV_CONTEXT": "dockerfile",
			},
			ComposefilePaths: []string{"docker-compose.yml"},
			ComposefileContents: [][]byte{
				[]byte(`
version: '3'
services:
  svc:
    image: unused
    build:
      context: ./${ENV_CONTEXT}
`),
			},
			DockerfilePaths: []string{
				filepath.Join("dockerfile", "Dockerfile"),
			},
			DockerfileContents: [][]byte{[]byte(`FROM busybox`)},
			Expected: []parse.IImage{
				test_utils.MakeImage(kind.Composefile, "busybox", "latest", "", map[string]interface{}{
					"path":            "docker-compose.yml",
					"servicePosition": 0,
					"serviceName":     "svc",
					"dockerfilePath":  filepath.Join("dockerfile", "Dockerfile"),
				}),
			},
		},
		{
			Name:             "Dot Env",
			ComposefilePaths: []string{"docker-compose.yml"},
			DotEnvContents: [][]byte{
				[]byte(`
DOT_ENV_IMAGE=busybox
`),
			},
			ComposefileContents: [][]byte{
				[]byte(`
version: '3'
services:
  svc:
    image: ${DOT_ENV_IMAGE}
`),
			},
			Expected: []parse.IImage{
				test_utils.MakeImage(kind.Composefile, "busybox", "latest", "", map[string]interface{}{
					"path":            "docker-compose.yml",
					"servicePosition": 0,
					"serviceName":     "svc",
				}),
			},
		},
		{
			Name:             "Os Env Overrides Dot Env",
			ComposefilePaths: []string{"docker-compose.yml"},
			EnvironmentVariables: map[string]string{
				"OS_ENV_OVERRIDES_DOT_ENV_IMAGE": "busybox",
			},
			DotEnvContents: [][]byte{
				[]byte(`
OS_ENV_OVERRIDES_DOT_ENV_IMAGE=ubuntu
`),
			},
			ComposefileContents: [][]byte{
				[]byte(`
version: '3'
services:
  svc:
    image: ${OS_ENV_OVERRIDES_DOT_ENV_IMAGE}
`),
			},
			Expected: []parse.IImage{
				test_utils.MakeImage(kind.Composefile, "busybox", "latest", "", map[string]interface{}{
					"path":            "docker-compose.yml",
					"servicePosition": 0,
					"serviceName":     "svc",
				}),
			},
		},
		{
			Name: "Dot Env Args Env List",
			DotEnvContents: [][]byte{
				[]byte(`
DOT_ENV_ARGS_ENV_LIST_IMAGE=busybox
`),
			},
			ComposefilePaths: []string{"docker-compose.yml"},
			ComposefileContents: [][]byte{
				[]byte(`
version: '3'
services:
  svc:
    image: unused
    build:
      context: .
      args:
      - DOT_ENV_ARGS_ENV_LIST_IMAGE
`),
			},
			DockerfilePaths: []string{"Dockerfile"},
			DockerfileContents: [][]byte{[]byte(`
ARG DOT_ENV_ARGS_ENV_LIST_IMAGE
FROM ${DOT_ENV_ARGS_ENV_LIST_IMAGE}
`)},
			Expected: []parse.IImage{
				test_utils.MakeImage(kind.Composefile, "busybox", "latest", "", map[string]interface{}{
					"path":            "docker-compose.yml",
					"servicePosition": 0,
					"serviceName":     "svc",
					"dockerfilePath":  "Dockerfile",
				}),
			},
		},
		{
			Name: "Args Env List",
			EnvironmentVariables: map[string]string{
				"ARGS_ENV_LIST_IMAGE": "busybox",
			},
			ComposefilePaths: []string{"docker-compose.yml"},
			ComposefileContents: [][]byte{
				[]byte(`
version: '3'
services:
  svc:
    image: unused
    build:
      context: .
      args:
      - ARGS_ENV_LIST_IMAGE
`),
			},
			DockerfilePaths: []string{"Dockerfile"},
			DockerfileContents: [][]byte{[]byte(`
ARG ARGS_ENV_LIST_IMAGE
FROM ${ARGS_ENV_LIST_IMAGE}
`)},
			Expected: []parse.IImage{
				test_utils.MakeImage(kind.Composefile, "busybox", "latest", "", map[string]interface{}{
					"path":            "docker-compose.yml",
					"servicePosition": 0,
					"serviceName":     "svc",
					"dockerfilePath":  "Dockerfile",
				}),
			},
		},
		{
			Name:             "Args Key Val List",
			ComposefilePaths: []string{"docker-compose.yml"},
			ComposefileContents: [][]byte{
				[]byte(`
version: '3'
services:
  svc:
    image: unused
    build:
      context: .
      args:
      - IMAGE=busybox
`),
			},
			DockerfilePaths: []string{"Dockerfile"},
			DockerfileContents: [][]byte{[]byte(`
ARG IMAGE
FROM ${IMAGE}
`)},
			Expected: []parse.IImage{
				test_utils.MakeImage(kind.Composefile, "busybox", "latest", "", map[string]interface{}{
					"path":            "docker-compose.yml",
					"servicePosition": 0,
					"serviceName":     "svc",
					"dockerfilePath":  "Dockerfile",
				}),
			},
		},
		{
			Name:             "Args Key Val Map",
			ComposefilePaths: []string{"docker-compose.yml"},
			ComposefileContents: [][]byte{
				[]byte(`
version: '3'
services:
  svc:
    image: unused
    build:
      context: .
      args:
        IMAGE: busybox
`),
			},
			DockerfilePaths: []string{"Dockerfile"},
			DockerfileContents: [][]byte{[]byte(`
ARG IMAGE
FROM ${IMAGE}
`)},
			Expected: []parse.IImage{
				test_utils.MakeImage(kind.Composefile, "busybox", "latest", "", map[string]interface{}{
					"path":            "docker-compose.yml",
					"servicePosition": 0,
					"serviceName":     "svc",
					"dockerfilePath":  "Dockerfile",
				}),
			},
		},
		{
			Name:             "Args Override",
			ComposefilePaths: []string{"docker-compose.yml"},
			ComposefileContents: [][]byte{
				[]byte(`
version: '3'
services:
  svc:
    image: unused
    build:
      context: .
      args:
        IMAGE: busybox
`),
			},
			DockerfilePaths: []string{"Dockerfile"},
			DockerfileContents: [][]byte{[]byte(`
ARG IMAGE=ubuntu
FROM ${IMAGE}
`)},
			Expected: []parse.IImage{
				test_utils.MakeImage(kind.Composefile, "busybox", "latest", "", map[string]interface{}{
					"path":            "docker-compose.yml",
					"servicePosition": 0,
					"serviceName":     "svc",
					"dockerfilePath":  "Dockerfile",
				}),
			},
		},
		{
			Name:             "Args No Arg",
			ComposefilePaths: []string{"docker-compose.yml"},
			ComposefileContents: [][]byte{
				[]byte(`
version: '3'
services:
  svc:
    image: unused
    build:
      context: .
`),
			},
			DockerfilePaths: []string{"Dockerfile"},
			DockerfileContents: [][]byte{[]byte(`
ARG IMAGE=busybox
FROM ${IMAGE}
`)},
			Expected: []parse.IImage{
				test_utils.MakeImage(kind.Composefile, "busybox", "latest", "", map[string]interface{}{
					"path":            "docker-compose.yml",
					"servicePosition": 0,
					"serviceName":     "svc",
					"dockerfilePath":  "Dockerfile",
				}),
			},
		},
		{
			Name: "Multiple Files",
			ComposefilePaths: []string{
				"docker-compose-one.yml", "docker-compose-two.yml",
			},
			ComposefileContents: [][]byte{
				[]byte(`
version: '3'
services:
  svc-one:
    image: unused
    build: ./one
`),
				[]byte(`
version: '3'
services:
  svc-two:
    image: unused
    build: ./two
`),
			},
			DockerfilePaths: []string{
				filepath.Join("one", "Dockerfile"),
				filepath.Join("two", "Dockerfile"),
			},
			DockerfileContents: [][]byte{
				[]byte(`FROM busybox`), []byte(`FROM busybox`),
			},
			Expected: []parse.IImage{
				test_utils.MakeImage(kind.Composefile, "busybox", "latest", "", map[string]interface{}{
					"path":            "docker-compose-one.yml",
					"servicePosition": 0,
					"serviceName":     "svc-one",
					"dockerfilePath":  filepath.Join("one", "Dockerfile"),
				}),
				test_utils.MakeImage(kind.Composefile, "busybox", "latest", "", map[string]interface{}{
					"path":            "docker-compose-two.yml",
					"servicePosition": 0,
					"serviceName":     "svc-two",
					"dockerfilePath":  filepath.Join("two", "Dockerfile"),
				}),
			},
		},
		{
			Name:             "Multiple Services",
			ComposefilePaths: []string{"docker-compose.yml"},
			ComposefileContents: [][]byte{
				[]byte(`
version: '3'
services:
  svc-one:
    image: unused
    build: ./one
  svc-two:
    image: unused
    build: ./two
`),
			},
			DockerfilePaths: []string{
				filepath.Join("one", "Dockerfile"),
				filepath.Join("two", "Dockerfile"),
			},
			DockerfileContents: [][]byte{
				[]byte(`FROM busybox`), []byte(`FROM busybox`),
			},
			Expected: []parse.IImage{
				test_utils.MakeImage(kind.Composefile, "busybox", "latest", "", map[string]interface{}{
					"path":            "docker-compose.yml",
					"servicePosition": 0,
					"serviceName":     "svc-one",
					"dockerfilePath":  filepath.Join("one", "Dockerfile"),
				}),
				test_utils.MakeImage(kind.Composefile, "busybox", "latest", "", map[string]interface{}{
					"path":            "docker-compose.yml",
					"servicePosition": 0,
					"serviceName":     "svc-two",
					"dockerfilePath":  filepath.Join("two", "Dockerfile"),
				}),
			},
		},
	}

	for _, test := range tests {
		test := test

		t.Run(test.Name, func(t *testing.T) {
			t.Parallel()

			tempDir := test_utils.MakeTempDir(t, composefileImageParserTestDir)
			defer os.RemoveAll(tempDir)

			for k, v := range test.EnvironmentVariables {
				os.Setenv(k, v)
			}

			test_utils.MakeParentDirsInTempDirFromFilePaths(
				t, tempDir, test.DockerfilePaths,
			)
			test_utils.MakeParentDirsInTempDirFromFilePaths(
				t, tempDir, test.ComposefilePaths,
			)
			if len(test.DotEnvContents) != 0 {
				dotEnvFiles := make([]string, len(test.DotEnvContents))

				for i := range test.DotEnvContents {
					dotEnvFiles[i] = ".env"
				}

				_ = test_utils.WriteFilesToTempDir(
					t, tempDir, dotEnvFiles, test.DotEnvContents,
				)
			}

			_ = test_utils.WriteFilesToTempDir(
				t, tempDir, test.DockerfilePaths, test.DockerfileContents,
			)
			pathsToParse := test_utils.WriteFilesToTempDir(
				t, tempDir, test.ComposefilePaths, test.ComposefileContents,
			)

			pathsToParseCh := make(chan collect.IPath, len(pathsToParse))
			for _, path := range pathsToParse {
				pathsToParseCh <- collect.NewPath(kind.Composefile, path, nil)
			}
			close(pathsToParseCh)

			done := make(chan struct{})

			composefileParser, err := parse.NewComposefileImageParser(
				parse.NewDockerfileImageParser(),
			)
			if err != nil {
				t.Fatal(err)
			}

			composefileImages := composefileParser.ParseFiles(
				pathsToParseCh, done,
			)

			var got []parse.IImage

			for composefileImage := range composefileImages {
				if composefileImage.Err() != nil {
					close(done)
					t.Fatal(composefileImage.Err())
				}
				got = append(got, composefileImage)
			}

			for _, composefileImage := range test.Expected {
				metadata := composefileImage.Metadata()

				metadata["path"] = filepath.Join(
					tempDir, composefileImage.Metadata()["path"].(string),
				)

				if dockerfilePath, ok := metadata["dockerfilePath"]; ok {
					metadata["dockerfilePath"] = filepath.Join(
						tempDir, dockerfilePath.(string),
					)
				}

				composefileImage.SetMetadata(metadata)
			}
			test_utils.SortComposefileImageParserResults(t, got)

			test_utils.AssertImagesEqual(t, test.Expected, got)
		})
	}
}
