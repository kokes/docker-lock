// Package generate provides functionality to generate a Lockfile.
package generate

import (
	"encoding/json"
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
	ImageSorter        IImageSorter
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
	imageSorter IImageSorter,
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

	if imageSorter == nil ||
		reflect.ValueOf(imageSorter).IsNil() {
		return nil, errors.New("imageSorter may not be nil")
	}

	return &Generator{
		PathCollector:      pathCollector,
		ImageParser:        imageParser,
		ImageDigestUpdater: imageDigestUpdater,
		ImageSorter:        imageSorter,
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
	images = g.ImageDigestUpdater.UpdateDigests(images, done)

	sortedImages, err := g.ImageSorter.SortImages(images, done)
	if err != nil {
		return err
	}

	lockfile := map[kind.Kind]interface{}{}

	for kind, images := range sortedImages {
		for _, image := range images {
			lockfile[kind] = image.Export()
		}
	}

	lockfileByt, err := json.MarshalIndent(lockfile, "", "\t")
	if err != nil {
		return err
	}

	_, err = writer.Write(lockfileByt)

	return err
}
