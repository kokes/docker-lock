package generate

import (
	"io"

	"github.com/safe-waters/docker-lock/pkg/generate/collect"
	"github.com/safe-waters/docker-lock/pkg/generate/parse"
	"github.com/safe-waters/docker-lock/pkg/kind"
)

type IPathCollector interface {
	CollectPaths(done <-chan struct{}) <-chan collect.IPath
}

type IImageParser interface {
	ParseFiles(paths <-chan collect.IPath, done <-chan struct{}) <-chan parse.IImage
}

type IImageDigestUpdater interface {
	UpdateDigests(
		images <-chan parse.IImage, done <-chan struct{},
	) <-chan parse.IImage
}

type ILockfile interface {
	Write(writer io.Writer) error
}

type IImageFormatter interface {
	FormatImages(images <-chan parse.IImage, done <-chan struct{}) (map[kind.Kind][]parse.IImage, error)
}
