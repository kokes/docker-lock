package update

import "github.com/safe-waters/docker-lock/pkg/generate/parse"

// IImageDigestUpdater provides an interface for ImageDigestUpdater's
// exported methods.
type IImageDigestUpdater interface {
	UpdateDigests(
		images <-chan parse.IImage,
		done <-chan struct{},
	) <-chan parse.IImage
}
