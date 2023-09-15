package syncer

import (
	"context"
	"github.com/sirupsen/logrus"
	"path/filepath"
	"sync"
)

const (
	inputDir  = "/tmp/iopipe/input"
	outputDir = "/tmp/iopipe/output"
)

// ConfReader will read config inputs from its source.
type ConfReader interface {
	Read(ctx context.Context) ([]OutputConfig, error)
}

// ConfWriter is used to write configs to their destinations.
type ConfWriter interface {
	Write(ctx context.Context, outputConfigs []OutputConfig) error
}

// Step represents a group of configs to be executed.
type Step struct {
	ID        string            `yaml:"id"`
	Configs   map[string]string `yaml:"configs"`
	InputDir  string            `yaml:"inputDir,omitempty"`
	OutputDir string            `yaml:"outputDir,omitempty"`
}

// OutputConfig holds all the required data for new config to be stored.
// Name: config name is usually a file name like infile-1.1.txt.
// Content: config output content is usually the entire file content from the input source.
type OutputConfig struct {
	Name    string
	Content string
}

// readResult is response from a reader.
// It is usually passed over to writer to be written.
type readResult struct {
	outputConfigs []OutputConfig
	err           error
}

// writeResult is response from a writer.
// Here we can listen for errors and handle different responses from a writer.
type writeResult struct {
	err error
}

// Sync will go through all steps and look for input config files and pass them over to writer.
// Each input config component (dir) from one step is handled concurrently,
// and all files from a single directory is collected concurrently.
// Steps are executed sequentially.
func Sync(ctx context.Context, steps []Step) error {
	for _, step := range steps {
		if step.InputDir == "" {
			step.InputDir = inputDir
		}
		if step.OutputDir == "" {
			step.OutputDir = outputDir
		}

		syncConfig(ctx, step)
	}

	return nil
}

// syncConfig will read configs from a reader and write new configs using a writer.
// It will read config files from an input directory, zip it and store it to an output directory.
// Each config is processed in separate goroutine.
// Each operation (read, bulk, write) per config is processed in separate goroutine.
func syncConfig(ctx context.Context, step Step) {
	wg := sync.WaitGroup{}

	for parent, path := range step.Configs {
		wg.Add(1)

		go func(wg *sync.WaitGroup, parent, path string) {
			defer wg.Done()

			inputPath := filepath.Join(step.InputDir, parent, path)

			confReader := NewFileConfReader(inputPath)
			confWriter := NewFileConfWriter(step.ID, step.OutputDir)

			readRes := readConfigs(ctx, confReader)
			bulkRes := collectAsBulk(ctx, readRes)
			// optionally add other filters in the pipeline
			writeRes := writeConfigs(ctx, confWriter, bulkRes)

			handleWriteResult(ctx, writeRes)
		}(&wg, parent, path)
	}

	wg.Wait()
}

// readConfigs will read input configs using the given confReader.
// Read configs are returned through readRes channel for further processing.
func readConfigs(ctx context.Context, confReader ConfReader) <-chan readResult {
	readRes := make(chan readResult)

	go func() {
		defer close(readRes)

		select {
		case <-ctx.Done():
			return
		default:
			configs, err := confReader.Read(ctx)
			if err != nil {
				readRes <- readResult{err: err}
				return
			}

			readRes <- readResult{outputConfigs: configs}
		}
	}()

	return readRes
}

// collectAsBulk will collect all read configs from readRes and send them as a bulk through bulkRes channel.
func collectAsBulk(ctx context.Context, readRes <-chan readResult) <-chan readResult {
	bulkResult := make(chan readResult)

	go func() {
		var outputs []OutputConfig

		defer func() {
			bulkResult <- readResult{outputConfigs: outputs}
			close(bulkResult)
		}()

		for {
			select {
			case <-ctx.Done():
				return
			case r, ok := <-readRes:
				if !ok {
					return
				}
				outputs = append(outputs, r.outputConfigs...)
			}
		}
	}()

	return bulkResult
}

// writeConfigs will write output configs using the given confWriter.
// Input configs for output are received from readRes channel.
// Results from written configs are returned through writeRes channel.
func writeConfigs(ctx context.Context, confWriter ConfWriter, readRes <-chan readResult) <-chan writeResult {
	writeRes := make(chan writeResult)

	go func() {
		defer close(writeRes)

		for {
			select {
			case <-ctx.Done():
				return
			case r, ok := <-readRes:
				if !ok {
					return
				}

				if r.err != nil {
					writeRes <- writeResult{err: r.err}
					break
				}

				if err := confWriter.Write(ctx, r.outputConfigs); err != nil {
					writeRes <- writeResult{err: err}
				}

				writeRes <- writeResult{}
			}
		}
	}()

	return writeRes
}

func handleWriteResult(ctx context.Context, writeRes <-chan writeResult) {
	for {
		select {
		case <-ctx.Done():
			return
		case r, ok := <-writeRes:
			if !ok {
				return
			}

			if r.err != nil {
				logrus.Error(r.err)
			}
		}
	}
}
