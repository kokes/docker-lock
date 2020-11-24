package sort

import (
	"sort"

	"github.com/safe-waters/docker-lock/pkg/generate/parse"
	"github.com/safe-waters/docker-lock/pkg/kind"
)

type composefileImageSorter struct {
	kind kind.Kind
}

func NewComposefileImageSorter(kind kind.Kind) IImageSorter {
	return &composefileImageSorter{kind: kind}
}

func (c *composefileImageSorter) Kind() kind.Kind {
	return c.kind
}

func (c *composefileImageSorter) SortImages(images <-chan parse.IImage) ([]parse.IImage, error) {
	var sortedImages []parse.IImage

	for image := range images {
		if image.Err() != nil {
			return nil, image.Err()
		}

		sortedImages = append(sortedImages, image)
	}

	sort.Slice(sortedImages, func(i int, j int) bool {
		switch {
		case sortedImages[i].Metadata()["path"].(string) != sortedImages[j].Metadata()["path"].(string):
			return sortedImages[i].Metadata()["path"].(string) < sortedImages[j].Metadata()["path"].(string)
		case sortedImages[i].Metadata()["serviceName"].(string) != sortedImages[j].Metadata()["serviceName"].(string):
			return sortedImages[i].Metadata()["serviceName"].(string) < sortedImages[j].Metadata()["serviceName"].(string)
		default:
			return sortedImages[i].Metadata()["position"].(int) < sortedImages[j].Metadata()["position"].(int)
		}
	})

	return sortedImages, nil
}
