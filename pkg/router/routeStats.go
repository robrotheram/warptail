package router

import (
	"time"
	"warptail/pkg/utils"
)

type RouterStatus string

const (
	STARTING = RouterStatus("Starting")
	RUNNING  = RouterStatus("Running")
	STOPPING = RouterStatus("Stopping")
	STOPPED  = RouterStatus("Stopped")
)

type Route interface {
	Start() error
	Stop() error
	Update(utils.RouteConfig) error
	Config() utils.RouteConfig
	Status() RouterStatus
	Stats() utils.TimeSeriesData
	Ping() time.Duration
}
