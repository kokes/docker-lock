package format

import (
	"github.com/safe-waters/docker-lock/pkg/generate/parse"
	"github.com/safe-waters/docker-lock/pkg/kind"
)

type IImageFormatter interface {
	Kind() kind.Kind
	FormatImages(images <-chan parse.IImage) (map[string][]interface{}, error)
}
