package dockerdaemonconfiguration

import (
	"github.com/stackrox/rox/benchmarks/checks/utils"
	"github.com/stackrox/rox/generated/api/v1"
)

type disableExperimentalBenchmark struct{}

func (c *disableExperimentalBenchmark) Definition() utils.Definition {
	return utils.Definition{
		BenchmarkCheckDefinition: v1.BenchmarkCheckDefinition{
			Name:        "CIS Docker v1.1.0 - 2.17",
			Description: "Ensure experimental features are avoided in production",
		}, Dependencies: []utils.Dependency{utils.InitInfo},
	}
}

func (c *disableExperimentalBenchmark) Run() (result v1.BenchmarkCheckResult) {
	if utils.DockerInfo.ExperimentalBuild {
		utils.Warn(&result)
		utils.AddNotes(&result, "Docker is running in experimental mode")
		return
	}
	utils.Pass(&result)
	return
}

// NewDisableExperimentalBenchmark implements CIS-2.17
func NewDisableExperimentalBenchmark() utils.Check {
	return &disableExperimentalBenchmark{}
}
