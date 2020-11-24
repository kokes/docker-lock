package generate

import (
	"encoding/json"
	"io"

	"github.com/safe-waters/docker-lock/pkg/generate/parse"
	"github.com/safe-waters/docker-lock/pkg/kind"
)

type lockfile struct {
	images map[kind.Kind][]interface{}
}

func NewLockfile(sortedImages map[kind.Kind][]parse.IImage) ILockfile {
	exportedImages := map[kind.Kind][]interface{}{}

	// TODO: go routine
	for kind, images := range sortedImages {
		for _, image := range images {
			exportedImages[kind] = append(
				exportedImages[kind], image.Export(),
			)
		}
	}

	return &lockfile{
		images: exportedImages,
	}
}

func (l *lockfile) Write(writer io.Writer) error {
	byt, err := json.MarshalIndent(l.images, "", "\t")
	if err != nil {
		return err
	}

	_, err = writer.Write(byt)

	return err
}
