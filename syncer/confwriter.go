package syncer

import (
	"archive/tar"
	"context"
	"fmt"
	"github.com/sirupsen/logrus"
	"os"
	"path/filepath"
)

type FileConfWriter struct {
	ID   string
	Dest string
}

func NewFileConfWriter(id, dest string) FileConfWriter {
	return FileConfWriter{
		ID:   id,
		Dest: dest,
	}
}

func (w FileConfWriter) Write(ctx context.Context, outputConfigs []OutputConfig) error {
	logrus.Infof("received %d items", len(outputConfigs))
	for _, outputConfig := range outputConfigs {
		compressedConfigName := generateCompressedConfigName(w.ID, outputConfig)
		fName, err := filepath.Abs(filepath.Join(w.Dest, compressedConfigName))
		if err != nil {
			return err
		}
		logrus.Infof("confWiter creating a file: %s", fName)

		f, err := os.Create(fName)
		if err != nil {
			return err
		}

		tr := tar.NewWriter(f)

		content := []byte(outputConfig.Content)
		hdr := &tar.Header{
			Name: outputConfig.Name,
			Mode: 0644,
			Size: int64(len(content)),
		}
		if err := tr.WriteHeader(hdr); err != nil {
			return err
		}

		_, err = tr.Write(content)
		if err != nil {
			return err
		}
		tr.Close()
		f.Close()
	}

	return nil
}

func generateCompressedConfigName(id string, outputConfig OutputConfig) string {
	fileExt := "gz"
	fileName := outputConfig.Name

	return fmt.Sprintf("id-%s.%s.%s", id, fileNameWithoutExtension(fileName), fileExt)
}

func fileNameWithoutExtension(fileName string) string {
	return fileName[:len(fileName)-len(filepath.Ext(fileName))]
}
