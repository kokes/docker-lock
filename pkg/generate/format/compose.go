package format

import (
	"sort"

	"github.com/safe-waters/docker-lock/pkg/generate/parse"
	"github.com/safe-waters/docker-lock/pkg/kind"
)

type composefileImageFormatter struct {
	kind kind.Kind
}

func NewComposefileImageFormatter(kind kind.Kind) IImageFormatter {
	return &composefileImageFormatter{kind: kind}
}

func (c *composefileImageFormatter) Kind() kind.Kind {
	return c.kind
}

func (c *composefileImageFormatter) FormatImages(images <-chan parse.IImage) ([]parse.IImage, error) {
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
		case formattedImages[i].Metadata()["serviceName"].(string) != formattedImages[j].Metadata()["serviceName"].(string):
			return formattedImages[i].Metadata()["serviceName"].(string) < formattedImages[j].Metadata()["serviceName"].(string)
		default:
			return formattedImages[i].Metadata()["position"].(int) < formattedImages[j].Metadata()["position"].(int)
		}
	})

	return formattedImages, nil
}
