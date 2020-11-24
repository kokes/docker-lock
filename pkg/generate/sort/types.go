package sort

import (
	"github.com/safe-waters/docker-lock/pkg/generate/parse"
	"github.com/safe-waters/docker-lock/pkg/kind"
)

type IImageSorter interface {
	Kind() kind.Kind
	SortImages(images <-chan parse.IImage) ([]parse.IImage, error)
}
