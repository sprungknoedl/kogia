[![Build Status](https://travis-ci.org/sprungknoedl/kogia.svg?branch=master)](https://travis-ci.org/sprungknoedl/kogia)
# kogia

Dynamically scale docker swarm services using the number of unacknowledged 
messages from an AMQP queue like RabbitMQ. This is useful if you use AMQP 
as a job queue and the number of messages indicates the load on your system.

## Installation
```
go get github.com/sprungknoedl/kogia
```

## Configuration
kogia is configured using a YAML configuration file. Below is an example 
configuration for the docker swarm service _helloworld_:
```yaml
connection:
  amqp: amqp://localhost:5672/
  docker: unix:///var/run/docker.sock
defaults:
  sample_rate: 2s
  scale_rate: 30s
  upscale_delay: 3m
  downscale_delay: 5m
  coverage: .75
services:
  - service: helloworld
    queue: helloworld
    threshold: 10
    minpods: 1
    maxpods: 10
```

### Parameters
kogia can scale multiple docker swarm services at once, all with different
scale intervals and delays.

Each service has the following configuration parameters. If the parameter
is not provided for a service, the default value specified in the configuration
will be used. kogia doesn't provide any internal defaults!

* `service`: name of the docker swarm service to scale.
* `metric`: name of the AMQP queue used as load indicator.
* `coverage`: percentage of metrics required to calculate average queue length (recommended: `0.75`).
* `threshold`: number of messages on a queue representing maximum load of **one** service replica.
* `min_replicas`: minimum number of replicas for this service. kogia will never scale the service below this number. It is safe to specify `0`; as soon as some message are queued, kogia will scale the replicas to `1`.
* `max_replicas`: maximum number of replicas for this service. kogia will never scale the service above this number.
* `sample_rate`: time interval between measurements of the metric (recommended: `2s`).
* `scale_rate`: time interval between autoscaling calculations. Must be a multiple of *sample_rate* (recommended: `30s`).
* `upscale_delay`: minimum time between the last rescaling and the next upscaling (recommended: `3m`).
* `downscale_delay`: minimum time between the last rescaling and the next downscaling. The *downscale_delay* should be higher than the *upscale_delay* to prevent flapping (recommended: `5m`).