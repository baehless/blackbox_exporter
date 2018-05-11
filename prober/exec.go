package prober

import (
	"context"
	//"fmt"
	"os/exec"
	"regexp"

	"github.com/go-kit/kit/log"

	"github.com/prometheus/client_golang/prometheus"

	"fmt"
	"github.com/go-kit/kit/log/level"
	"github.com/prometheus/blackbox_exporter/config"
)

func ProbeExec(ctx context.Context, target string, module config.Module, registry *prometheus.Registry, logger log.Logger) bool {
	fmt.Println("Starting")
	probeExecExpectedAnswwerGauge := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "probe_exec_expected_answer",
		Help: "Returns 1 if the executable returned the expected answer and 0 otherwise",
	})
	registry.MustRegister(probeExecExpectedAnswwerGauge)

	fmt.Println("Executing command", target, "with", module.Exec.Arguments)
	out, err := exec.Command(target, module.Exec.Arguments...).Output()
	if err != nil {
		level.Error(logger).Log("msg", "Error running command", "err", err)
		fmt.Println("Error: ", err)
		return false
	}
	level.Info(logger).Log("msg", "Output of command "+target, "output", string(out))
	fmt.Println("Output of command "+target, "output", string(out))

	fmt.Println("Checking regex")
	matched, err := regexp.MatchString(module.Exec.ValidationRegex, string(out))

	if err != nil {
		// TODO: log
		fmt.Println("Error: ", err)
		return false
	}
	level.Info(logger).Log("msg", "Matching result against '"+module.Exec.ValidationRegex+"'", "match", matched)
	fmt.Println("Matching result against '"+module.Exec.ValidationRegex+"'", "match", matched)

	if matched {
		probeExecExpectedAnswwerGauge.Set(1)
	} else {
		probeExecExpectedAnswwerGauge.Set(0)
	}

	return matched
}
