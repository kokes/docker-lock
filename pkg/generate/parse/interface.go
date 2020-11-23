package parse

import "sync"

type IImage interface {
	SetKind(kind string)
	Kind() string
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
	ParseFiles(
		paths <-chan string,
		done <-chan struct{},
	) <-chan IImage
}

type IDockerfileImageParser interface {
	IImageParser
	ParseFile(
		path string,
		buildArgs map[string]string,
		dockerfileImages chan<- IImage,
		done <-chan struct{},
		waitGroup *sync.WaitGroup,
	)
}

type IComposefileImageParser interface {
	IImageParser
	ParseFile(
		path string,
		composefileImages chan<- IImage,
		done <-chan struct{},
		waitGroup *sync.WaitGroup,
	)
}

type IKubernetesfileImageParser interface {
	IImageParser
	ParseFile(
		path string,
		kubernetesfileImages chan<- IImage,
		done <-chan struct{},
		waitGroup *sync.WaitGroup,
	)
}
