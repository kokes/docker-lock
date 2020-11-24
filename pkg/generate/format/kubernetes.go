package format

import (
	"sort"

	"github.com/safe-waters/docker-lock/pkg/generate/parse"
	"github.com/safe-waters/docker-lock/pkg/kind"
)

type kubernetesfileImageFormatter struct {
	kind kind.Kind
}

func NewKubernetesfileImageFormatter(kind kind.Kind) IImageFormatter {
	return &kubernetesfileImageFormatter{kind: kind}
}

func (k *kubernetesfileImageFormatter) Kind() kind.Kind {
	return k.kind
}

func (k *kubernetesfileImageFormatter) FormatImages(images <-chan parse.IImage) ([]parse.IImage, error) {
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
		case formattedImages[i].Metadata()["docPosition"].(int) != formattedImages[j].Metadata()["docPosition"].(int):
			return formattedImages[i].Metadata()["docPosition"].(int) < formattedImages[j].Metadata()["docPosition"].(int)
		default:
			return formattedImages[i].Metadata()["imagePosition"].(int) < formattedImages[j].Metadata()["imagePosition"].(int)
		}
	})

	return formattedImages, nil
}
