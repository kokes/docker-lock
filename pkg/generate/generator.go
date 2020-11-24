// Package generate provides functionality to generate a Lockfile.
package generate

import (
	"encoding/json"
	"errors"
	"io"
	"reflect"
)

// Generator creates a Lockfile.
type Generator struct {
	PathCollector      IPathCollector
	ImageParser        IImageParser
	ImageDigestUpdater IImageDigestUpdater
	ImageFormatter     IImageFormatter
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
	imageFormatter IImageFormatter,
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

	if imageFormatter == nil ||
		reflect.ValueOf(imageFormatter).IsNil() {
		return nil, errors.New("imageFormatter may not be nil")
	}

	return &Generator{
		PathCollector:      pathCollector,
		ImageParser:        imageParser,
		ImageDigestUpdater: imageDigestUpdater,
		ImageFormatter:     imageFormatter,
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

	formattedImages, err := g.ImageFormatter.FormatImages(images, done)
	if err != nil {
		close(done)
		return err
	}

	byt, err := json.MarshalIndent(formattedImages, "", "\t")
	if err != nil {
		return err
	}

	_, err = writer.Write(byt)

	return err
}
