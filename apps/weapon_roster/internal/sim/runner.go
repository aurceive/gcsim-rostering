package sim

import (
	"context"

	"github.com/genshinsim/gcsim/pkg/model"
	"github.com/genshinsim/gcsim/pkg/optimization"
	"github.com/genshinsim/gcsim/pkg/simulator"
)

type SimulationRunner interface {
	OptimizeAndRun(ctx context.Context, configPath string) (*model.SimulationResult, error)
}

type GcsimRunner struct{}

func (r GcsimRunner) OptimizeAndRun(ctx context.Context, configPath string) (*model.SimulationResult, error) {
	simopt := simulator.Options{
		ConfigPath:       configPath,
		ResultSaveToPath: configPath, // overwrite with optimized
		GZIPResult:       false,
	}

	var res *model.SimulationResult
	err := withSilencedStdoutStderr(func() error {
		optimization.RunSubstatOptim(simopt, false, "")
		simopt.ResultSaveToPath = "" // no save
		var runErr error
		res, runErr = simulator.Run(ctx, simopt)
		return runErr
	})
	if err != nil {
		return nil, err
	}
	return res, nil
}
