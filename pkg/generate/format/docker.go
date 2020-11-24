package format

import (
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

		path := image.Metadata()["path"].(string)
		position := image.Metadata()["position"].(int)

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
