package main

import (
	"context"
	"log"

	"github.com/docker/docker/api"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/pkg/errors"
)

type MockOrchestration struct {
	scale int
}

func (swarm MockOrchestration) GetScale(name string) (int, error) {
	return swarm.scale, nil
}

func (swarm *MockOrchestration) SetScale(name string, scale int) error {
	swarm.scale = scale
	return nil
}

type DockerSwarm struct {
	conn *client.Client
}

func NewDockerSwarm(url string) DockerSwarm {
	conn, err := client.NewClient(url, api.DefaultVersion, nil, nil)
	if err != nil {
		log.Fatal(err)
	}

	return DockerSwarm{conn: conn}
}

func (swarm DockerSwarm) GetReplicas(name string) (int, error) {
	ctx := context.Background()
	srv, _, err := swarm.conn.ServiceInspectWithRaw(ctx, name)
	if err != nil {
		return 0, err
	}

	replicas := int(*srv.Spec.Mode.Replicated.Replicas)
	return replicas, nil
}

func (swarm DockerSwarm) SetReplicas(name string, scale int) error {
	ctx := context.Background()
	service, _, err := swarm.conn.ServiceInspectWithRaw(ctx, name)
	if err != nil {
		return err
	}

	serviceMode := &service.Spec.Mode
	if serviceMode.Replicated == nil {
		return errors.Errorf("scale can only be used with replicated mode")
	}

	replicas := uint64(scale)
	serviceMode.Replicated.Replicas = &replicas
	_, err = swarm.conn.ServiceUpdate(ctx, service.ID, service.Version, service.Spec, types.ServiceUpdateOptions{})
	if err != nil {
		return err
	}

	return nil
}
