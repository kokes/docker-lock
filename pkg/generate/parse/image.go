package parse

import (
	"fmt"
	"sync"

	"github.com/safe-waters/docker-lock/pkg/kind"
)

type image struct {
	kind          kind.Kind
	name          string
	tag           string
	digest        string
	metadataMutex *sync.Mutex
	metadata      map[string]interface{}
	err           error
}

func NewImage(
	kind kind.Kind,
	name string,
	tag string,
	digest string,
	metadata map[string]interface{},
	err error,
) IImage {
	return &image{
		kind:          kind,
		name:          name,
		tag:           tag,
		digest:        digest,
		metadataMutex: &sync.Mutex{},
		metadata:      metadata,
		err:           err,
	}
}

func (i *image) SetKind(kind kind.Kind) {
	i.kind = kind
}

func (i *image) Kind() kind.Kind {
	return i.kind
}

func (i *image) SetName(name string) {
	i.name = name
}

func (i *image) Name() string {
	return i.name
}

func (i *image) SetTag(tag string) {
	i.tag = tag
}

func (i *image) Tag() string {
	return i.tag
}

func (i *image) SetDigest(digest string) {
	i.digest = digest
}

func (i *image) Digest() string {
	return i.digest
}

func (i *image) SetMetadata(metadata map[string]interface{}) {
	i.metadataMutex.Lock()
	defer i.metadataMutex.Unlock()

	i.metadata = i.deepCopyMetadata(metadata)
}

func (i *image) Metadata() map[string]interface{} {
	i.metadataMutex.Lock()
	defer i.metadataMutex.Unlock()

	metadata := i.deepCopyMetadata(i.metadata)

	return metadata
}

func (i *image) ImageLine() string {
	return fmt.Sprintf("%s:%s@sha256:%s", i.Name(), i.Tag(), i.Digest())
}

func (i *image) SetNameTagDigestFromImageLine(imageLine string) {
	tagSeparator := -1
	digestSeparator := -1

loop:
	for i, c := range imageLine {
		switch c {
		case ':':
			tagSeparator = i
		case '/':
			// reset tagSeparator
			// for instance, 'localhost:5000/my-image'
			tagSeparator = -1
		case '@':
			digestSeparator = i
			break loop
		}
	}

	var name, tag, digest string

	switch {
	case tagSeparator != -1 && digestSeparator != -1:
		// ubuntu:18.04@sha256:9b1702...
		name = imageLine[:tagSeparator]
		tag = imageLine[tagSeparator+1 : digestSeparator]
		digest = imageLine[digestSeparator+1+len("sha256:"):]
	case tagSeparator != -1 && digestSeparator == -1:
		// ubuntu:18.04
		name = imageLine[:tagSeparator]
		tag = imageLine[tagSeparator+1:]
	case tagSeparator == -1 && digestSeparator != -1:
		// ubuntu@sha256:9b1702...
		name = imageLine[:digestSeparator]
		digest = imageLine[digestSeparator+1+len("sha256:"):]
	default:
		// ubuntu
		name = imageLine
		if name != "scratch" {
			tag = "latest"
		}
	}

	i.SetName(name)
	i.SetTag(tag)
	i.SetDigest(digest)
}

func (i *image) SetErr(err error) {
	i.err = err
}

func (i *image) Err() error {
	return i.err
}

func (i *image) deepCopyMetadata(
	m map[string]interface{},
) map[string]interface{} {
	cp := make(map[string]interface{})

	for k, v := range m {
		vm, ok := v.(map[string]interface{})
		if ok {
			cp[k] = i.deepCopyMetadata(vm)
		} else {
			cp[k] = v
		}
	}

	return cp
}
