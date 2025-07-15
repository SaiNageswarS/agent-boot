package appconfig

import (
	"github.com/SaiNageswarS/go-api-boot/config"
)

type AppConfig struct {
	config.BootConfig `ini:",extends"`

	TemporalHostPort    string `ini:"temporal_host_port"`
	TemporalGoTaskQueue string `ini:"temporal_go_task_queue"`
	TemporalPyTaskQueue string `ini:"temporal_py_task_queue"`

	SignUpAllowed bool `ini:"sign_up_allowed"`

	ClaudeVersion   string `ini:"claude_version"`
	ClaudeMini      string `ini:"claude_mini"`
	OllamaModel     string `ini:"ollama_model"`
	OllamaMiniModel string `ini:"ollama_mini_model"`

	TitleGenModel string `ini:"title_gen_model"`
}
