package utils

import (
	"context"
	"errors"
	"fmt"
	"sync"
)

type ObjectID string

var (
	Cache           = ObjectID("cache")
	MaxMindProvider = ObjectID("maxmind-provider")
	DbIpProvider    = ObjectID("dbip-provider")
	IpStackProvider = ObjectID("ipstack-provider")
	GlobioProvider  = ObjectID("globio-provider")
)

type IoCContainer struct {
	sync.RWMutex
	objects map[ObjectID]interface{}
}

var (
	Container = &IoCContainer{
		objects: make(map[ObjectID]interface{}),
	}
)

func (c *IoCContainer) Fetch(ctx context.Context, name ObjectID) interface{} {
	c.RLock()
	defer c.RUnlock()

	if c.objects[name] == nil {
		panic(fmt.Sprintf("no object named %s found in the container", name))
	}

	obj := c.objects[name]

	return obj
}

func (c *IoCContainer) Assign(ctx context.Context, name ObjectID, obj interface{}) error {
	c.Lock()
	defer c.Unlock()

	if c.objects[name] != nil {
		return errors.New("object already exists with the same name")
	}

	c.objects[name] = obj

	return nil
}

func (c *IoCContainer) Clear(ctx context.Context) {
	c.Lock()
	defer c.Unlock()

	c.objects = make(map[ObjectID]interface{})
}
