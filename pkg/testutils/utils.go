package testutils

import (
	"bytes"
	"crypto/rand"
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

	"github.com/safe-waters/docker-lock/pkg/generate/collect"
	"github.com/safe-waters/docker-lock/pkg/generate/parse"
)

const (
	BusyboxLatestSHA = "bae015c28bc7cdee3b7ef20d35db4299e3068554a769070950229d9f53f58572" // nolint: lll
	GolangLatestSHA  = "6cb55c08bbf44793f16e3572bd7d2ae18f7a858f6ae4faa474c0a6eae1174a5d" // nolint: lll
	RedisLatestSHA   = "09c33840ec47815dc0351f1eca3befe741d7105b3e95bc8fdb9a7e4985b9e1e5" // nolint: lll
)

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
			t.Fatalf(
				"expected kind %#v, got %#v", expected[i].Kind(), got[i].Kind(),
			)
		}

		if expected[i].Name() != got[i].Name() {
			t.Fatalf(
				"expected name %#v, got %#v", expected[i].Name(), got[i].Name(),
			)
		}

		if expected[i].Tag() != got[i].Tag() {
			t.Fatalf(
				"expected digest %#v, got %#v", expected[i].Tag(), got[i].Tag(),
			)
		}

		if expected[i].Digest() != got[i].Digest() {
			t.Fatalf(
				"expected digest %#v, got %#v", expected[i].Digest(),
				got[i].Digest(),
			)
		}

		expectedMetadataByt, err := json.MarshalIndent(
			expected[i].Metadata(), "", "\t",
		)
		if err != nil {
			t.Fatal(err)
		}

		gotMetadataByt, err := json.MarshalIndent(got[i].Metadata(), "", "\t")
		if err != nil {
			t.Fatal(err)
		}

		if !bytes.Equal(expectedMetadataByt, gotMetadataByt) {
			t.Fatalf(
				"expected metadata %#v, got %#v",
				string(expectedMetadataByt), string(gotMetadataByt),
			)
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
				case "redis:latest":
					digest = RedisLatestSHA
				case "golang:latest":
					digest = GolangLatestSHA
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
		var (
			path1     = results[i].Metadata()["path"].(string)
			path2     = results[j].Metadata()["path"].(string)
			position1 = results[i].Metadata()["position"].(int)
			position2 = results[j].Metadata()["position"].(int)
		)

		switch {
		case path1 != path2:
			return path1 < path2
		default:
			return position1 < position2
		}
	})
}

func SortKubernetesfileImageParserResults(
	t *testing.T,
	results []parse.IImage,
) {
	t.Helper()

	sort.Slice(results, func(i, j int) bool {
		var (
			path1          = results[i].Metadata()["path"].(string)
			path2          = results[j].Metadata()["path"].(string)
			docPosition1   = results[i].Metadata()["docPosition"].(int)
			docPosition2   = results[j].Metadata()["docPosition"].(int)
			imagePosition1 = results[i].Metadata()["imagePosition"].(int)
			imagePosition2 = results[j].Metadata()["imagePosition"].(int)
		)

		switch {
		case path1 != path2:
			return path1 < path2
		case docPosition1 != docPosition2:
			return docPosition1 < docPosition2
		default:
			return imagePosition1 < imagePosition2
		}
	})
}

func SortComposefileImageParserResults(
	t *testing.T,
	results []parse.IImage,
) {
	t.Helper()

	sort.Slice(results, func(i, j int) bool {
		var (
			path1            = results[i].Metadata()["path"].(string)
			path2            = results[j].Metadata()["path"].(string)
			serviceName1     = results[i].Metadata()["serviceName"].(string)
			serviceName2     = results[j].Metadata()["serviceName"].(string)
			servicePosition1 = results[i].Metadata()["servicePosition"].(int)
			servicePosition2 = results[j].Metadata()["servicePosition"].(int)
		)

		switch {
		case path1 != path2:
			return path1 < path2
		case serviceName1 != serviceName2:
			return serviceName1 < serviceName2
		default:
			return servicePosition1 < servicePosition2
		}
	})
}

func SortPaths(t *testing.T, paths []collect.IPath) {
	t.Helper()

	sort.Slice(paths, func(i, j int) bool {
		switch {
		case paths[i].Kind() != paths[j].Kind():
			return paths[i].Kind() < paths[j].Kind()
		default:
			return paths[i].Path() < paths[j].Path()
		}
	})
}

func MakeTempDirInCurrentDir(t *testing.T) string {
	t.Helper()

	tempDir := generateUUID(t)
	MakeDir(t, tempDir)

	return tempDir
}

func generateUUID(t *testing.T) string {
	t.Helper()

	b := make([]byte, 16)

	_, err := rand.Read(b)
	if err != nil {
		t.Fatal(err)
	}

	uuid := fmt.Sprintf("%x-%x-%x-%x-%x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:],
	)

	return uuid
}
