package prober

import (
	"context"
	"fmt"
	"github.com/go-kit/kit/log"
	"github.com/prometheus/blackbox_exporter/config"
	"github.com/prometheus/client_golang/prometheus"
	"os/exec"
	"regexp"
	"time"
)

func ProbeBWTester(ctx context.Context, target string, module config.Module, registry *prometheus.Registry, logger log.Logger) bool {
	probeTestExecutedGauge := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "probe_test_executed",
		Help: "Returns 1 if the bwtester executed the bandwidth test and 0 otherwise",
	})
	registry.MustRegister(probeTestExecutedGauge)

	args := []string{"-c", module.BWTester.Client, "-s", target}
	cmd := exec.Command("bwtestclient", args...)
	inChan := make(chan bool)

	go func(inChan chan bool) {
		var timeout int = module.BWTester.Timeout
		if timeout <= 0 {
			timeout = 30
		}
		timer := time.NewTimer(time.Duration(timeout) * time.Second)
		// Wait for timer to expire or for completed execution signal
		select {
		case <-timer.C:
			cmd.Process.Kill()
			fmt.Println("Execution of bwtestclient", args, "exceeded timout of", timeout, "seconds")
		case <-inChan:
			timer.Stop()
		}
	}(inChan)

	fmt.Println("Executing command bwtestclient ", args)
	out, err := cmd.CombinedOutput()
	// Signal to timer that command has terminated
	inChan <- true
	if err != nil {
		//level.Error(logger).Log("msg", "Error running command", "err", err)
		fmt.Println("Error in executing bwtestclient", args, ": ", err, string(out))
		return false
	}
	//level.Info(logger).Log("msg", "Output of command "+target, "output", string(out))
	fmt.Println("Output of command bwtestclient", args, ":", string(out))

	var regex string = module.BWTester.ValidationRegex
	if len(regex) == 0 {
		regex = "(?s).*results.*"
	}
	matched, err := regexp.MatchString(regex, string(out))

	if err != nil {
		fmt.Println("Error in matching output:", err)
		return false
	}
	//level.Info(logger).Log("msg", "Matching result against '"+module.Exec.ValidationRegex+"'", "match", matched)
	//fmt.Println("Matching result against '" + module.Exec.ValidationRegex+"'", "match", matched)

	if matched {
		probeTestExecutedGauge.Set(1)
	} else {
		fmt.Println("Failed matching output of bwtestclient", args, "with:", regex)
		probeTestExecutedGauge.Set(0)
	}

	return matched
}
