package generate

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/joho/godotenv"
	"github.com/safe-waters/docker-lock/pkg/generate"
	"github.com/safe-waters/docker-lock/pkg/generate/collect"
	"github.com/safe-waters/docker-lock/pkg/generate/parse"
	"github.com/safe-waters/docker-lock/pkg/generate/registry"
	"github.com/safe-waters/docker-lock/pkg/generate/registry/contrib"
	"github.com/safe-waters/docker-lock/pkg/generate/registry/firstparty"
	"github.com/safe-waters/docker-lock/pkg/generate/sort"
	"github.com/safe-waters/docker-lock/pkg/generate/update"
	"github.com/safe-waters/docker-lock/pkg/kind"
)

// DefaultPathCollector creates a PathCollector for Generator.
func DefaultPathCollector(flags *Flags) (generate.IPathCollector, error) {
	if err := ensureFlagsNotNil(flags); err != nil {
		return nil, err
	}

	var dockerfileCollector collect.IPathCollector

	var composefileCollector collect.IPathCollector

	var kubernetesfileCollector collect.IPathCollector

	var err error

	if !flags.DockerfileFlags.ExcludePaths {
		dockerfileCollector, err = collect.NewPathCollector(
			kind.Dockerfile,
			flags.FlagsWithSharedValues.BaseDir, []string{"Dockerfile"},
			flags.DockerfileFlags.ManualPaths, flags.DockerfileFlags.Globs,
			flags.DockerfileFlags.Recursive,
		)
		if err != nil {
			return nil, err
		}
	}

	if !flags.ComposefileFlags.ExcludePaths {
		composefileCollector, err = collect.NewPathCollector(
			kind.Composefile,
			flags.FlagsWithSharedValues.BaseDir,
			[]string{"docker-compose.yml", "docker-compose.yaml"},
			flags.ComposefileFlags.ManualPaths, flags.ComposefileFlags.Globs,
			flags.ComposefileFlags.Recursive,
		)
		if err != nil {
			return nil, err
		}
	}

	if !flags.KubernetesfileFlags.ExcludePaths {
		kubernetesfileCollector, err = collect.NewPathCollector(
			kind.Kubernetesfile,
			flags.FlagsWithSharedValues.BaseDir,
			[]string{
				"deployment.yml", "deployment.yaml",
				"pod.yml", "pod.yaml",
				"job.yml", "job.yaml",
			},
			flags.KubernetesfileFlags.ManualPaths,
			flags.KubernetesfileFlags.Globs,
			flags.KubernetesfileFlags.Recursive,
		)
		if err != nil {
			return nil, err
		}
	}

	return generate.NewPathCollector(dockerfileCollector, composefileCollector, kubernetesfileCollector)
}

// DefaultImageParser creates an ImageParser for Generator.
func DefaultImageParser(flags *Flags) (generate.IImageParser, error) {
	if err := ensureFlagsNotNil(flags); err != nil {
		return nil, err
	}

	var dockerfileImageParser parse.IDockerfileImageParser

	var composefileImageParser parse.IComposefileImageParser

	var kubernetesfileImageParser parse.IKubernetesfileImageParser

	if !flags.DockerfileFlags.ExcludePaths ||
		!flags.ComposefileFlags.ExcludePaths {
		dockerfileImageParser = parse.NewDockerfileImageParser(kind.Dockerfile)
	}

	if !flags.ComposefileFlags.ExcludePaths {
		var err error

		composefileImageParser, err = parse.NewComposefileImageParser(
			kind.Composefile,
			dockerfileImageParser,
		)

		if err != nil {
			return nil, err
		}
	}

	if !flags.KubernetesfileFlags.ExcludePaths {
		kubernetesfileImageParser = parse.NewKubernetesfileImageParser(kind.Kubernetesfile)
	}

	return generate.NewImageParser(dockerfileImageParser, composefileImageParser, kubernetesfileImageParser)
}

func DefaultImageSorter() (generate.IImageSorter, error) {
	dockerfileImageSorter := sort.NewDockerfileImageSorter(kind.Dockerfile)
	composefileImageSorter := sort.NewComposefileImageSorter(kind.Composefile)
	kubernetesfileImageSorter := sort.NewKubernetesfileImageSorter(kind.Kubernetesfile)

	return generate.NewImageSorter(dockerfileImageSorter, composefileImageSorter, kubernetesfileImageSorter)
}

// DefaultImageDigestUpdater creates an ImageDigestUpdater for Generator.
func DefaultImageDigestUpdater(
	client *registry.HTTPClient,
	flags *Flags,
) (generate.IImageDigestUpdater, error) {
	if err := ensureFlagsNotNil(flags); err != nil {
		return nil, err
	}

	wrapperManager, err := DefaultWrapperManager(
		client, flags.FlagsWithSharedValues.ConfigPath,
	)
	if err != nil {
		return nil, err
	}

	imageDigestUpdater, err := update.NewImageDigestUpdater(wrapperManager)
	if err != nil {
		return nil, err
	}

	return generate.NewImageDigestUpdater(
		imageDigestUpdater, flags.FlagsWithSharedValues.IgnoreMissingDigests,
	)
}

// DefaultConfigPath returns the default location of docker's config.json
// for all platforms.
func DefaultConfigPath() string {
	if homeDir, err := os.UserHomeDir(); err == nil {
		configPath := filepath.Join(homeDir, ".docker", "config.json")
		if _, err := os.Stat(configPath); err != nil {
			return ""
		}

		return configPath
	}

	return ""
}

// DefaultWrapperManager creates a WrapperManager with all possible Wrappers,
// the default being the docker wrapper.
func DefaultWrapperManager(
	client *registry.HTTPClient,
	configPath string,
) (*registry.WrapperManager, error) {
	defaultWrapper, err := firstparty.DefaultWrapper(client, configPath)
	if err != nil {
		return nil, err
	}

	wrapperManager := registry.NewWrapperManager(defaultWrapper)
	wrapperManager.Add(firstparty.AllWrappers(client, configPath)...)
	wrapperManager.Add(contrib.AllWrappers(client, configPath)...)

	return wrapperManager, nil
}

// DefaultLoadEnv loads .env files based on the path. If a path does not
// exist and that path is not ".env", an error will occur.
func DefaultLoadEnv(path string) error {
	if _, err := os.Stat(path); err != nil {
		if path == ".env" {
			return nil
		}

		return err
	}

	return godotenv.Load(path)
}

func ensureFlagsNotNil(flags *Flags) error {
	if flags == nil {
		return errors.New("flags cannot be nil")
	}

	if flags.DockerfileFlags == nil {
		return errors.New("flags.DockerfileFlags cannot be nil")
	}

	if flags.ComposefileFlags == nil {
		return errors.New("flags.ComposefileFlags cannot be nil")
	}

	if flags.KubernetesfileFlags == nil {
		return errors.New("flags.KubernetesfileFlags cannot be nil")
	}

	if flags.FlagsWithSharedValues == nil {
		return errors.New("flags.FlagsWithSharedValues cannot be nil")
	}

	return nil
}
