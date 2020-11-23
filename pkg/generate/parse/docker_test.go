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

const dockerfileImageParserTestDir = "dockerfileParser-tests"

func TestDockerfileImageParser(t *testing.T) {
	t.Parallel()

	tests := []struct {
		Name               string
		DockerfilePaths    []string
		DockerfileContents [][]byte
		Expected           []parse.IImage
		ShouldFail         bool
	}{
		{
			Name:            "Position",
			DockerfilePaths: []string{"Dockerfile"},
			DockerfileContents: [][]byte{
				[]byte(`
FROM ubuntu:bionic
FROM golang:1.14
FROM node
`),
			},
			Expected: []parse.IImage{
				test_utils.MakeImage(kind.Dockerfile, "ubuntu", "bionic", "", map[string]interface{}{
					"path":     "Dockerfile",
					"position": 0,
				}),
				test_utils.MakeImage(kind.Dockerfile, "golang", "1.14", "", map[string]interface{}{
					"path":     "Dockerfile",
					"position": 1,
				}),
				test_utils.MakeImage(kind.Dockerfile, "node", "latest", "", map[string]interface{}{
					"path":     "Dockerfile",
					"position": 2,
				}),
			},
		},
		{
			Name:            "Scratch",
			DockerfilePaths: []string{"Dockerfile"},
			DockerfileContents: [][]byte{
				[]byte(`
FROM scratch
`),
			},
			Expected: []parse.IImage{
				test_utils.MakeImage(kind.Dockerfile, "scratch", "", "", map[string]interface{}{
					"path":     "Dockerfile",
					"position": 0,
				}),
			},
		},
		{
			Name:            "Digest",
			DockerfilePaths: []string{"Dockerfile"},
			DockerfileContents: [][]byte{
				[]byte(`
FROM ubuntu@sha256:bae015c28bc7
`),
			},
			Expected: []parse.IImage{
				test_utils.MakeImage(kind.Dockerfile, "ubuntu", "", "bae015c28bc7", map[string]interface{}{
					"path":     "Dockerfile",
					"position": 0,
				}),
			},
		},
		{
			Name:            "Flag",
			DockerfilePaths: []string{"Dockerfile"},
			DockerfileContents: [][]byte{
				[]byte(`
FROM --platform=$BUILDPLATFORM ubuntu@sha256:bae015c28bc7
`),
			},
			Expected: []parse.IImage{
				test_utils.MakeImage(kind.Dockerfile, "ubuntu", "", "bae015c28bc7", map[string]interface{}{
					"path":     "Dockerfile",
					"position": 0,
				}),
			},
		},
		{
			Name:            "Tag And Digest",
			DockerfilePaths: []string{"Dockerfile"},
			DockerfileContents: [][]byte{
				[]byte(`
FROM ubuntu:bionic@sha256:bae015c28bc7
`),
			},
			Expected: []parse.IImage{
				test_utils.MakeImage(kind.Dockerfile, "ubuntu", "bionic", "bae015c28bc7", map[string]interface{}{
					"path":     "Dockerfile",
					"position": 0,
				}),
			},
		},
		{
			Name:            "Port, Tag, And Digest",
			DockerfilePaths: []string{"Dockerfile"},
			DockerfileContents: [][]byte{
				[]byte(`
FROM localhost:5000/ubuntu:bionic@sha256:bae015c28bc7
`),
			},
			Expected: []parse.IImage{
				test_utils.MakeImage(kind.Dockerfile, "localhost:5000/ubuntu", "bionic", "bae015c28bc7", map[string]interface{}{
					"path":     "Dockerfile",
					"position": 0,
				}),
			},
		},
		{
			Name:            "Local Arg",
			DockerfilePaths: []string{"Dockerfile"},
			DockerfileContents: [][]byte{
				[]byte(`
ARG IMAGE=busybox
FROM ${IMAGE}
ARG IMAGE=ubuntu
FROM ${IMAGE}
`),
			},
			Expected: []parse.IImage{
				test_utils.MakeImage(kind.Dockerfile, "busybox", "latest", "", map[string]interface{}{
					"path":     "Dockerfile",
					"position": 0,
				}),
				test_utils.MakeImage(kind.Dockerfile, "busybox", "latest", "", map[string]interface{}{
					"path":     "Dockerfile",
					"position": 1,
				}),
			},
		},
		{
			Name:            "Build Stage",
			DockerfilePaths: []string{"Dockerfile"},
			DockerfileContents: [][]byte{
				[]byte(`
FROM busybox AS busy
FROM busy as anotherbusy
FROM ubuntu as worker
`),
			},
			Expected: []parse.IImage{
				test_utils.MakeImage(kind.Dockerfile, "busybox", "latest", "", map[string]interface{}{
					"path":     "Dockerfile",
					"position": 0,
				}),
				test_utils.MakeImage(kind.Dockerfile, "ubuntu", "latest", "", map[string]interface{}{
					"path":     "Dockerfile",
					"position": 1,
				}),
			},
		},
		{
			Name:            "Multiple Files",
			DockerfilePaths: []string{"Dockerfile-one", "Dockerfile-two"},
			DockerfileContents: [][]byte{
				[]byte(`
FROM busybox
FROM ubuntu
`),
				[]byte(`
FROM ubuntu
FROM busybox
`),
			},
			Expected: []parse.IImage{
				test_utils.MakeImage(kind.Dockerfile, "busybox", "latest", "", map[string]interface{}{
					"path":     "Dockerfile-one",
					"position": 0,
				}),
				test_utils.MakeImage(kind.Dockerfile, "ubuntu", "latest", "", map[string]interface{}{
					"path":     "Dockerfile-one",
					"position": 1,
				}),
				test_utils.MakeImage(kind.Dockerfile, "ubuntu", "latest", "", map[string]interface{}{
					"path":     "Dockerfile-two",
					"position": 0,
				}),
				test_utils.MakeImage(kind.Dockerfile, "busybox", "latest", "", map[string]interface{}{
					"path":     "Dockerfile-two",
					"position": 1,
				}),
			},
		},
		{
			Name:            "Invalid Arg",
			DockerfilePaths: []string{"Dockerfile"},
			DockerfileContents: [][]byte{
				[]byte(`
ARG
FROM busybox
`),
			},
			ShouldFail: true,
		},
		{
			Name:            "Invalid From",
			DockerfilePaths: []string{"Dockerfile"},
			DockerfileContents: [][]byte{
				[]byte(`
FROM
`),
			},
			ShouldFail: true,
		},
	}

	for _, test := range tests { // nolint: dupl
		test := test

		t.Run(test.Name, func(t *testing.T) {
			t.Parallel()

			tempDir := test_utils.MakeTempDir(t, dockerfileImageParserTestDir)
			defer os.RemoveAll(tempDir)

			test_utils.MakeParentDirsInTempDirFromFilePaths(
				t, tempDir, test.DockerfilePaths,
			)
			pathsToParse := test_utils.WriteFilesToTempDir(
				t, tempDir, test.DockerfilePaths, test.DockerfileContents,
			)

			pathsToParseCh := make(chan collect.IPath, len(pathsToParse))
			for _, path := range pathsToParse {
				pathsToParseCh <- collect.NewPath(kind.Dockerfile, path, nil)
			}
			close(pathsToParseCh)

			done := make(chan struct{})

			dockerfileParser := parse.NewDockerfileImageParser(kind.Dockerfile)
			dockerfileImages := dockerfileParser.ParseFiles(
				pathsToParseCh, done,
			)

			var got []parse.IImage

			for dockerfileImage := range dockerfileImages {
				if test.ShouldFail {
					if dockerfileImage.Err() == nil {
						t.Fatal("expected error but did not get one")
					}

					return
				}

				if dockerfileImage.Err() != nil {
					close(done)
					t.Fatal(dockerfileImage.Err())
				}

				got = append(got, dockerfileImage)
			}

			test_utils.SortDockerfileImageParserResults(t, got)

			for _, dockerfileImage := range test.Expected {
				dockerfileImage.SetMetadata(map[string]interface{}{
					"path": filepath.Join(
						tempDir, dockerfileImage.Metadata()["path"].(string),
					),
					"position": dockerfileImage.Metadata()["position"],
				})
			}

			test_utils.AssertDockerfileImagesEqual(t, test.Expected, got)
		})
	}
}
