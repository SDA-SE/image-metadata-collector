package fs

import (
	"io"
	"os"
)

type fsParameters struct {
	baseDir string
}

func NewFs(baseDir string) (*fsParameters, error) {

	return &fsParameters{
		baseDir: baseDir,
	}, nil
}

func (s fsParameters) Upload(content []byte, fileName string, environmentName string) error {
	var writer io.Writer

	if s.baseDir == "" {
		writer = os.Stdout
	} else {
		file, _ := os.Create(s.baseDir + fileName)
		defer file.Close()
		writer = file
	}

	_, err := writer.Write(content)

	return err
}
