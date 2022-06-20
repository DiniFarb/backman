package config

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"
)

var (
	config     Config
	once       sync.Once
	configFile string = "config.json"
)

type Config struct {
	Port               int
	LogLevel           string `json:"log_level"`
	LoggingTimestamp   bool   `json:"logging_timestamp"`
	Username           string
	Password           string
	DisableWeb         bool               `json:"disable_web"`
	DisableMetrics     bool               `json:"disable_metrics"`
	DisableRestore     bool               `json:"disable_restore"`
	UnprotectedMetrics bool               `json:"unprotected_metrics"`
	Notifications      NotificationConfig `json:"notifications"`
	S3                 S3Config
	Services           map[string]Service
	Foreground         bool
}

type S3Config struct {
	DisableSSL          bool   `json:"disable_ssl"`
	SkipSSLVerification bool   `json:"skip_ssl_verification"`
	ServiceType         string `json:"service_type"`
	ServiceLabel        string `json:"service_label"` // fallback for service_type, for backwards compatibility
	ServiceName         string `json:"service_name"`
	BucketName          string `json:"bucket_name"`
	EncryptionKey       string `json:"encryption_key"`
	// optional values, backman will try to find them in config.Services.Binding or VCAP_SERVICES if not defined here
	// order of precedence: S3Config.* > Config.Services.Binding > VCAP_SERVICES
	Host      string // optional
	AccessKey string `json:"access_key"` // optional
	SecretKey string `json:"secret_key"` // optional
}

type NotificationConfig struct {
	Teams TeamsNotificationConfig `json:"teams,omitempty"`
}

type TeamsNotificationConfig struct {
	Webhook string   `json:"webhook"`
	Events  []string `json:"events"`
}

func SetConfigFile(file string) {
	configFile = file
}

func Init() {
	Get() // initializes config struct
}

func Get() *Config {
	once.Do(func() {
		config = *new()
	})
	return &config
}

func new() *Config {
	// initialize
	config = Config{
		Services: make(map[string]Service),
	}

	// first, load the config file if it exists
	if _, err := os.Stat(configFile); err == nil {
		data, err := ioutil.ReadFile(configFile)
		if err != nil {
			log.Printf("could not load '%s'\n", configFile)
			log.Fatalln(err.Error())
		}
		if err := json.Unmarshal(data, &config); err != nil {
			log.Printf("could not parse '%s'\n", configFile)
			log.Fatalln(err.Error())
		}
	}

	// now load and overwrite with env provided config, if it exists
	env := os.Getenv(BackmanConfig)
	if len(env) > 0 {
		envConfig := Config{}
		if err := json.Unmarshal([]byte(env), &envConfig); err != nil {
			log.Printf("could not parse environment variable '%s'\n", BackmanConfig)
			log.Fatalln(err.Error())
		}

		// merge config values
		if envConfig.Port > 0 {
			config.Port = envConfig.Port
		}
		if len(envConfig.LogLevel) > 0 {
			config.LogLevel = envConfig.LogLevel
		}
		if envConfig.LoggingTimestamp {
			config.LoggingTimestamp = envConfig.LoggingTimestamp
		}
		if len(envConfig.Username) > 0 {
			config.Username = envConfig.Username
		}
		if len(envConfig.Password) > 0 {
			config.Password = envConfig.Password
		}
		if envConfig.DisableWeb {
			config.DisableWeb = envConfig.DisableWeb
		}
		if envConfig.DisableMetrics {
			config.DisableMetrics = envConfig.DisableMetrics
		}
		if envConfig.DisableRestore {
			config.DisableRestore = envConfig.DisableRestore
		}
		if envConfig.UnprotectedMetrics {
			config.UnprotectedMetrics = envConfig.UnprotectedMetrics
		}

		// teams notifications
		if len(envConfig.Notifications.Teams.Webhook) > 0 {
			config.Notifications.Teams.Webhook = envConfig.Notifications.Teams.Webhook
		}
		if len(envConfig.Notifications.Teams.Events) > 0 {
			config.Notifications.Teams.Events = envConfig.Notifications.Teams.Events
		}

		// s3
		if envConfig.S3.DisableSSL {
			config.S3.DisableSSL = envConfig.S3.DisableSSL
		}
		if envConfig.S3.SkipSSLVerification {
			config.S3.SkipSSLVerification = envConfig.S3.SkipSSLVerification
		}
		if len(envConfig.S3.ServiceType) > 0 {
			config.S3.ServiceType = envConfig.S3.ServiceType
		}
		if len(envConfig.S3.ServiceLabel) > 0 {
			config.S3.ServiceLabel = envConfig.S3.ServiceLabel
		}
		if len(envConfig.S3.ServiceName) > 0 {
			config.S3.ServiceName = envConfig.S3.ServiceName
		}
		if len(envConfig.S3.BucketName) > 0 {
			config.S3.BucketName = envConfig.S3.BucketName
		}
		if len(envConfig.S3.EncryptionKey) > 0 {
			config.S3.EncryptionKey = envConfig.S3.EncryptionKey
		}
		if len(envConfig.S3.Host) > 0 {
			config.S3.Host = envConfig.S3.Host
		}
		if len(envConfig.S3.AccessKey) > 0 {
			config.S3.AccessKey = envConfig.S3.AccessKey
		}
		if len(envConfig.S3.SecretKey) > 0 {
			config.S3.SecretKey = envConfig.S3.SecretKey
		}

		// services
		for serviceName, serviceConfig := range envConfig.Services {
			mergedServiceConfig := config.Services[serviceName]
			if len(serviceConfig.Schedule) > 0 {
				mergedServiceConfig.Schedule = serviceConfig.Schedule
			}
			if serviceConfig.Timeout.Seconds() > 1 {
				mergedServiceConfig.Timeout = serviceConfig.Timeout
			}
			if serviceConfig.Retention.Days > 0 {
				mergedServiceConfig.Retention.Days = serviceConfig.Retention.Days
			}
			if serviceConfig.Retention.Files > 0 {
				mergedServiceConfig.Retention.Files = serviceConfig.Retention.Files
			}
			if serviceConfig.DirectS3 {
				mergedServiceConfig.DirectS3 = serviceConfig.DirectS3
			}
			if serviceConfig.DisableColumnStatistics {
				mergedServiceConfig.DisableColumnStatistics = serviceConfig.DisableColumnStatistics
			}
			if serviceConfig.LogStdErr {
				mergedServiceConfig.LogStdErr = serviceConfig.LogStdErr
			}
			if serviceConfig.ForceImport {
				mergedServiceConfig.ForceImport = serviceConfig.ForceImport
			}
			if len(serviceConfig.LocalBackupPath) > 0 {
				mergedServiceConfig.LocalBackupPath = serviceConfig.LocalBackupPath
			}
			if len(serviceConfig.IgnoreTables) > 0 {
				mergedServiceConfig.IgnoreTables = serviceConfig.IgnoreTables
			}
			if len(serviceConfig.BackupOptions) > 0 {
				mergedServiceConfig.BackupOptions = serviceConfig.BackupOptions
			}
			if len(serviceConfig.RestoreOptions) > 0 {
				mergedServiceConfig.RestoreOptions = serviceConfig.RestoreOptions
			}

			// bindings
			if len(serviceConfig.Binding.Type) > 0 {
				mergedServiceConfig.Binding.Type = serviceConfig.Binding.Type
			}
			if len(serviceConfig.Binding.Provider) > 0 {
				mergedServiceConfig.Binding.Provider = serviceConfig.Binding.Provider
			}
			if len(serviceConfig.Binding.Host) > 0 {
				mergedServiceConfig.Binding.Host = serviceConfig.Binding.Host
			}
			if serviceConfig.Binding.Port > 0 {
				mergedServiceConfig.Binding.Port = serviceConfig.Binding.Port
			}
			if len(serviceConfig.Binding.URI) > 0 {
				mergedServiceConfig.Binding.URI = serviceConfig.Binding.URI
			}
			if len(serviceConfig.Binding.Username) > 0 {
				mergedServiceConfig.Binding.Username = serviceConfig.Binding.Username
			}
			if len(serviceConfig.Binding.Password) > 0 {
				mergedServiceConfig.Binding.Password = serviceConfig.Binding.Password
			}
			if len(serviceConfig.Binding.Database) > 0 {
				mergedServiceConfig.Binding.Database = serviceConfig.Binding.Database
			}

			config.Services[serviceName] = mergedServiceConfig
		}
	}

	// set port if missing
	if config.Port == 0 {
		config.Port, _ = strconv.Atoi(os.Getenv(BackmanPort))
	}

	// set loglevel if missing
	if len(config.LogLevel) == 0 {
		config.LogLevel = "info"
	}

	// use username & password from env if defined
	if os.Getenv(BackmanUsername) != "" {
		config.Username = os.Getenv(BackmanUsername)
	}
	if os.Getenv(BackmanPassword) != "" {
		config.Password = os.Getenv(BackmanPassword)
	}

	// use s3 encryption key from env if defined
	if os.Getenv(BackmanEncryptionKey) != "" {
		config.S3.EncryptionKey = os.Getenv(BackmanEncryptionKey)
	}

	// use teams webhook url from env if defined
	if os.Getenv(BackmanTeamsWebhook) != "" {
		config.Notifications.Teams.Webhook = os.Getenv(BackmanTeamsWebhook)
	}

	// use teams events configuration from env if defined
	if os.Getenv(BackmanTeamsEvents) != "" {
		var events []string
		eventsString := os.Getenv(BackmanTeamsEvents)
		if eventsString != "" {
			events = strings.Split(eventsString, ",")
		}

		config.Notifications.Teams.Events = events
	}

	return &config
}
