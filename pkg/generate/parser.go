package generate

import (
	"fmt"
	"reflect"
	"sync"

	"github.com/safe-waters/docker-lock/pkg/generate/collect"
	"github.com/safe-waters/docker-lock/pkg/generate/parse"
	"github.com/safe-waters/docker-lock/pkg/kind"
)

type imageParser struct {
	parsers map[kind.Kind]parse.IImageParser
}

func NewImageParser(parsers map[kind.Kind]parse.IImageParser) (IImageParser, error) {
	for kind, parser := range parsers {
		if parser == nil || reflect.ValueOf(parser).IsNil() {
			return nil, fmt.Errorf("%s parser is nil", kind)
		}
	}

	return &imageParser{parsers: parsers}, nil
}

// ParseFiles parses all files for Images.
func (i *imageParser) ParseFiles(
	paths <-chan collect.IPath,
	done <-chan struct{},
) <-chan parse.IImage {
	images := make(chan parse.IImage)

	var waitGroup sync.WaitGroup

	waitGroup.Add(1)

	go func() {
		defer waitGroup.Done()

		kindPaths := map[kind.Kind]chan collect.IPath{}

		for _, kind := range kind.AllKinds() {
			kindPaths[kind] = make(chan collect.IPath)
		}

		var kindPathsWaitGroup sync.WaitGroup

		for path := range paths {
			path := path

			kindPathsWaitGroup.Add(1)

			go func() {
				defer kindPathsWaitGroup.Done()

				select {
				case <-done:
				case kindPaths[path.Kind()] <- path:
				}
			}()
		}

		go func() {
			kindPathsWaitGroup.Wait()

			for _, paths := range kindPaths {
				close(paths)
			}
		}()

		for kind, paths := range kindPaths {
			kind := kind
			paths := paths

			waitGroup.Add(1)

			go func() {
				defer waitGroup.Done()

				for image := range i.parsers[kind].ParseFiles(paths, done) {
					select {
					case <-done:
						return
					case images <- image:
						if image.Err() != nil {
							return
						}
					}
				}
			}()
		}
	}()

	go func() {
		waitGroup.Wait()
		close(images)
	}()

	return images
}
