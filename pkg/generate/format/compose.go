package format

import (
	"sort"
	"sync"

	"github.com/safe-waters/docker-lock/pkg/generate/parse"
	"github.com/safe-waters/docker-lock/pkg/kind"
)

type composefileImageFormatter struct {
	kind kind.Kind
}

// ComposefileImage annotates an image with data about the docker-compose file
// and/or the Dockerfile from which it was parsed.
type formattedComposefileImage struct {
	Name            string `json:"name"`
	Tag             string `json:"tag"`
	Digest          string `json:"digest"`
	DockerfilePath  string `json:"dockerfile,omitempty"`
	ServiceName     string `json:"service"`
	servicePosition int
}

func NewComposefileImageFormatter(kind kind.Kind) IImageFormatter {
	return &composefileImageFormatter{kind: kind}
}

func (c *composefileImageFormatter) Kind() kind.Kind {
	return c.kind
}

func (c *composefileImageFormatter) FormatImages(images <-chan parse.IImage) (map[string][]interface{}, error) {
	formattedImages := map[string][]interface{}{}

	for image := range images {
		if image.Err() != nil {
			return nil, image.Err()
		}

		path := image.Metadata()["path"].(string)
		dockerfilePath, _ := image.Metadata()["dockerfilePath"].(string)
		serviceName := image.Metadata()["serviceName"].(string)
		servicePosition := image.Metadata()["servicePosition"].(int)

		formattedImage := &formattedComposefileImage{
			Name:            image.Name(),
			Tag:             image.Tag(),
			Digest:          image.Digest(),
			DockerfilePath:  dockerfilePath,
			ServiceName:     serviceName,
			servicePosition: servicePosition,
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
				image1 := images[i].(*formattedComposefileImage)
				image2 := images[j].(*formattedComposefileImage)

				switch {
				case image1.ServiceName != image2.ServiceName:
					return image1.ServiceName < image2.ServiceName
				case image1.DockerfilePath != image2.DockerfilePath:
					return image1.DockerfilePath < image2.DockerfilePath
				default:
					return image1.servicePosition < image2.servicePosition
				}
			})
		}()
	}

	waitGroup.Wait()

	return formattedImages, nil
}
