package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"

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
	configfile := flag.String("c", "kogia.yml", "configuration file")
	flag.Parse()

	// --- read configuration
	var cfg KogiaConfig
	source, err := ioutil.ReadFile(*configfile)
	if err != nil {
		log.Fatal(err)
	}

	err = yaml.Unmarshal(source, &cfg)
	if err != nil {
		log.Fatal(err)
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
		return
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
