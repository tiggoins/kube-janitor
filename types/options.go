package types

import (
	"time"
	flag "github.com/spf13/pflag"
)

type Options struct {
	DryRun bool
	Force  bool
	Interval time.Duration
}

func ParseOptions() Options {
	var dryRun bool
	var force bool
	var interval time.Duration

	flag.BoolVar(&dryRun, "dry-run", false, "Only log actions without performing any cleanup")
	flag.BoolVar(&force, "force", false, "Force removal of containers/filesystem")
	flag.DurationVar(&interval, "interval", 1*time.Hour, "Interval for daemon mode")

	flag.Parse()

	return Options{
		DryRun:   dryRun,
		Force:    force,
		Interval: interval,
	}
}
		