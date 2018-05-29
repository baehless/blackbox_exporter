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

func ProbeSensorFetcher(ctx context.Context, target string, module config.Module, registry *prometheus.Registry, logger log.Logger) bool {
	probeSensorFetchedGauge := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "probe_temperature_fetched",
		Help: "Returns 1 if the sensorfetcher fetched the temperature and 0 otherwise",
	})
	registry.MustRegister(probeSensorFetchedGauge)

	args := []string{"-c", module.SensorFetcher.Client, "-s", target}
	cmd := exec.Command("sensorfetcher", args...)
	inChan := make(chan bool)

	go func(inChan chan bool) {
		var timeout int = module.SensorFetcher.Timeout
		if timeout <= 0 {
			timeout = 15
		}
		timer := time.NewTimer(time.Duration(timeout) * time.Second)
		// Wait for timer to expire or for completed execution signal
		select {
		case <-timer.C:
			cmd.Process.Kill()
			fmt.Println("Execution of sensorfetcher ", args, "exceeded timout of", timeout, "seconds")
		case <-inChan:
			timer.Stop()
		}
	}(inChan)

	fmt.Println("Executing command sensorfetcher ", args)
	out, err := cmd.CombinedOutput()
	// Signal to timer that command has terminated
	inChan <- true
	if err != nil {
		//level.Error(logger).Log("msg", "Error running command", "err", err)
		fmt.Println("Error in executing sensorfetcher", args, ": ", err, string(out))
		return false
	}
	//level.Info(logger).Log("msg", "Output of command "+target, "output", string(out))
	fmt.Println("Output of command sensorfetcher", args, ": ", string(out))

	var regex string = module.BWTester.ValidationRegex
	if len(regex) == 0 {
		regex = "(?s).*Temperature.*"
	}
	matched, err := regexp.MatchString(regex, string(out))

	if err != nil {
		fmt.Println("Error in matching output:", err)
		return false
	}
	//level.Info(logger).Log("msg", "Matching result against '"+module.Exec.ValidationRegex+"'", "match", matched)
	//fmt.Println("Matching result against '" + module.Exec.ValidationRegex+"'", "match", matched)

	if matched {
		probeSensorFetchedGauge.Set(1)
	} else {
		fmt.Println("Failed matching output of sensorfetcher", args, "with:", regex)
		probeSensorFetchedGauge.Set(0)
	}

	return matched
}
