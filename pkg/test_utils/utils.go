package test_utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/safe-waters/docker-lock/pkg/generate/parse"
	"github.com/safe-waters/docker-lock/pkg/kind"
)

const BusyboxLatestSHA = "bae015c28bc7cdee3b7ef20d35db4299e3068554a769070950229d9f53f58572" // nolint: lll

func AssertImagesEqual(
	t *testing.T,
	expected []parse.IImage,
	got []parse.IImage,
) {
	t.Helper()

	if len(expected) != len(got) {
		t.Fatalf("expected %d images, got %d", len(expected), len(got))
	}

	var i int

	for i < len(expected) {
		if expected[i].Kind() != got[i].Kind() {
			t.Fatalf("expected kind %#v, got %#v", expected[i].Kind(), got[i].Kind())
		}
		if expected[i].Name() != got[i].Name() {
			t.Fatalf("expected name %#v, got %#v", expected[i].Name(), got[i].Name())
		}
		if expected[i].Tag() != got[i].Tag() {
			t.Fatalf("expected digest %#v, got %#v", expected[i].Tag(), got[i].Tag())
		}
		if expected[i].Digest() != got[i].Digest() {
			t.Fatalf("expected digest %#v, got %#v", expected[i].Digest(), got[i].Digest())
		}

		expectedMetadataByt, err := json.MarshalIndent(expected[i].Metadata(), "", "\t")
		if err != nil {
			t.Fatal(err)
		}

		gotMetadataByt, err := json.MarshalIndent(got[i].Metadata(), "", "\t")

		if !bytes.Equal(expectedMetadataByt, gotMetadataByt) {
			t.Fatalf("expected metadata %#v, got %#v", string(expectedMetadataByt), string(gotMetadataByt))
		}

		i++
	}
}

func AssertNumNetworkCallsEqual(t *testing.T, expected uint64, got uint64) {
	t.Helper()

	if expected != got {
		t.Fatalf("expected %d network calls, got %d", expected, got)
	}
}

func MockServer(t *testing.T, numNetworkCalls *uint64) *httptest.Server {
	t.Helper()

	server := httptest.NewServer(
		http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
			switch url := req.URL.String(); {
			case strings.Contains(url, "scope"):
				byt := []byte(`{"token": "NOT_USED"}`)
				_, err := res.Write(byt)
				if err != nil {
					t.Fatal(err)
				}
			case strings.Contains(url, "manifests"):
				atomic.AddUint64(numNetworkCalls, 1)

				urlParts := strings.Split(url, "/")
				repo, ref := urlParts[2], urlParts[len(urlParts)-1]

				var digest string
				switch fmt.Sprintf("%s:%s", repo, ref) {
				case "busybox:latest":
					digest = BusyboxLatestSHA
				default:
					digest = fmt.Sprintf(
						"repo %s with ref %s not defined for testing",
						repo, ref,
					)
				}

				res.Header().Set("Docker-Content-Digest", digest)
			}
		}))

	return server
}

func WriteFilesToTempDir(
	t *testing.T,
	tempDir string,
	fileNames []string,
	fileContents [][]byte,
) []string {
	t.Helper()

	if len(fileNames) != len(fileContents) {
		t.Fatalf(
			"different number of names and contents: %d names, %d contents",
			len(fileNames), len(fileContents))
	}

	fullPaths := make([]string, len(fileNames))

	for i, name := range fileNames {
		fullPath := filepath.Join(tempDir, name)

		if err := ioutil.WriteFile(
			fullPath, fileContents[i], 0777,
		); err != nil {
			t.Fatal(err)
		}

		fullPaths[i] = fullPath
	}

	return fullPaths
}

func MakeDir(t *testing.T, dirPath string) {
	t.Helper()

	err := os.MkdirAll(dirPath, 0777)
	if err != nil {
		t.Fatal(err)
	}
}

func MakeTempDir(t *testing.T, dirName string) string {
	t.Helper()

	dir, err := ioutil.TempDir("", dirName)
	if err != nil {
		t.Fatal(err)
	}

	return dir
}

func MakeParentDirsInTempDirFromFilePaths(
	t *testing.T,
	tempDir string,
	paths []string,
) {
	t.Helper()

	for _, p := range paths {
		dir, _ := filepath.Split(p)
		fullDir := filepath.Join(tempDir, dir)

		MakeDir(t, fullDir)
	}
}

func SortDockerfileImageParserResults(
	t *testing.T,
	results []parse.IImage,
) {
	t.Helper()

	sort.Slice(results, func(i, j int) bool {
		switch {
		case results[i].Metadata()["path"].(string) != results[j].Metadata()["path"].(string):
			return results[i].Metadata()["path"].(string) < results[j].Metadata()["path"].(string)
		default:
			return results[i].Metadata()["position"].(int) < results[j].Metadata()["position"].(int)
		}
	})
}

func SortKubernetesfileImageParserResults(
	t *testing.T,
	results []parse.IImage,
) {
	t.Helper()

	sort.Slice(results, func(i, j int) bool {
		switch {
		case results[i].Metadata()["path"].(string) != results[j].Metadata()["path"].(string):
			return results[i].Metadata()["path"].(string) < results[j].Metadata()["path"].(string)
		case results[i].Metadata()["docPosition"].(int) != results[j].Metadata()["docPosition"].(int):
			return results[i].Metadata()["docPosition"].(int) < results[j].Metadata()["docPosition"].(int)
		default:
			return results[i].Metadata()["imagePosition"].(int) < results[j].Metadata()["imagePosition"].(int)
		}
	})
}

func SortComposefileImageParserResults(
	t *testing.T,
	results []parse.IImage,
) {
	t.Helper()

	sort.Slice(results, func(i, j int) bool {
		switch {
		case results[i].Metadata()["path"].(string) != results[j].Metadata()["path"].(string):
			return results[i].Metadata()["path"].(string) < results[j].Metadata()["path"].(string)
		case results[i].Metadata()["serviceName"].(string) != results[j].Metadata()["serviceName"].(string):
			return results[i].Metadata()["serviceName"].(string) < results[j].Metadata()["serviceName"].(string)
		case results[i].Metadata()["dockerfilePath"].(string) != results[j].Metadata()["dockerfilePath"].(string):
			return results[i].Metadata()["dockerfilePath"].(string) < results[j].Metadata()["dockerfilePath"].(string)
		default:
			return results[i].Metadata()["position"].(int) < results[j].Metadata()["position"].(int)
		}
	})
}

func MakeImage(
	kind kind.Kind,
	name string,
	tag string,
	digest string,
	metadata map[string]interface{},
) parse.IImage {
	return parse.NewImage(kind, name, tag, digest, metadata, nil)
}
