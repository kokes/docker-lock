package format

import (
	"sort"

	"github.com/safe-waters/docker-lock/pkg/generate/parse"
	"github.com/safe-waters/docker-lock/pkg/kind"
)

type dockerfileImageFormatter struct {
	kind kind.Kind
}

func NewDockerfileImageFormatter(kind kind.Kind) IImageFormatter {
	return &dockerfileImageFormatter{kind: kind}
}

func (d *dockerfileImageFormatter) Kind() kind.Kind {
	return d.kind
}

func (d *dockerfileImageFormatter) FormatImages(images <-chan parse.IImage) ([]parse.IImage, error) {
	var formattedImages []parse.IImage

	for image := range images {
		if image.Err() != nil {
			return nil, image.Err()
		}

		formattedImages = append(formattedImages, image)
	}

	sort.Slice(formattedImages, func(i int, j int) bool {
		switch {
		case formattedImages[i].Metadata()["path"].(string) != formattedImages[j].Metadata()["path"].(string):
			return formattedImages[i].Metadata()["path"].(string) < formattedImages[j].Metadata()["path"].(string)
		default:
			return formattedImages[i].Metadata()["position"].(int) < formattedImages[j].Metadata()["position"].(int)
		}
	})

	return formattedImages, nil
}
