package parse_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/safe-waters/docker-lock/pkg/generate/collect"
	"github.com/safe-waters/docker-lock/pkg/generate/parse"
	"github.com/safe-waters/docker-lock/pkg/kind"
	"github.com/safe-waters/docker-lock/pkg/testutils"
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
				parse.NewImage(
					kind.Dockerfile, "ubuntu", "bionic", "",
					map[string]interface{}{
						"path":     "Dockerfile",
						"position": 0,
					}, nil,
				),
				parse.NewImage(
					kind.Dockerfile, "golang", "1.14", "",
					map[string]interface{}{
						"path":     "Dockerfile",
						"position": 1,
					}, nil,
				),
				parse.NewImage(
					kind.Dockerfile, "node", "latest", "",
					map[string]interface{}{
						"path":     "Dockerfile",
						"position": 2,
					}, nil,
				),
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
				parse.NewImage(
					kind.Dockerfile, "scratch", "", "",
					map[string]interface{}{
						"path":     "Dockerfile",
						"position": 0,
					}, nil,
				),
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
				parse.NewImage(
					kind.Dockerfile, "ubuntu", "", "bae015c28bc7",
					map[string]interface{}{
						"path":     "Dockerfile",
						"position": 0,
					}, nil,
				),
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
				parse.NewImage(
					kind.Dockerfile, "ubuntu", "", "bae015c28bc7",
					map[string]interface{}{
						"path":     "Dockerfile",
						"position": 0,
					}, nil,
				),
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
				parse.NewImage(
					kind.Dockerfile, "ubuntu", "bionic", "bae015c28bc7",
					map[string]interface{}{
						"path":     "Dockerfile",
						"position": 0,
					}, nil,
				),
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
				parse.NewImage(
					kind.Dockerfile, "localhost:5000/ubuntu", "bionic",
					"bae015c28bc7", map[string]interface{}{
						"path":     "Dockerfile",
						"position": 0,
					}, nil,
				),
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
				parse.NewImage(
					kind.Dockerfile, "busybox", "latest", "",
					map[string]interface{}{
						"path":     "Dockerfile",
						"position": 0,
					}, nil,
				),
				parse.NewImage(
					kind.Dockerfile, "busybox", "latest", "",
					map[string]interface{}{
						"path":     "Dockerfile",
						"position": 1,
					}, nil,
				),
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
				parse.NewImage(
					kind.Dockerfile, "busybox", "latest", "",
					map[string]interface{}{
						"path":     "Dockerfile",
						"position": 0,
					}, nil,
				),
				parse.NewImage(
					kind.Dockerfile, "ubuntu", "latest", "",
					map[string]interface{}{
						"path":     "Dockerfile",
						"position": 1,
					}, nil,
				),
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
				parse.NewImage(
					kind.Dockerfile, "busybox", "latest", "",
					map[string]interface{}{
						"path":     "Dockerfile-one",
						"position": 0,
					}, nil,
				),
				parse.NewImage(
					kind.Dockerfile, "ubuntu", "latest", "",
					map[string]interface{}{
						"path":     "Dockerfile-one",
						"position": 1,
					}, nil,
				),
				parse.NewImage(
					kind.Dockerfile, "ubuntu", "latest", "",
					map[string]interface{}{
						"path":     "Dockerfile-two",
						"position": 0,
					}, nil,
				),
				parse.NewImage(
					kind.Dockerfile, "busybox", "latest", "",
					map[string]interface{}{
						"path":     "Dockerfile-two",
						"position": 1,
					}, nil,
				),
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

			tempDir := testutils.MakeTempDir(t, dockerfileImageParserTestDir)
			defer os.RemoveAll(tempDir)

			testutils.MakeParentDirsInTempDirFromFilePaths(
				t, tempDir, test.DockerfilePaths,
			)
			pathsToParse := testutils.WriteFilesToTempDir(
				t, tempDir, test.DockerfilePaths, test.DockerfileContents,
			)

			pathsToParseCh := make(chan collect.IPath, len(pathsToParse))
			for _, path := range pathsToParse {
				pathsToParseCh <- collect.NewPath(kind.Dockerfile, path, nil)
			}
			close(pathsToParseCh)

			done := make(chan struct{})

			dockerfileParser := parse.NewDockerfileImageParser()
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

			testutils.SortDockerfileImageParserResults(t, got)

			for _, dockerfileImage := range test.Expected {
				dockerfileImage.SetMetadata(map[string]interface{}{
					"path": filepath.Join(
						tempDir, dockerfileImage.Metadata()["path"].(string),
					),
					"position": dockerfileImage.Metadata()["position"],
				})
			}

			testutils.AssertImagesEqual(t, test.Expected, got)
		})
	}
}
