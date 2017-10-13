package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"

	yaml "gopkg.in/yaml.v2"
)

type Input interface {
	GetMetric(name string) (int, error)
}

type Orchestration interface {
	GetReplicas(name string) (int, error)
	SetReplicas(name string, count int) error
}

func main() {
	testOnly := flag.Bool("t", false, "test configuration and exit")
	configFile := flag.String("c", "kogia.yml", "configuration file")
	flag.Parse()

	// --- read configuration
	var cfg KogiaConfig
	source, err := ioutil.ReadFile(*configFile)
	if err != nil {
		log.Fatal(err)
	}

	err = yaml.Unmarshal(source, &cfg)
	if err != nil {
		log.Fatal(err)
	}

	if *testOnly {
		fmt.Printf("the configuration file %s syntax is ok\n", *configFile)
	}

	// --- validate configuration
	valid := true
	for _, service := range cfg.Services {
		service.FillWithDefaults(*cfg.Defaults)
		errors := service.Validate()
		if len(errors) > 0 {
			valid = false
			fmt.Printf("failed to validate monitor definition for %q:\n", service.Service)
			for name, err := range errors {
				fmt.Printf("- %-16s: %s\n", name, err)
			}
			fmt.Printf("\n")
		}
	}

	if !valid {
		os.Exit(1)
	}

	if *testOnly {
		fmt.Printf("configuration file %s test is successful\n", *configFile)
		os.Exit(0)
	}

	// --- connect to input and orchestrator
	quit := make(chan bool)
	input := NewAMQPInput(cfg.Connection.AMQP)
	orchestrator := NewDockerSwarm(cfg.Connection.Docker)

	// --- run autoscalers
	for _, service := range cfg.Services {
		scaler := Autoscaler{
			Input:         input,
			Orchestration: orchestrator,
			Config:        service,
		}

		go scaler.Run()
	}

	<-quit // block main routine, work is happening in Autoscaler
}
