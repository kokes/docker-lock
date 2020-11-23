package generate

import (
	"io"

	"github.com/safe-waters/docker-lock/pkg/generate/collect"
	"github.com/safe-waters/docker-lock/pkg/generate/parse"
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

type IImageSorter func(i int, j int) bool
