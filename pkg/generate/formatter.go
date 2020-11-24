package generate

import (
	"errors"
	"fmt"
	"reflect"
	"sync"

	"github.com/safe-waters/docker-lock/pkg/generate/format"
	"github.com/safe-waters/docker-lock/pkg/generate/parse"
	"github.com/safe-waters/docker-lock/pkg/kind"
)

type imageFormatter struct {
	formatters map[kind.Kind]format.IImageFormatter
}

type formattedResult struct {
	kind            kind.Kind
	formattedImages map[string][]interface{}
	err             error
}

func NewImageFormatter(formatters ...format.IImageFormatter) (IImageFormatter, error) {
	kindFormatter := map[kind.Kind]format.IImageFormatter{}

	for _, formatter := range formatters {
		if formatter != nil && !reflect.ValueOf(formatter).IsNil() {
			kindFormatter[formatter.Kind()] = formatter
		}
	}

	if len(kindFormatter) == 0 {
		return nil, errors.New("non nil formatter must be greater than 0")
	}

	return &imageFormatter{formatters: kindFormatter}, nil
}

func (i *imageFormatter) FormatImages(images <-chan parse.IImage, done <-chan struct{}) (map[kind.Kind]map[string][]interface{}, error) {
	kindImages := map[kind.Kind]chan parse.IImage{}

	for kind := range i.formatters {
		kindImages[kind] = make(chan parse.IImage)
	}

	var kindImagesWaitGroup sync.WaitGroup

	for image := range images {
		if image.Err() != nil {
			return nil, image.Err()
		}

		if _, ok := i.formatters[image.Kind()]; !ok {
			return nil, fmt.Errorf("kind %s does not have a formatter defined", image.Kind())
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

	var formattedWaitGroup sync.WaitGroup

	formattedResults := make(chan *formattedResult)

	for kind, images := range kindImages {
		kind := kind
		images := images

		formattedWaitGroup.Add(1)

		go func() {
			defer formattedWaitGroup.Done()

			formattedImages, err := i.formatters[kind].FormatImages(images)
			if err != nil {
				select {
				case <-done:
				case formattedResults <- &formattedResult{err: err}:
				}

				return
			}

			select {
			case <-done:
			case formattedResults <- &formattedResult{kind: kind, formattedImages: formattedImages}:
			}
		}()
	}

	go func() {
		formattedWaitGroup.Wait()
		close(formattedResults)
	}()

	formattedKindImages := map[kind.Kind]map[string][]interface{}{}

	for formattedResult := range formattedResults {
		if formattedResult.err != nil {
			return nil, formattedResult.err
		}

		formattedKindImages[formattedResult.kind] = formattedResult.formattedImages
	}

	return formattedKindImages, nil
}
