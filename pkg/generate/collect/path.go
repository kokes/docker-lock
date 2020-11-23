package collect

type path struct {
	val string
	err error
}

func NewPath(val string, err error) IPath {
	return &path{
		val: val,
		err: err,
	}
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
