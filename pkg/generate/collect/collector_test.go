package collect_test

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/safe-waters/docker-lock/pkg/generate/collect"
	"github.com/safe-waters/docker-lock/pkg/kind"
	"github.com/safe-waters/docker-lock/pkg/test_utils"
)

func TestPathCollector(t *testing.T) {
	t.Parallel()

	tests := []struct {
		Name          string
		DefaultPaths  []string
		ManualPaths   []string
		Globs         []string
		Recursive     bool
		ShouldFail    bool
		Expected      []string
		PathsToCreate []string
	}{
		{
			Name:          "Default Path Exists",
			DefaultPaths:  []string{"Dockerfile"},
			PathsToCreate: []string{"Dockerfile"},
			Expected:      []string{"Dockerfile"},
		},
		{
			Name:         "Default Path Does Not Exist",
			DefaultPaths: []string{"Dockerfile"},
		},
		{
			Name:          "Do Not Use Default Paths If Other Methods Specified",
			DefaultPaths:  []string{"Dockerfile"},
			ManualPaths:   []string{"Dockerfile-Manual"},
			Expected:      []string{"Dockerfile-Manual"},
			PathsToCreate: []string{"Dockerfile", "Dockerfile-Manual"},
		},
		{
			Name:          "Manual Paths",
			ManualPaths:   []string{"Dockerfile-Manual"},
			Expected:      []string{"Dockerfile-Manual"},
			PathsToCreate: []string{"Dockerfile-Manual"},
		},
		{
			Name:          "Globs",
			Globs:         []string{"Dockerfile-*"},
			Expected:      []string{"Dockerfile-glob"},
			PathsToCreate: []string{"Dockerfile-glob"},
		},
		{
			Name:          "Duplicate Paths",
			ManualPaths:   []string{"Dockerfile-Manual", "Dockerfile-Manual"},
			Expected:      []string{"Dockerfile-Manual"},
			PathsToCreate: []string{"Dockerfile-Manual"},
		},
		{
			Name:         "Default Path Outside Of Base Directory",
			DefaultPaths: []string{filepath.Join("..", "..", "Dockerfile")},
			ShouldFail:   true,
		},
		{
			Name:        "Manual Path Outside Of Base Directory",
			ManualPaths: []string{filepath.Join("..", "..", "Dockerfile")},
			ShouldFail:  true,
		},
	}

	for _, test := range tests {
		test := test

		t.Run(test.Name, func(t *testing.T) {
			t.Parallel()

			tempDir := test_utils.MakeTempDirInCurrentDir(t)
			defer os.RemoveAll(tempDir)

			var expected []string

			if len(test.PathsToCreate) != 0 {
				pathsToCreateContents := make([][]byte, len(test.PathsToCreate))
				test_utils.WriteFilesToTempDir(
					t, tempDir, test.PathsToCreate, pathsToCreateContents,
				)

				for _, path := range test.Expected {
					expected = append(
						expected, filepath.Join(tempDir, path),
					)
				}
			}

			collector, err := collect.NewPathCollector(kind.Dockerfile, tempDir, test.DefaultPaths, test.ManualPaths, test.Globs, test.Recursive)
			if err != nil {
				t.Fatal(err)
			}

			var got []string

			done := make(chan struct{})
			for path := range collector.CollectPaths(done) {
				if path.Err() != nil {
					close(done)
					err = path.Err()
					break
				}
				got = append(got, path.Path())
			}

			if test.ShouldFail {
				if err == nil {
					t.Fatal("expected an error but did not get one")
				}

				return
			}

			if err != nil {
				t.Fatal(err)
			}

			if !reflect.DeepEqual(expected, got) {
				t.Fatalf("expected %v, got %v", expected, got)
			}
		})
	}
}
