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

var (
	timeout time.Duration = 30
	regex   string        = "(?s).*Done, exiting.*"
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
		timer := time.NewTimer(timeout * time.Second)
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
	inChan <- true
	if err != nil {
		//level.Error(logger).Log("msg", "Error running command", "err", err)
		fmt.Println("Error in executing imagefetcher", args, ": ", err, string(out))
		return false
	}
	//level.Info(logger).Log("msg", "Output of command "+target, "output", string(out))
	fmt.Println("Output of command imagefetcher ", args, ": ", string(out))

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
		fmt.Println("Failed matching output of imagefetcher ", args, "with: ")
		probeImageFetchedGauge.Set(0)
	}

	return matched
}
