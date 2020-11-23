package parse

import (
	"sync"

	"github.com/safe-waters/docker-lock/pkg/generate/collect"
	"github.com/safe-waters/docker-lock/pkg/kind"
)

type IImage interface {
	SetKind(kind kind.Kind)
	Kind() kind.Kind
	SetName(name string)
	Name() string
	SetTag(tag string)
	Tag() string
	SetDigest(digest string)
	Digest() string
	SetMetadata(metadata map[string]interface{})
	Metadata() map[string]interface{}
	ImageLine() string
	SetNameTagDigestFromImageLine(imageLine string)
	SetErr(err error)
	Err() error
}

type IImageParser interface {
	Kind() kind.Kind
	ParseFiles(
		paths <-chan collect.IPath,
		done <-chan struct{},
	) <-chan IImage
}

type IDockerfileImageParser interface {
	IImageParser
	ParseFile(
		path collect.IPath,
		buildArgs map[string]string,
		dockerfileImages chan<- IImage,
		done <-chan struct{},
		waitGroup *sync.WaitGroup,
	)
}

type IComposefileImageParser interface {
	IImageParser
	ParseFile(
		path collect.IPath,
		composefileImages chan<- IImage,
		done <-chan struct{},
		waitGroup *sync.WaitGroup,
	)
}

type IKubernetesfileImageParser interface {
	IImageParser
	ParseFile(
		path collect.IPath,
		kubernetesfileImages chan<- IImage,
		done <-chan struct{},
		waitGroup *sync.WaitGroup,
	)
}
