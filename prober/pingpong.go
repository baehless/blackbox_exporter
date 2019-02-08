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

func ProbePingPong(ctx context.Context, target string, module config.Module, registry *prometheus.Registry, logger log.Logger) bool {
	pingPongTestExecutedGauge := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "pingpong_test_executed",
		Help: "Returns 1 if at least one successful pingpong exchange took place.",
	})
	registry.MustRegister(pingPongTestExecutedGauge)

	args := []string{"-local", module.PingPong.Local, "-remote", target, "-count", "2"}
	cmd := exec.Command("pingpong", args...)
	inChan := make(chan bool)

	go func(inChan chan bool) {
		var timeout int = module.PingPong.Timeout
		if timeout <= 0 {
			timeout = 10
		}
		timer := time.NewTimer(time.Duration(timeout) * time.Second)
		// Wait for timer to expire or for completed execution signal
		select {
		case <-timer.C:
			cmd.Process.Kill()
			fmt.Println("Execution of pingpong", args, "exceeded timeout of", timeout, "seconds")
		case <-inChan:
			timer.Stop()
		}
	}(inChan)

	fmt.Println("Executing command pingpong ", args)
	out, err := cmd.CombinedOutput()
	// Signal to timer that command has terminated
	inChan <- true
	if err != nil {
		//level.Error(logger).Log("msg", "Error running command", "err", err)
		fmt.Println("Error in executing pingpong", args, ": ", err, string(out))
		return false
	}
	//level.Info(logger).Log("msg", "Output of command "+target, "output", string(out))
	fmt.Println("Output of command pingpong", args, ":", string(out))

	var regex string = module.PingPong.ValidationRegex
	if len(regex) == 0 {
		regex = ".*Received 13 bytes from.*"
	}
	matched, err := regexp.MatchString(regex, string(out))

	if err != nil {
		fmt.Println("Error in matching output:", err)
		return false
	}
	//level.Info(logger).Log("msg", "Matching result against '"+module.Exec.ValidationRegex+"'", "match", matched)
	//fmt.Println("Matching result against '" + module.Exec.ValidationRegex+"'", "match", matched)

	if matched {
		pingPongTestExecutedGauge.Set(1)
	} else {
		fmt.Println("Failed matching output of pingpong", args, "with:", regex)
		pingPongTestExecutedGauge.Set(0)
	}

	return matched
}

