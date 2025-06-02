package appconfig

import (
	"github.com/SaiNageswarS/go-api-boot/config"
)

type AppConfig struct {
	config.BootConfig `ini:",extends"`

	MongoURI          string `env:"MONGO-URI" ini:"mongo_uri"`
	TemporalHostPort  string `env:"TEMPORAL-HOST-PORT" ini:"temporal_host_port"`
	SearchIndexBucket string `env:"SEARCH-INDEX-BUCKET" ini:"search_index_bucket"`
}
