package collect

// IPathCollector provides an interface for PathCollector's exported methods.
type IPathCollector interface {
	CollectPaths(done <-chan struct{}) <-chan IPath
}

type IPath interface {
	Path() string
	SetPath(path string)
	Err() error
	SetErr(err error)
}
