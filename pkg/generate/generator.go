// Package generate provides functionality to generate a Lockfile.
package generate

import (
	"errors"
	"io"
	"reflect"

	"github.com/safe-waters/docker-lock/pkg/kind"
)

// Generator creates a Lockfile.
type Generator struct {
	PathCollector      IPathCollector
	ImageParser        IImageParser
	ImageDigestUpdater IImageDigestUpdater
	ImageSorters       map[kind.Kind]IImageSorter
}

// IGenerator provides an interface for Generator's exported
// methods, which are used by docker-lock's cli as well as Verifier.
type IGenerator interface {
	GenerateLockfile(writer io.Writer) error
}

// NewGenerator returns a Generator after validating its fields.
func NewGenerator(
	pathCollector IPathCollector,
	imageParser IImageParser,
	imageDigestUpdater IImageDigestUpdater,
	imageSorters map[kind.Kind]IImageSorter,
) (*Generator, error) {
	if pathCollector == nil || reflect.ValueOf(pathCollector).IsNil() {
		return nil, errors.New("pathCollector may not be nil")
	}

	if imageParser == nil || reflect.ValueOf(imageParser).IsNil() {
		return nil, errors.New("imageParser may not be nil")
	}

	if imageDigestUpdater == nil ||
		reflect.ValueOf(imageDigestUpdater).IsNil() {
		return nil, errors.New("imageDigestUpdater may not be nil")
	}

	if len(imageSorters) == 0 {
		return nil, errors.New("imageSorters may not be nil")
	}

	return &Generator{
		PathCollector:      pathCollector,
		ImageParser:        imageParser,
		ImageDigestUpdater: imageDigestUpdater,
		ImageSorters:       imageSorters,
	}, nil
}

// GenerateLockfile creates a Lockfile and writes it to an io.Writer.
func (g *Generator) GenerateLockfile(writer io.Writer) error {
	if writer == nil || reflect.ValueOf(writer).IsNil() {
		return errors.New("writer cannot be nil")
	}

	done := make(chan struct{})

	paths := g.PathCollector.CollectPaths(done)
	images := g.ImageParser.ParseFiles(paths, done)
	imagesWithDigests := g.ImageDigestUpdater.UpdateDigests(images, done)

	lockfile, err := NewLockfile(imagesWithDigests, g.ImageSorters)
	if err != nil {
		close(done)
		return err
	}

	return lockfile.Write(writer)
}
