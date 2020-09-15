package parse_test

import (
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/safe-waters/docker-lock/generate/collect"
	"github.com/safe-waters/docker-lock/generate/parse"
)

const dockerfileImageParserTestDir = "dockerfileParser-tests"

func TestDockerfileImageParser(t *testing.T) {
	t.Parallel()

	tests := []struct {
		Name               string
		DockerfilePaths    []string
		DockerfileContents [][]byte
		Expected           []*parse.DockerfileImage
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
			Expected: []*parse.DockerfileImage{
				{
					Image:    &parse.Image{Name: "ubuntu", Tag: "bionic"},
					Position: 0,
					Path:     "Dockerfile",
				},
				{
					Image:    &parse.Image{Name: "golang", Tag: "1.14"},
					Position: 1,
					Path:     "Dockerfile",
				},
				{
					Image:    &parse.Image{Name: "node", Tag: "latest"},
					Position: 2,
					Path:     "Dockerfile",
				},
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
			Expected: []*parse.DockerfileImage{
				{
					Image:    &parse.Image{Name: "busybox", Tag: "latest"},
					Position: 0,
					Path:     "Dockerfile",
				},
				{
					Image:    &parse.Image{Name: "busybox", Tag: "latest"},
					Position: 1,
					Path:     "Dockerfile",
				},
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
			Expected: []*parse.DockerfileImage{
				{
					Image:    &parse.Image{Name: "busybox", Tag: "latest"},
					Position: 0,
					Path:     "Dockerfile",
				},
				{
					Image:    &parse.Image{Name: "ubuntu", Tag: "latest"},
					Position: 1,
					Path:     "Dockerfile",
				},
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
			Expected: []*parse.DockerfileImage{
				{
					Image:    &parse.Image{Name: "busybox", Tag: "latest"},
					Position: 0,
					Path:     "Dockerfile-one",
				},
				{
					Image:    &parse.Image{Name: "ubuntu", Tag: "latest"},
					Position: 1,
					Path:     "Dockerfile-one",
				},

				{
					Image:    &parse.Image{Name: "ubuntu", Tag: "latest"},
					Position: 0,
					Path:     "Dockerfile-two",
				},
				{
					Image:    &parse.Image{Name: "busybox", Tag: "latest"},
					Position: 1,
					Path:     "Dockerfile-two",
				},
			},
		},
	}

	for _, test := range tests {
		test := test

		t.Run(test.Name, func(t *testing.T) {
			t.Parallel()

			tempDir := makeTempDir(t, dockerfileImageParserTestDir)
			defer os.RemoveAll(tempDir)

			makeParentDirsInTempDirFromFilePaths(
				t, tempDir, test.DockerfilePaths,
			)
			pathsToParse := writeFilesToTempDir(
				t, tempDir, test.DockerfilePaths, test.DockerfileContents,
			)

			pathsToParseCh := make(chan *collect.PathResult, len(pathsToParse))
			for _, path := range pathsToParse {
				pathsToParseCh <- &collect.PathResult{Path: path}
			}
			close(pathsToParseCh)

			done := make(chan struct{})

			dockerfileParser := &parse.DockerfileImageParser{}
			dockerfileImages := dockerfileParser.ParseFiles(
				pathsToParseCh, done,
			)

			var got []*parse.DockerfileImage

			for dockerfileImage := range dockerfileImages {
				if dockerfileImage.Err != nil {
					close(done)
					t.Fatal(dockerfileImage.Err)
				}
				got = append(got, dockerfileImage)
			}

			sortDockerfileImageParserResults(t, got)

			for _, dockerfileImage := range test.Expected {
				dockerfileImage.Path = filepath.Join(
					tempDir, dockerfileImage.Path,
				)
			}

			assertDockerfileImagesEqual(t, test.Expected, got)
		})
	}
}

func sortDockerfileImageParserResults(
	t *testing.T,
	results []*parse.DockerfileImage,
) {
	t.Helper()

	sort.Slice(results, func(i, j int) bool {
		switch {
		case results[i].Path != results[j].Path:
			return results[i].Path < results[j].Path
		default:
			return results[i].Position < results[j].Position
		}
	})
}