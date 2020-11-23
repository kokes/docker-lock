package generate

import (
	"fmt"
	"reflect"
	"sync"

	"github.com/safe-waters/docker-lock/pkg/generate/collect"
	"github.com/safe-waters/docker-lock/pkg/kind"
)

type pathCollector struct {
	collectors map[kind.Kind]collect.IPathCollector
}

func NewPathCollector(collectors map[kind.Kind]collect.IPathCollector) (IPathCollector, error) {
	for kind, collector := range collectors {
		if collector == nil || reflect.ValueOf(collector).IsNil() {
			return nil, fmt.Errorf("%s collector is nil", kind)
		}
	}

	return &pathCollector{collectors: collectors}, nil
}

// CollectPaths collects paths to be parsed.
func (p *pathCollector) CollectPaths(done <-chan struct{}) <-chan collect.IPath {
	paths := make(chan collect.IPath)

	var waitGroup sync.WaitGroup

	waitGroup.Add(1)

	go func() {
		defer waitGroup.Done()

		for _, collector := range p.collectors {
			waitGroup.Add(1)

			go func() {
				defer waitGroup.Done()

				collectedPaths := collector.CollectPaths(done)
				for path := range collectedPaths {
					select {
					case <-done:
						return
					case paths <- path:
						if path.Err() != nil {
							return
						}
					}
				}
			}()
		}
	}()

	go func() {
		waitGroup.Wait()
		close(paths)
	}()

	return paths
}
