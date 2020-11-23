package collect

import "github.com/safe-waters/docker-lock/pkg/kind"

// IPathCollector provides an interface for PathCollector's exported methods.
type IPathCollector interface {
	Kind() kind.Kind
	CollectPaths(done <-chan struct{}) <-chan IPath
}

type IPath interface {
	Kind() kind.Kind
	SetKind(kind kind.Kind)
	Path() string
	SetPath(path string)
	Err() error
	SetErr(err error)
}
