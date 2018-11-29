package containerruntime

import (
	//"context"

	"github.com/stackrox/rox/benchmarks/checks/utils"
	"github.com/stackrox/rox/generated/api/v1"
)

type latestImageBenchmark struct{}

func (c *latestImageBenchmark) Definition() utils.Definition {
	return utils.Definition{
		BenchmarkCheckDefinition: v1.BenchmarkCheckDefinition{
			Name:        "CIS Docker v1.1.0 - 5.27",
			Description: "Ensure docker commands always get the latest version of the image",
		}, Dependencies: []utils.Dependency{utils.InitImages},
	}
}

func (c *latestImageBenchmark) Run() (result v1.BenchmarkCheckResult) {
	utils.Note(&result)
	utils.AddNotes(&result, "Pulling images is invasive and not always possible depending on credential management")
	return
}

// NewLatestImageBenchmark implements CIS-5.27
func NewLatestImageBenchmark() utils.Check {
	return &latestImageBenchmark{}
}
