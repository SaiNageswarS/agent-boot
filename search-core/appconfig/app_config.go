package appconfig

import (
	"github.com/SaiNageswarS/go-api-boot/config"
)

type AppConfig struct {
	config.BootConfig `ini:",extends"`

	MongoURI string `ini:"mongo_uri"`

	TemporalHostPort    string `ini:"temporal_host_port"`
	TemporalGoTaskQueue string `ini:"temporal_go_task_queue"`
	TemporalPyTaskQueue string `ini:"temporal_py_task_queue"`
}
