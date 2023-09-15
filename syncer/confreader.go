package syncer

import (
	"context"
	"github.com/semirm-dev/iopipe/errwrapper"
	"github.com/sirupsen/logrus"
	"os"
	"path"
)

type FileConfReader struct {
	Src string
}

func NewFileConfReader(src string) FileConfReader {
	return FileConfReader{
		Src: src,
	}
}

type fileRead struct {
	name    string
	content []byte
	err     error
}

// Read all files from a given input source. It can be either directory or single file.
// Each file from a directory is read concurrently.
func (r FileConfReader) Read(ctx context.Context) ([]OutputConfig, error) {
	info, err := os.Stat(r.Src)
	if err != nil {
		return nil, err
	}

	if info.IsDir() {
		return readDirectory(ctx, r.Src)
	}

	return readSingleFile(info, r.Src)
}

func readDirectory(ctx context.Context, path string) ([]OutputConfig, error) {
	filesToRead, err := scanForFiles(path)
	if err != nil {
		return nil, err
	}

	logrus.Infof("files to read: %v", filesToRead)

	readResponse := make(chan fileRead)
	go readFiles(ctx, path, filesToRead, readResponse)

	result := make([]OutputConfig, 0)
	var errWrapped error

	for i := 0; i < len(filesToRead); i++ {
		select {
		case resp := <-readResponse:
			if resp.err != nil {
				errWrapped = errwrapper.Wrap(errWrapped, resp.err)
				break
			}

			result = append(result, OutputConfig{
				Name:    resp.name,
				Content: string(resp.content),
			})
		}
	}

	return result, errWrapped
}

func readSingleFile(info os.FileInfo, path string) ([]OutputConfig, error) {
	logrus.Infof("file to read: %v", info.Name())

	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	return []OutputConfig{
		{
			Name:    info.Name(),
			Content: string(content),
		},
	}, nil
}

func readFiles(ctx context.Context, dirPath string, files []string, readResponse chan fileRead) {
	for _, file := range files {
		select {
		case <-ctx.Done():
			return
		default:
			go func(ctx context.Context, filename string) {
				select {
				case <-ctx.Done():
					return
				default:
					content, err := os.ReadFile(path.Join(dirPath, filename))
					if err != nil {
						readResponse <- fileRead{err: err}
						return
					}

					readResponse <- fileRead{
						name:    filename,
						content: content,
					}
				}
			}(ctx, file)
		}
	}
}

func scanForFiles(dirPath string) ([]string, error) {
	files, err := os.ReadDir(dirPath)
	if err != nil {
		return nil, err
	}

	return filterFilesOnly(files), nil
}

func filterFilesOnly(files []os.DirEntry) []string {
	filesToRead := make([]string, 0)

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		filesToRead = append(filesToRead, file.Name())
	}

	return filesToRead
}
