package sort

import (
	"sort"

	"github.com/safe-waters/docker-lock/pkg/generate/parse"
	"github.com/safe-waters/docker-lock/pkg/kind"
)

type dockerfileImageSorter struct {
	kind kind.Kind
}

func NewDockerfileImageSorter(kind kind.Kind) IImageSorter {
	return &dockerfileImageSorter{kind: kind}
}

func (d *dockerfileImageSorter) Kind() kind.Kind {
	return d.kind
}

func (d *dockerfileImageSorter) SortImages(images <-chan parse.IImage) ([]parse.IImage, error) {
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
		default:
			return sortedImages[i].Metadata()["position"].(int) < sortedImages[j].Metadata()["position"].(int)
		}
	})

	return sortedImages, nil
}
