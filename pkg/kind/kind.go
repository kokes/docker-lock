package kind

type Kind string

const (
	Dockerfile     = "Dockerfile"
	Composefile    = "Composefile"
	Kubernetesfile = "Kubernetesfile"
)

func AllKinds() []Kind {
	return []Kind{Dockerfile, Composefile, Kubernetesfile}
}
