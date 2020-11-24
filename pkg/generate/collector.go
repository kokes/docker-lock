package generate

import (
	"errors"
	"sync"

	"github.com/safe-waters/docker-lock/pkg/generate/collect"
)

type pathCollector struct {
	collectors []collect.IPathCollector
}

func NewPathCollector(collectors ...collect.IPathCollector) (IPathCollector, error) {
	if len(collectors) == 0 {
		return nil, errors.New("collectors must be greater than 0")
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
			collector := collector

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
