package collect

import "github.com/safe-waters/docker-lock/pkg/kind"

type path struct {
	kind kind.Kind
	val  string
	err  error
}

func NewPath(kind kind.Kind, val string, err error) IPath {
	return &path{
		kind: kind,
		val:  val,
		err:  err,
	}
}

func (p *path) Kind() kind.Kind {
	return p.kind
}

func (p *path) SetKind(kind kind.Kind) {
	p.kind = kind
}

func (p *path) Path() string {
	return p.val
}

func (p *path) SetPath(val string) {
	p.val = val
}

func (p *path) Err() error {
	return p.err
}

func (p *path) SetErr(err error) {
	p.err = err
}
