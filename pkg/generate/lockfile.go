package generate

import (
	"encoding/json"
	"errors"
	"io"
	"reflect"
	"sort"
	"sync"

	"github.com/safe-waters/docker-lock/pkg/generate/parse"
	"github.com/safe-waters/docker-lock/pkg/kind"
)

type lockfile struct {
	images map[kind.Kind][]parse.IImage
}

// TODO: windows paths
func NewLockfile(images <-chan parse.IImage, sorters map[kind.Kind]IImageSorter) (ILockfile, error) {
	kindImages := map[kind.Kind][]parse.IImage{}

	for image := range images {
		if image.Err() != nil {
			return nil, image.Err()
		}

		kindImages[image.Kind()] = append(kindImages[image.Kind()])
	}

	var waitGroup sync.WaitGroup

	for kind, images := range kindImages {
		kind := kind
		images := images

		waitGroup.Add(1)

		go func() {
			defer waitGroup.Done()

			sort.Slice(images, sorters[kind])
		}()
	}

	waitGroup.Wait()

	return &lockfile{images: kindImages}, nil

}

// Write writes the Lockfile in JSON format to an io.Writer.
func (l *lockfile) Write(writer io.Writer) error {
	if writer == nil || reflect.ValueOf(writer).IsNil() {
		return errors.New("writer cannot be nil")
	}

	lockfileByt, err := json.MarshalIndent(l, "", "\t")
	if err != nil {
		return err
	}

	if _, err := writer.Write(lockfileByt); err != nil {
		return err
	}

	return nil
}
