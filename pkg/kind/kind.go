package kind

type Kind string

const (
	Dockerfile     = "docker"
	Composefile    = "docker-compose"
	Kubernetesfile = "kubernetes"
)
