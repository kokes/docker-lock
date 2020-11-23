package parse

import (
	"bytes"
	"io"
	"io/ioutil"
	"sync"

	"github.com/safe-waters/docker-lock/pkg/generate/collect"
	"github.com/safe-waters/docker-lock/pkg/kind"
	"gopkg.in/yaml.v2"
	"k8s.io/client-go/kubernetes/scheme"
)

// kubernetesfileImageParser extracts image values from Kubernetesfiles.
type kubernetesfileImageParser struct {
	kind kind.Kind
}

func NewKubernetesfileImageParser(kind kind.Kind) IKubernetesfileImageParser {
	return &kubernetesfileImageParser{
		kind: kind,
	}
}

func (k *kubernetesfileImageParser) Kind() kind.Kind {
	return k.kind
}

// ParseFiles reads Kubernetesfiles to parse all images.
func (k *kubernetesfileImageParser) ParseFiles(
	paths <-chan collect.IPath,
	done <-chan struct{},
) <-chan IImage {
	if paths == nil {
		return nil
	}

	kubernetesfileImages := make(chan IImage)

	var waitGroup sync.WaitGroup

	waitGroup.Add(1)

	go func() {
		defer waitGroup.Done()

		for path := range paths {
			waitGroup.Add(1)

			go k.ParseFile(
				path, kubernetesfileImages, done, &waitGroup,
			)
		}
	}()

	go func() {
		waitGroup.Wait()
		close(kubernetesfileImages)
	}()

	return kubernetesfileImages
}

func (k *kubernetesfileImageParser) ParseFile(
	path collect.IPath,
	kubernetesfileImages chan<- IImage,
	done <-chan struct{},
	waitGroup *sync.WaitGroup,
) {
	defer waitGroup.Done()

	if path.Err() != nil {
		select {
		case <-done:
		case kubernetesfileImages <- NewImage(k.kind, "", "", "", nil, path.Err()):
		}

		return
	}

	byt, err := ioutil.ReadFile(path.Path())
	if err != nil {
		select {
		case <-done:
		case kubernetesfileImages <- NewImage(k.kind, "", "", "", nil, err):
		}

		return
	}

	_, _, err = scheme.Codecs.UniversalDeserializer().Decode(byt, nil, nil)
	if err != nil {
		select {
		case <-done:
		case kubernetesfileImages <- NewImage(k.kind, "", "", "", nil, err):
		}

		return
	}

	dec := yaml.NewDecoder(bytes.NewReader(byt))

	for docPosition := 0; ; docPosition++ {
		var doc yaml.MapSlice

		if err := dec.Decode(&doc); err != nil {
			if err != io.EOF {
				select {
				case <-done:
				case kubernetesfileImages <- NewImage(k.kind, "", "", "", nil, err):
				}

				return
			}

			break
		}

		waitGroup.Add(1)

		go k.parseDoc(
			path.Path(), doc, kubernetesfileImages, docPosition, done, waitGroup,
		)
	}
}

func (k *kubernetesfileImageParser) parseDoc(
	path string,
	doc interface{},
	kubernetesfileImages chan<- IImage,
	docPosition int,
	done <-chan struct{},
	waitGroup *sync.WaitGroup,
) {
	defer waitGroup.Done()

	var imagePosition int

	k.parseDocRecursive(
		path, doc, kubernetesfileImages, docPosition, &imagePosition, done,
	)
}

func (k *kubernetesfileImageParser) parseDocRecursive(
	path string,
	doc interface{},
	kubernetesfileImages chan<- IImage,
	docPosition int,
	imagePosition *int,
	done <-chan struct{},
) {
	switch doc := doc.(type) {
	case yaml.MapSlice:
		var name string

		var imageLine string

		for _, item := range doc {
			key, _ := item.Key.(string)
			val, _ := item.Value.(string)

			switch key {
			case "name":
				name = val
			case "image":
				imageLine = val
			}
		}

		if name != "" && imageLine != "" {
			image := NewImage(k.kind, "", "", "", map[string]interface{}{
				"containerName": name,
				"path":          path,
				"imagePosition": *imagePosition,
				"docPosition":   docPosition,
			}, nil)
			image.SetNameTagDigestFromImageLine(imageLine)

			select {
			case <-done:
			case kubernetesfileImages <- image:
			}

			*imagePosition++
		}

		for _, item := range doc {
			k.parseDocRecursive(
				path, item.Value, kubernetesfileImages,
				docPosition, imagePosition, done,
			)
		}
	case []interface{}:
		for _, doc := range doc {
			k.parseDocRecursive(
				path, doc, kubernetesfileImages,
				docPosition, imagePosition, done,
			)
		}
	}
}
