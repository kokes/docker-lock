// Package update provides functionality to update images with digests.
package update

import (
	"errors"
	"sync"

	"github.com/safe-waters/docker-lock/pkg/generate/parse"
	"github.com/safe-waters/docker-lock/pkg/generate/registry"
)

type imageDigestUpdater struct {
	wrapperManager *registry.WrapperManager
}

// NewImageDigestUpdater returns an ImageDigestUpdater after validating its
// fields.
func NewImageDigestUpdater(
	wrapperManager *registry.WrapperManager,
) (*imageDigestUpdater, error) {
	if wrapperManager == nil {
		return nil, errors.New("wrapperManager cannot be nil")
	}

	return &imageDigestUpdater{wrapperManager: wrapperManager}, nil
}

// UpdateDigests queries registries for digests of images that do not
// already specify their digests. It updates images with those
// digests.
func (i *imageDigestUpdater) UpdateDigests(
	images <-chan parse.IImage,
	done <-chan struct{},
) <-chan parse.IImage {
	if images == nil {
		return nil
	}

	updatedImages := make(chan parse.IImage)

	var waitGroup sync.WaitGroup

	waitGroup.Add(1)

	go func() {
		defer waitGroup.Done()

		for image := range images {
			image := image

			waitGroup.Add(1)

			go func() {
				defer waitGroup.Done()

				if image.Err() != nil || image.Digest() != "" {
					select {
					case <-done:
					case updatedImages <- image:
					}

					return
				}

				wrapper := i.wrapperManager.Wrapper(image.Name())

				digest, err := wrapper.Digest(image.Name(), image.Tag())
				if err != nil {
					select {
					case <-done:
					case updatedImages <- parse.NewImage(image.Kind(), "", "", "", nil, err):
					}

					return
				}

				select {
				case <-done:
					return
				case updatedImages <- parse.NewImage(
					image.Kind(), image.Name(), image.Tag(),
					digest, image.Metadata(), nil,
				):
				}
			}()
		}
	}()

	go func() {
		waitGroup.Wait()
		close(updatedImages)
	}()

	return updatedImages
}
