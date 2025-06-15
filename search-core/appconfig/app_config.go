package appconfig

import (
	"github.com/SaiNageswarS/go-api-boot/config"
)

type AppConfig struct {
	config.BootConfig `ini:",extends"`

	TemporalHostPort    string `ini:"temporal_host_port"`
	TemporalGoTaskQueue string `ini:"temporal_go_task_queue"`
	TemporalPyTaskQueue string `ini:"temporal_py_task_queue"`
	SignUpAllowed       bool   `ini:"sign_up_allowed"`
	VectorSearchEnabled bool   `ini:"vector_search_enabled"`
	TextSearchEnabled   bool   `ini:"text_search_enabled"`
}
