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

func ProbeImageFetcher(ctx context.Context, target string, module config.Module, registry *prometheus.Registry, logger log.Logger) bool {
	probeImageFetchedGauge := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "probe_image_fetched",
		Help: "Returns 1 if the imagefetcher fetched an image and 0 otherwise",
	})
	registry.MustRegister(probeImageFetchedGauge)

	args := []string{"-c", module.ImageFetcher.Client, "-s", target}
	cmd := exec.Command("imagefetcher", args...)
	inChan := make(chan bool)

	go func(inChan chan bool) {
		var timeout int = module.ImageFetcher.Timeout
		if timeout <= 0 {
			timeout = 20
		}
		timer := time.NewTimer(30 * time.Second)
		// Wait for timer to expire or for completed execution signal
		select {
		case <-timer.C:
			cmd.Process.Kill()
			fmt.Println("Execution of imagefetcher ", args, "exceeded timout of", timeout, "seconds")
		case <-inChan:
			timer.Stop()
		}
	}(inChan)

	fmt.Println("Executing command imagefetcher ", args)
	out, err := cmd.CombinedOutput()
	// Signal to timer that command has terminated
	inChan <- true
	// Remove any .jpg file in the current directory
	exec.Command("bash", "-c", "find -name \"*.jpg\" -delete").Run()
	if err != nil {
		//level.Error(logger).Log("msg", "Error running command", "err", err)
		fmt.Println("Error in executing imagefetcher", args, ": ", err, string(out))
		return false
	}
	//level.Info(logger).Log("msg", "Output of command "+target, "output", string(out))
	fmt.Println("Output of command imagefetcher", args, ": ", string(out))

	var regex string = module.BWTester.ValidationRegex
	if len(regex) == 0 {
		regex = "(?s).*Done, exiting.*"
	}
	matched, err := regexp.MatchString(regex, string(out))

	if err != nil {
		fmt.Println("Error in matching output: ", err)
		return false
	}
	//level.Info(logger).Log("msg", "Matching result against '"+module.Exec.ValidationRegex+"'", "match", matched)
	//fmt.Println("Matching result against '" + module.Exec.ValidationRegex+"'", "match", matched)

	if matched {
		probeImageFetchedGauge.Set(1)
	} else {
		fmt.Println("Failed matching output of imagefetcher", args, "with:", regex)
		probeImageFetchedGauge.Set(0)
	}

	return matched
}
