package generate

import (
	"errors"
	"sync"

	"github.com/safe-waters/docker-lock/pkg/generate/parse"
	"github.com/safe-waters/docker-lock/pkg/generate/sort"
	"github.com/safe-waters/docker-lock/pkg/kind"
)

type imageSorter struct {
	sorters map[kind.Kind]sort.IImageSorter
}

func NewImageSorter(sorters ...sort.IImageSorter) (IImageSorter, error) {
	if len(sorters) == 0 {
		return nil, errors.New("sorters must be greater than 0")
	}

	kindSorter := map[kind.Kind]sort.IImageSorter{}

	for _, sorter := range sorters {
		kindSorter[sorter.Kind()] = sorter
	}

	return &imageSorter{sorters: kindSorter}, nil
}

func (i *imageSorter) SortImages(images <-chan parse.IImage, done <-chan struct{}) (map[kind.Kind][]parse.IImage, error) {
	sortedKindImages := map[kind.Kind][]parse.IImage{}

	errCh := make(chan error)

	kindImages := map[kind.Kind]chan parse.IImage{}

	for _, kind := range kind.AllKinds() {
		kindImages[kind] = make(chan parse.IImage)
	}

	var kindImagesWaitGroup sync.WaitGroup

	for image := range images {
		if image.Err() != nil {
			return nil, image.Err()
		}

		image := image

		kindImagesWaitGroup.Add(1)

		go func() {
			defer kindImagesWaitGroup.Done()

			select {
			case <-done:
				return
			case kindImages[image.Kind()] <- image:
			}
		}()
	}

	go func() {
		kindImagesWaitGroup.Wait()

		for _, paths := range kindImages {
			close(paths)
		}
	}()

	var sortedWaitGroup sync.WaitGroup

	for kind, images := range kindImages {
		kind := kind
		images := images

		sortedKindImages[kind] = []parse.IImage{}

		sortedWaitGroup.Add(1)

		go func() {
			defer sortedWaitGroup.Done()

			sortedImages, err := i.sorters[kind].SortImages(images)
			if err != nil {
				select {
				case <-done:
				case errCh <- err:
				}

				return
			}

			sortedKindImages[kind] = sortedImages
		}()
	}

	go func() {
		sortedWaitGroup.Wait()
		close(errCh)
	}()

	for err := range errCh {
		if err != nil {
			return nil, err
		}
	}

	return sortedKindImages, nil
}
