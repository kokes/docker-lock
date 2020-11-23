package test_utils

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"

	"github.com/safe-waters/docker-lock/pkg/generate/parse"
)

func AssertDockerfileImagesEqual(
	t *testing.T,
	expected []parse.IImage,
	got []parse.IImage,
) {
	t.Helper()

	if !reflect.DeepEqual(expected, got) {
		t.Fatalf(
			"expected %+v, got %+v",
			fmt.Sprintf("%#v", expected),
			fmt.Sprintf("%#v", got),
		)
	}
}

func AssertKubernetesfileImagesEqual(
	t *testing.T,
	expected []parse.IImage,
	got []parse.IImage,
) {
	t.Helper()

	if !reflect.DeepEqual(expected, got) {
		log.Printf("%#v", got[0].Metadata())
		t.Fatalf(
			"expected %+v, got %+v",
			JsonPrettyPrint(t, expected),
			JsonPrettyPrint(t, got),
		)
	}
}

func AssertComposefileImagesEqual(
	t *testing.T,
	expected []parse.IImage,
	got []parse.IImage,
) {
	t.Helper()

	if !reflect.DeepEqual(expected, got) {
		for k, v := range got {
			log.Println(k, v)
		}
		t.Fatalf(
			"expected %+v, got %+v",
			JsonPrettyPrint(t, expected),
			JsonPrettyPrint(t, got),
		)
	}
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

func JsonPrettyPrint(t *testing.T, i interface{}) string {
	t.Helper()

	byt, err := json.MarshalIndent(i, "", "\t")
	if err != nil {
		t.Fatal(err)
	}

	return string(byt)
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
	kind string,
	name string,
	tag string,
	digest string,
	metadata map[string]interface{},
) parse.IImage {
	return parse.NewImage(kind, name, tag, digest, metadata, nil)
}