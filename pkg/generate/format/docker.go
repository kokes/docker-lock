package format

import (
	"errors"
	"path/filepath"
	"sort"
	"sync"

	"github.com/safe-waters/docker-lock/pkg/generate/parse"
	"github.com/safe-waters/docker-lock/pkg/kind"
)

type dockerfileImageFormatter struct {
	kind kind.Kind
}

type formattedDockerfileImage struct {
	Name     string `json:"name"`
	Tag      string `json:"tag"`
	Digest   string `json:"digest"`
	position int
}

func NewDockerfileImageFormatter(kind kind.Kind) IImageFormatter {
	return &dockerfileImageFormatter{kind: kind}
}

func (d *dockerfileImageFormatter) Kind() kind.Kind {
	return d.kind
}

func (d *dockerfileImageFormatter) FormatImages(images <-chan parse.IImage) (map[string][]interface{}, error) {
	formattedImages := map[string][]interface{}{}

	for image := range images {
		if image.Err() != nil {
			return nil, image.Err()
		}

		path, ok := image.Metadata()["path"].(string)
		if !ok {
			return nil, errors.New("missing path in dockerfile image")
		}

		path = filepath.ToSlash(path)

		position, ok := image.Metadata()["position"].(int)
		if !ok {
			return nil, errors.New("missing position in dockerfile image")
		}

		formattedImage := &formattedDockerfileImage{
			Name:     image.Name(),
			Tag:      image.Tag(),
			Digest:   image.Digest(),
			position: position,
		}

		formattedImages[path] = append(formattedImages[path], formattedImage)
	}

	var waitGroup sync.WaitGroup

	for _, images := range formattedImages {
		images := images

		waitGroup.Add(1)

		go func() {
			defer waitGroup.Done()

			sort.Slice(images, func(i int, j int) bool {
				image1 := images[i].(*formattedDockerfileImage)
				image2 := images[j].(*formattedDockerfileImage)

				return image1.position < image2.position
			})
		}()
	}

	waitGroup.Wait()

	return formattedImages, nil

}
