package prober

import (
	"context"
	"time"

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
	probeExecExpectedAnswwerGauge := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "probe_exec_expected_answer",
		Help: "Returns 1 if the executable returned the expected answer and 0 otherwise or if an error occurred",
	})
	registry.MustRegister(probeExecExpectedAnswwerGauge)

	inChan := make(chan bool)
	args := module.Exec.Arguments
	cmd := exec.Command(target, args...)
	cmdString := fmt.Sprintf("%s %v", target, args)
	var timeout int = module.Exec.Timeout
	if timeout <= 0 {
		timeout = 10
	}
	var regex string = module.Exec.ValidationRegex
	// If no regex is provided then match anything but the empty string
	if len(regex) == 0 {
		regex = "(?s).+"
	}

	// Timer for command execution
	go func(inChan chan bool) {
		timer := time.NewTimer(time.Duration(timeout) * time.Second)
		// Wait for timer to expire or for completed execution signal
		select {
		case <-timer.C:
			cmd.Process.Kill()
			level.Error(logger).Log("msg", fmt.Sprintf("Execution of %s exceeded timeout of %is", cmdString, timeout))
			//fmt.Printf("Execution of %s %v exceeded timout of %is.", target, args, timeout)
		case <-inChan:
			timer.Stop()
		}
	}(inChan)

	//fmt.Printf("Executing command %s %v with timeout %is", target, module.Exec.Arguments, module.Exec)
	level.Info(logger).Log("msg", fmt.Sprintf("Starting execution of %s with timeout of %is", cmdString, timeout))
	out, err := cmd.CombinedOutput()

	// Signal to timer that command has terminated
	inChan <- true

	if err != nil {
		level.Error(logger).Log("msg", fmt.Sprintf("Error while executing %s", cmdString), "err", err)
		//fmt.Println("Error: ", err)
		probeExecExpectedAnswwerGauge.Set(0)
		return false
	}

	level.Info(logger).Log("msg", fmt.Sprintf("%s successfully terminated", target), "output", string(out))
	//fmt.Println("Output of command "+target, "output", string(out))

	//fmt.Println("Checking regex")
	matched, err := regexp.MatchString(regex, string(out))

	if err != nil {
		//fmt.Println("Error while matching output: ", err)
		level.Error(logger).Log("msg", fmt.Sprintf("Error while matching %s against %s", string(out), regex), "err", err)
		probeExecExpectedAnswwerGauge.Set(0)
		return false
	}

	level.Info(logger).Log("msg", fmt.Sprintf("Matched %s against %s", string(out), regex), "matches", matched)
	//fmt.Println("Matching result against '"+module.Exec.ValidationRegex+"'", "match", matched)
	if matched {
		probeExecExpectedAnswwerGauge.Set(1)
	} else {
		probeExecExpectedAnswwerGauge.Set(0)
	}

	return matched
}
