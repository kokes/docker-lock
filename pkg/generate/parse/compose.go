package parse

import (
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/docker/cli/cli/compose/loader"
	"github.com/docker/cli/cli/compose/types"
	"github.com/docker/cli/opts"
)

type composefileImageParser struct {
	dockerfileImageParser IDockerfileImageParser
}

// NewComposefileImageParser returns a composefileImageParser after validating
// its fields.
func NewComposefileImageParser(
	dockerfileImageParser IDockerfileImageParser,
) (IComposefileImageParser, error) {
	if dockerfileImageParser == nil {
		return nil, errors.New("dockerfileImageParser cannot be nil")
	}

	return &composefileImageParser{
		dockerfileImageParser: dockerfileImageParser,
	}, nil
}

// ParseFiles reads docker-compose YAML to parse all images.
func (c *composefileImageParser) ParseFiles(
	paths <-chan string,
	done <-chan struct{},
) <-chan IImage {
	if paths == nil {
		return nil
	}

	composefileImages := make(chan IImage)

	var waitGroup sync.WaitGroup

	waitGroup.Add(1)

	go func() {
		defer waitGroup.Done()

		for path := range paths {
			waitGroup.Add(1)

			go c.ParseFile(
				path, composefileImages, done, &waitGroup,
			)
		}
	}()

	go func() {
		waitGroup.Wait()
		close(composefileImages)
	}()

	return composefileImages
}

func (c *composefileImageParser) ParseFile(
	path string,
	composefileImages chan<- IImage,
	done <-chan struct{},
	waitGroup *sync.WaitGroup,
) {
	defer waitGroup.Done()

	byt, err := ioutil.ReadFile(path)
	if err != nil {
		select {
		case <-done:
		case composefileImages <- NewImage("Composefile", "", "", "", nil, err):
		}

		return
	}

	composefileData, err := loader.ParseYAML(byt)
	if err != nil {
		select {
		case <-done:
		case composefileImages <- NewImage("Composefile", "", "", "", nil, err):
		}

		return
	}

	envVars := map[string]string{}

	for _, envVarStr := range os.Environ() {
		envVarVal := strings.SplitN(envVarStr, "=", 2)
		envVars[envVarVal[0]] = envVarVal[1]
	}

	var envFileVars []string

	if envFileVars, err = opts.ParseEnvFile(
		filepath.Join(filepath.Dir(path), ".env"),
	); err == nil {
		for _, envVarStr := range envFileVars {
			envVarVal := strings.SplitN(envVarStr, "=", 2)
			if _, ok := envVars[envVarVal[0]]; !ok {
				envVars[envVarVal[0]] = envVarVal[1]
			}
		}
	}

	loadedComposefile, err := loader.Load(
		types.ConfigDetails{
			ConfigFiles: []types.ConfigFile{
				{
					Config:   composefileData,
					Filename: path,
				},
			},
			// replaces env vars with $ in file
			Environment: envVars,
		},
	)
	if err != nil {
		select {
		case <-done:
		case composefileImages <- NewImage("Composefile", "", "", "", nil, err):
		}

		return
	}

	for _, serviceConfig := range loadedComposefile.Services {
		waitGroup.Add(1)

		go c.parseService(
			serviceConfig, path, envVars, composefileImages, waitGroup, done,
		)
	}
}

func (c *composefileImageParser) parseService(
	serviceConfig types.ServiceConfig,
	path string,
	envVars map[string]string,
	composefileImages chan<- IImage,
	waitGroup *sync.WaitGroup,
	done <-chan struct{},
) {
	defer waitGroup.Done()

	if serviceConfig.Build.Context == "" {
		image := NewImage("Composefile", "", "", "", map[string]interface{}{
			"serviceName": serviceConfig.Name,
			"position":    0,
			"path":        path,
		}, nil)

		image.SetNameTagDigestFromImageLine(serviceConfig.Image)

		select {
		case <-done:
		case composefileImages <- image:
		}

		return
	}

	dockerfileImages := make(chan IImage)

	var dockerfileImageWaitGroup sync.WaitGroup

	dockerfileImageWaitGroup.Add(1)

	go func() {
		defer dockerfileImageWaitGroup.Done()

		context := serviceConfig.Build.Context
		if !filepath.IsAbs(context) {
			context = filepath.Join(filepath.Dir(path), context)
		}

		dockerfile := serviceConfig.Build.Dockerfile
		if dockerfile == "" {
			dockerfile = "Dockerfile"
		}

		dockerfilePath := filepath.Join(context, dockerfile)

		buildArgs := map[string]string{}

		for arg, val := range serviceConfig.Build.Args {
			if val == nil {
				// For the case:
				//	args:
				//	  - MYENVVAR
				// where MYENVVAR does not have $ in front
				buildArgs[arg] = envVars[arg]
			} else {
				buildArgs[arg] = *val
			}
		}

		dockerfileImageWaitGroup.Add(1)

		go c.dockerfileImageParser.ParseFile(
			dockerfilePath, buildArgs, dockerfileImages,
			done, &dockerfileImageWaitGroup,
		)
	}()

	go func() {
		dockerfileImageWaitGroup.Wait()
		close(dockerfileImages)
	}()

	for dockerfileImage := range dockerfileImages {
		dockerfileImage.SetKind("Composefile")

		if dockerfileImage.Err() != nil {
			select {
			case <-done:
			case composefileImages <- dockerfileImage:
			}

			return
		}

		dockerfileImage.SetMetadata(map[string]interface{}{
			"dockerfilePath": dockerfileImage.Metadata()["path"],
			"position":       dockerfileImage.Metadata()["position"],
			"serviceName":    serviceConfig.Name,
			"path":           path,
		})

		select {
		case <-done:
			return
		case composefileImages <- dockerfileImage:
		}
	}
}
