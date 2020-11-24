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

type sortedResult struct {
	kind         kind.Kind
	sortedImages []parse.IImage
	err          error
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

		for _, images := range kindImages {
			close(images)
		}
	}()

	var sortedWaitGroup sync.WaitGroup

	sortedResults := make(chan *sortedResult)

	for kind, images := range kindImages {
		kind := kind
		images := images

		sortedWaitGroup.Add(1)

		go func() {
			defer sortedWaitGroup.Done()

			sortedImages, err := i.sorters[kind].SortImages(images)
			if err != nil {
				select {
				case <-done:
				case sortedResults <- &sortedResult{err: err}:
				}

				return
			}

			select {
			case <-done:
			case sortedResults <- &sortedResult{kind: kind, sortedImages: sortedImages}:
			}
		}()
	}

	go func() {
		sortedWaitGroup.Wait()
		close(sortedResults)
	}()

	sortedKindImages := map[kind.Kind][]parse.IImage{}

	for sortedResult := range sortedResults {
		if sortedResult.err != nil {
			return nil, sortedResult.err
		}

		sortedKindImages[sortedResult.kind] = sortedResult.sortedImages
	}

	return sortedKindImages, nil
}
