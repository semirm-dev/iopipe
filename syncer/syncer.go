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
// It will usually read config files from an input directory, zip it and store it to an output directory.
func syncConfig(ctx context.Context, step Step) {
	readRes := make(chan readResult)
	bulkRes := make(chan readResult)

	go collectReadRes(ctx, readRes, bulkRes)

	// we can add other filters in the pipeline before sending data to writer for a final writing
	writeRes := make(chan writeResult)
	confWriter := NewFileConfWriter(step.ID, step.OutputDir)
	go writeConfigs(ctx, confWriter, bulkRes, writeRes)

	wg := &sync.WaitGroup{}
	for comp, path := range step.Configs {
		wg.Add(1)
		inputPath := filepath.Join(step.InputDir, comp, path)

		go func(wg *sync.WaitGroup, inputPath string) {
			defer wg.Done()
			confReader := NewFileConfReader(inputPath)
			readConfigs(ctx, confReader, readRes)
		}(wg, inputPath)
	}
	wg.Wait()
	close(readRes)
	handleWriteResult(writeRes)
}

// readConfigs will read input configs using the given confReader.
// Read configs are passed over to readRes channel for further processing by a writer.
func readConfigs(ctx context.Context, confReader ConfReader, readRes chan readResult) {
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
}

// collectReadRes will collect all read configs from readRes and send them as a bulk to bulkRes.
func collectReadRes(ctx context.Context, readRes chan readResult, bulkRes chan readResult) {
	var outputs []OutputConfig

	defer func() {
		bulkRes <- readResult{outputConfigs: outputs}
		close(bulkRes)
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
}

// writeConfigs will write output configs using the given confWriter.
// Input configs for output are received from readRes channel.
// Results from written configs are passed over to writeRes channel.
func writeConfigs(
	ctx context.Context,
	confWriter ConfWriter,
	readRes chan readResult,
	writeRes chan writeResult,
) {
	defer func() {
		logrus.Infof("writer finished")
		close(writeRes)
	}()

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
				continue
			}

			if err := confWriter.Write(ctx, r.outputConfigs); err != nil {
				writeRes <- writeResult{err: err}
			}
		}
	}
}

func handleWriteResult(writeRes chan writeResult) {
	for {
		select {
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
