package format

import (
	"sort"
	"sync"

	"github.com/safe-waters/docker-lock/pkg/generate/parse"
	"github.com/safe-waters/docker-lock/pkg/kind"
)

type kubernetesfileImageFormatter struct {
	kind kind.Kind
}

type formattedKubernetesfileImage struct {
	Name          string `json:"name"`
	Tag           string `json:"tag"`
	Digest        string `json:"digest"`
	ContainerName string `json:"container"`
	imagePosition int
	docPosition   int
}

func NewKubernetesfileImageFormatter(kind kind.Kind) IImageFormatter {
	return &kubernetesfileImageFormatter{kind: kind}
}

func (k *kubernetesfileImageFormatter) Kind() kind.Kind {
	return k.kind
}

func (k *kubernetesfileImageFormatter) FormatImages(images <-chan parse.IImage) (map[string][]interface{}, error) {
	formattedImages := map[string][]interface{}{}

	for image := range images {
		if image.Err() != nil {
			return nil, image.Err()
		}

		path := image.Metadata()["path"].(string)
		containerName := image.Metadata()["containerName"].(string)
		imagePosition := image.Metadata()["imagePosition"].(int)
		docPosition := image.Metadata()["docPosition"].(int)

		formattedImage := &formattedKubernetesfileImage{
			Name:          image.Name(),
			Tag:           image.Tag(),
			Digest:        image.Digest(),
			ContainerName: containerName,
			imagePosition: imagePosition,
			docPosition:   docPosition,
		}

		formattedImages[path] = append(formattedImages[path], formattedImage)
	}

	var waitGroup sync.WaitGroup

	for _, images := range formattedImages {
		images := images

		waitGroup.Add(1)

		go func() {
			defer waitGroup.Done()

			sort.Slice(images, func(i, j int) bool {
				image1 := images[i].(*formattedKubernetesfileImage)
				image2 := images[j].(*formattedKubernetesfileImage)

				switch {
				case image1.docPosition != image2.docPosition:
					return image1.docPosition < image2.docPosition
				default:
					return image1.imagePosition < image2.imagePosition
				}
			})
		}()
	}

	waitGroup.Wait()

	return formattedImages, nil
}
