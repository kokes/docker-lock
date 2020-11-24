package sort

import (
	"sort"

	"github.com/safe-waters/docker-lock/pkg/generate/parse"
	"github.com/safe-waters/docker-lock/pkg/kind"
)

type kubernetesfileImageSorter struct {
	kind kind.Kind
}

func NewKubernetesfileImageSorter(kind kind.Kind) IImageSorter {
	return &kubernetesfileImageSorter{kind: kind}
}

func (k *kubernetesfileImageSorter) Kind() kind.Kind {
	return k.kind
}

func (k *kubernetesfileImageSorter) SortImages(images <-chan parse.IImage) ([]parse.IImage, error) {
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
		case sortedImages[i].Metadata()["docPosition"].(int) != sortedImages[j].Metadata()["docPosition"].(int):
			return sortedImages[i].Metadata()["docPosition"].(int) < sortedImages[j].Metadata()["docPosition"].(int)
		default:
			return sortedImages[i].Metadata()["imagePosition"].(int) < sortedImages[j].Metadata()["imagePosition"].(int)
		}
	})

	return sortedImages, nil
}
