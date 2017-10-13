package main

import (
	"log"
	"math"
	"time"
)

type KogiaConfig struct {
	Connection *ConnectionConfig  `yaml:"connection"`
	Defaults   *AutoscaleConfig   `yaml:"defaults,omitempty"`
	Services   []*AutoscaleConfig `yaml:"services"`
}

type ConnectionConfig struct {
	AMQP   string `yaml:"amqp"`
	Docker string `yaml:"docker"`
}

type AutoscaleConfig struct {
	Service string   `yaml:"service,omitempty"`
	Metrics []string `yaml:"metrics,omitempty"`
	// Coverage is the percent of samples that must be above the threshold to
	// affect the required replica size.
	Coverage float64 `yaml:"coverage,omitempty"`
	// Threshold is the targeted maximum number of message a replica can
	// have as a backqueue before more replicas make sense
	Threshold   int `yaml:"threshold,omitempty"`
	MinReplicas int `yaml:"min_replicas,omitempty"`
	MaxReplicas int `yaml:"max_replicas,omitempty"`
	// SampleRate is the time between measurements of the metric.
	SampleRate time.Duration `yaml:"sample_rate,omitempty"`
	// ScaleRate is the time between rescaling calculations. It must be a multiple
	// of the SampleRate.
	ScaleRate time.Duration `yaml:"scale_rate,omitempty"`
	// UpscaleDelay is the time between a rescale anupscalingd the next upscaling. It must
	// be a multiple of ScaleRate.
	UpscaleDelay time.Duration `yaml:"upscale_delay,omitempty"`
	// DownscaleDelay is the time between a rescale and the next downscaling. It must
	// be a multiple of ScaleRate and should be longer than the UpscaleDelay to
	// prevent flapping.
	DownscaleDelay time.Duration `yaml:"downscale_delay,omitempty"`
}

// Validate validates the autoscaler configuration if all required values are specified
// and if the values adhere to the rules mentioned in the documentation.
func (cfg AutoscaleConfig) Validate() map[string]string {
	errors := map[string]string{}

	if cfg.Service == "" {
		errors["service"] = "can't be empty"
	}
	if len(cfg.Metrics) == 0 {
		errors["metrics"] = "can't be empty"
	}
	if cfg.Coverage <= 0 || cfg.Coverage > 1 {
		errors["coverage"] = "invalid coverage, must be between 0 and 1"
	}
	if cfg.Threshold <= 0 {
		errors["threshold"] = "can't be negative or zero"
	}
	if cfg.MinReplicas < 0 {
		errors["min_replicas"] = "can't be negative"
	}
	if cfg.MaxReplicas < 0 {
		errors["max_replicas"] = "can't be negative"
	}
	if cfg.MaxReplicas < cfg.MinReplicas {
		errors["max_replicas"] = "can't be less than min_replicas"
	}
	if cfg.SampleRate < time.Second {
		errors["sample_rate"] = "can't be less than 1 second"
	}
	if (cfg.ScaleRate % cfg.SampleRate) != 0 {
		errors["scale_rate"] = "must be multiple of sample_rate"
	}
	if (cfg.UpscaleDelay % cfg.ScaleRate) != 0 {
		errors["upscale_delay"] = "must be multiple of scale_rate"
	}
	if (cfg.DownscaleDelay % cfg.ScaleRate) != 0 {
		errors["downscale_delay"] = "must be multiple of scale_rate"
	}

	return errors
}

// FillWithDefaults fills all missing or empty values with the default values
// passed as argument.
func (cfg *AutoscaleConfig) FillWithDefaults(def AutoscaleConfig) {
	if cfg.Service == "" {
		cfg.Service = def.Service
	}
	if len(cfg.Metrics) == 0 {
		cfg.Metrics = def.Metrics
	}
	if cfg.Coverage == 0 {
		cfg.Coverage = def.Coverage
	}
	if cfg.Threshold == 0 {
		cfg.Threshold = def.Threshold
	}
	if cfg.MinReplicas == 0 {
		cfg.MinReplicas = def.MinReplicas
	}
	if cfg.MaxReplicas == 0 {
		cfg.MaxReplicas = def.MaxReplicas
	}
	if cfg.SampleRate == 0 {
		cfg.SampleRate = def.SampleRate
	}
	if cfg.ScaleRate == 0 {
		cfg.ScaleRate = def.ScaleRate
	}
	if cfg.UpscaleDelay == 0 {
		cfg.UpscaleDelay = def.UpscaleDelay
	}
	if cfg.DownscaleDelay == 0 {
		cfg.DownscaleDelay = def.DownscaleDelay
	}
}

type Autoscaler struct {
	Config        *AutoscaleConfig
	Input         Input
	Orchestration Orchestration
}

func (scaler Autoscaler) Run() {
	ticks := 0
	rescaled := math.MinInt64
	samples := []int{}

	scaleRatio := int(scaler.Config.ScaleRate / scaler.Config.SampleRate)
	upscaleDelay := int(scaler.Config.UpscaleDelay / scaler.Config.ScaleRate)
	downscaleDelay := int(scaler.Config.DownscaleDelay / scaler.Config.ScaleRate)

	log.Printf("starting autoscaler for %s", scaler.Config.Service)
	for range time.Tick(scaler.Config.SampleRate) {
		ticks++

		// collect metrics
		metric := 0
		for _, name := range scaler.Config.Metrics {
			m, err := scaler.Input.GetMetric(name)
			if err != nil {
				log.Printf("ERROR: failed to get metric %q for %s: %v", name, scaler.Config.Service, err)
				continue
			}

			metric += m
		}

		samples = append(samples, metric)
		if ticks%scaleRatio == 0 {
			samples = samples[len(samples)-scaleRatio:] // trim samples
			newReplicas := required(samples, scaler.Config.Coverage, scaler.Config.Threshold)
			newReplicas = bound(newReplicas, scaler.Config.MinReplicas, scaler.Config.MaxReplicas)

			curReplicas, err := scaler.Orchestration.GetReplicas(scaler.Config.Service)
			if err != nil {
				log.Printf("ERROR: failed to get replica count for %s: %v", scaler.Config.Service, err)
				continue
			}

			if newReplicas > curReplicas {
				// upscale
				now := ticks / scaleRatio
				if now >= (rescaled + upscaleDelay) {
					log.Printf("%s scaled to %d", scaler.Config.Service, newReplicas)
					err = scaler.Orchestration.SetReplicas(scaler.Config.Service, newReplicas)
					if err != nil {
						log.Printf("ERROR: failed to scale %s: %v", scaler.Config.Service, err)
						continue
					}

					rescaled = now
				}
			}

			if newReplicas < curReplicas {
				// downscale
				now := ticks / scaleRatio
				if now >= (rescaled + downscaleDelay) {
					log.Printf("%s scaled to %d", scaler.Config.Service, newReplicas)
					err = scaler.Orchestration.SetReplicas(scaler.Config.Service, newReplicas)
					if err != nil {
						log.Printf("ERROR: failed to scale %s: %v", scaler.Config.Service, err)
						continue
					}

					rescaled = now
				}
			}
		}
	}
}

func required(samples []int, coverage float64, target int) int {
	req := int(float64(len(samples)) * coverage)
	if req <= 0 || req > len(samples) {
		return -1
	}

	for replicas := 0; ; replicas++ {
		cnt := 0
		for _, val := range samples {
			if val >= (replicas*target)+1 {
				cnt++
			}
		}

		if cnt < req {
			return replicas
		}
	}
}

func bound(num, min, max int) int {
	if num < min {
		num = min
	}
	if num > max {
		num = max
	}
	return num
}
