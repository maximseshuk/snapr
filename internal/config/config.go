package config

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"

	"github.com/go-playground/validator/v10"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"

	"github.com/maximseshuk/snapr/internal/utils"
)

type Config struct {
	Logs   LogsConfig   `mapstructure:"logs"`
	Server ServerConfig `mapstructure:"server"`
	Jobs   []JobConfig  `mapstructure:"jobs" validate:"required,min=1,dive"`
}

// LogsConfig configures on-disk logs. Files are JSON-lines and rotated internally.
// Stdout always gets the console stream regardless of these settings.
type LogsConfig struct {
	Path       string `mapstructure:"path"`
	System     bool   `mapstructure:"system"`
	PerJob     bool   `mapstructure:"perJob"`
	MaxSizeMB  int    `mapstructure:"maxSizeMB" validate:"omitempty,min=0"`
	MaxBackups int    `mapstructure:"maxBackups" validate:"omitempty,min=0"`
	MaxAgeDays int    `mapstructure:"maxAgeDays" validate:"omitempty,min=0"`
	Compress   bool   `mapstructure:"compress"`
}

// ServerConfig configures the HTTP server. Disabled → scheduler-only daemon.
type ServerConfig struct {
	Enabled         bool               `mapstructure:"enabled"`
	Address         string             `mapstructure:"address" validate:"required_if=Enabled true"`
	Secret          string             `mapstructure:"secret"`
	DefaultLanguage string             `mapstructure:"defaultLanguage" validate:"omitempty,oneof=en ru"`
	Auth            *AuthConfig        `mapstructure:"auth"`
	LogLimits       *LogLimitsConfig   `mapstructure:"logLimits"`
	Permissions     *PermissionsConfig `mapstructure:"permissions"`
	UI              UIConfig           `mapstructure:"ui"`
}

// UIConfig toggles the bundled SPA. Disabled → JSON API and /metrics only.
type UIConfig struct {
	Enabled bool `mapstructure:"enabled"`
}

type AuthConfig struct {
	Enabled         bool          `mapstructure:"enabled"`
	Username        string        `mapstructure:"username" validate:"required_if=Enabled true"`
	Password        string        `mapstructure:"password" validate:"required_if=Enabled true"`
	TokenExpiration int           `mapstructure:"tokenExpiration"`
	Cookies         *CookieConfig `mapstructure:"cookies"`
}

type CookieConfig struct {
	Secure   bool   `mapstructure:"secure"`
	SameSite string `mapstructure:"sameSite"`
	Domain   string `mapstructure:"domain"`
}

type LogLimitsConfig struct {
	JobLogs    int `mapstructure:"jobLogs"`
	SystemLogs int `mapstructure:"systemLogs"`
}

type PermissionsConfig struct {
	AllowBackupDownload bool `mapstructure:"allowBackupDownload"`
	AllowManualRun      bool `mapstructure:"allowManualRun"`
	ShowConfig          bool `mapstructure:"showConfig"`
}

type JobConfig struct {
	Name           string            `mapstructure:"name" validate:"required"`
	Schedule       string            `mapstructure:"schedule" validate:"required"`
	Sources        []SourceConfig    `mapstructure:"sources" validate:"required,min=1,dive"`
	Storages       []StorageConfig   `mapstructure:"storages" validate:"required,min=1,dive"`
	DefaultStorage string            `mapstructure:"defaultStorage" validate:"omitempty"`
	Compression    string            `mapstructure:"compression" validate:"omitempty,oneof=tar tar.gz gzip gz tar.zst zstd zst tar.xz xz zip"`
	Retention      RetentionConfig   `mapstructure:"retention" validate:"required"`
	BeforeScript   string            `mapstructure:"beforeScript,omitempty"`
	AfterScript    string            `mapstructure:"afterScript,omitempty"`
	Notifiers      []NotifierConfig  `mapstructure:"notifiers,omitempty" validate:"omitempty,dive"`
	Encryption     *EncryptionConfig `mapstructure:"encryption,omitempty" validate:"omitempty"`
	Split          *SplitConfig      `mapstructure:"split,omitempty" validate:"omitempty"`
}

// SplitConfig cuts the final archive (post-compression, post-encryption) into
// fixed-size ".part-aaa", ".part-aab" parts. Retention treats the set as one backup.
type SplitConfig struct {
	ChunkSize string `mapstructure:"chunkSize" validate:"required"`
}

type SourceConfig struct {
	Type string `mapstructure:"type" validate:"required,oneof=postgresql mysql mariadb mongodb redis sqlite local bunny s3"`

	Path        string            `mapstructure:"path,omitempty" validate:"required_if=Type sqlite"`
	Excludes    []string          `mapstructure:"excludes,omitempty"`
	ExtraParams map[string]string `mapstructure:"extraParams,omitempty"`

	Host          string   `mapstructure:"host,omitempty" validate:"required_if=Type postgresql"`
	Port          int      `mapstructure:"port,omitempty" validate:"omitempty,min=1,max=65535"`
	Username      string   `mapstructure:"username,omitempty" validate:"required_if=Type postgresql"`
	Password      string   `mapstructure:"password,omitempty"`
	Database      string   `mapstructure:"database,omitempty" validate:"required_if=Type postgresql"`
	ExcludeTables []string `mapstructure:"excludeTables,omitempty"`
	Tables        []string `mapstructure:"tables,omitempty"`
	AllDatabases  bool     `mapstructure:"allDatabases,omitempty"`
	Socket        string   `mapstructure:"socket,omitempty"`
	AuthDatabase  string   `mapstructure:"authDatabase,omitempty"`
	URI           string   `mapstructure:"uri,omitempty"`
	Oplog         bool     `mapstructure:"oplog,omitempty"`

	Endpoint  string `mapstructure:"endpoint,omitempty" validate:"required_if=Type bunny"`
	ZoneName  string `mapstructure:"zoneName,omitempty" validate:"required_if=Type bunny"`
	AccessKey string `mapstructure:"accessKey,omitempty" validate:"required_if=Type bunny"`
	SyncPath  string `mapstructure:"syncPath,omitempty"`

	PullZoneHostname     string `mapstructure:"pullZoneHostname,omitempty"`
	PullZoneTokenAuthKey string `mapstructure:"pullZoneTokenAuthKey,omitempty"`
	PullZoneTokenTTL     int    `mapstructure:"pullZoneTokenTTL,omitempty"`

	Bucket          string `mapstructure:"bucket,omitempty" validate:"required_if=Type s3"`
	Region          string `mapstructure:"region,omitempty" validate:"required_if=Type s3"`
	AccessKeyID     string `mapstructure:"accessKeyId,omitempty" validate:"required_if=Type s3"`
	SecretAccessKey string `mapstructure:"secretAccessKey,omitempty" validate:"required_if=Type s3"`
	UsePathStyle    bool   `mapstructure:"usePathStyle,omitempty"`
}

type StorageConfig struct {
	Name string `mapstructure:"name" validate:"required"`
	Type string `mapstructure:"type" validate:"required,oneof=s3 local bunny sftp webdav"`

	Path string `mapstructure:"path,omitempty" validate:"required_if=Type local"`

	Host          string `mapstructure:"host,omitempty" validate:"required_if=Type sftp"`
	Port          int    `mapstructure:"port,omitempty" validate:"omitempty,min=1,max=65535"`
	Username      string `mapstructure:"username,omitempty"`
	Password      string `mapstructure:"password,omitempty"`
	PrivateKey    string `mapstructure:"privateKey,omitempty"`
	Passphrase    string `mapstructure:"passphrase,omitempty"`
	KnownHosts    string `mapstructure:"knownHosts,omitempty"`
	StrictHostKey *bool  `mapstructure:"strictHostKey,omitempty"`

	URL string `mapstructure:"url,omitempty" validate:"required_if=Type webdav"`

	Bucket          string `mapstructure:"bucket,omitempty" validate:"required_if=Type s3"`
	Region          string `mapstructure:"region,omitempty" validate:"required_if=Type s3"`
	AccessKeyID     string `mapstructure:"accessKeyId,omitempty" validate:"required_if=Type s3"`
	SecretAccessKey string `mapstructure:"secretAccessKey,omitempty" validate:"required_if=Type s3"`
	Endpoint        string `mapstructure:"endpoint,omitempty" validate:"required_if=Type bunny"`
	StorageClass    string `mapstructure:"storageClass,omitempty"`

	DownloadMode string `mapstructure:"downloadMode,omitempty" validate:"omitempty,oneof=proxy signed"`
	SignedURLTTL int    `mapstructure:"signedUrlTTL,omitempty" validate:"omitempty,min=60,max=86400"`

	ZoneName  string `mapstructure:"zoneName,omitempty" validate:"required_if=Type bunny"`
	AccessKey string `mapstructure:"accessKey,omitempty" validate:"required_if=Type bunny"`

	PullZoneHostname     string `mapstructure:"pullZoneHostname,omitempty"`
	PullZoneTokenAuthKey string `mapstructure:"pullZoneTokenAuthKey,omitempty"`
	PullZoneTokenTTL     int    `mapstructure:"pullZoneTokenTTL,omitempty"`
}

type RetentionConfig struct {
	Last int `mapstructure:"last" validate:"required,min=1"`
}

type EncryptionConfig struct {
	Type     string `mapstructure:"type,omitempty" validate:"omitempty,oneof=openssl"`
	Cipher   string `mapstructure:"cipher,omitempty"`
	Password string `mapstructure:"password" validate:"required"`
}

type NotifierConfig struct {
	Name      string `mapstructure:"name,omitempty"`
	Type      string `mapstructure:"type" validate:"required,oneof=webhook telegram email"`
	OnSuccess bool   `mapstructure:"onSuccess"`
	OnFailure bool   `mapstructure:"onFailure"`

	URL     string            `mapstructure:"url,omitempty" validate:"required_if=Type webhook"`
	Method  string            `mapstructure:"method,omitempty"`
	Headers map[string]string `mapstructure:"headers,omitempty"`

	BotToken string `mapstructure:"botToken,omitempty" validate:"required_if=Type telegram"`
	ChatID   string `mapstructure:"chatId,omitempty" validate:"required_if=Type telegram"`

	SMTPHost string   `mapstructure:"smtpHost,omitempty" validate:"required_if=Type email"`
	SMTPPort int      `mapstructure:"smtpPort,omitempty" validate:"required_if=Type email"`
	SMTPUser string   `mapstructure:"smtpUser,omitempty"`
	SMTPPass string   `mapstructure:"smtpPass,omitempty"`
	From     string   `mapstructure:"from,omitempty" validate:"required_if=Type email"`
	To       []string `mapstructure:"to,omitempty" validate:"required_if=Type email,dive,email"`
	UseTLS   bool     `mapstructure:"useTLS,omitempty"`
}

var (
	validateOnce     sync.Once
	validateInstance *validator.Validate
)

func defaultSearchPaths() []string {
	paths := []string{"."}
	if homeDir, err := os.UserHomeDir(); err == nil {
		paths = append(paths,
			filepath.Join(homeDir, ".config", "snapr"),
			homeDir,
		)
	}
	paths = append(paths, "/etc/snapr")
	return paths
}

func getValidator() *validator.Validate {
	validateOnce.Do(func() {
		validateInstance = validator.New()
		validateInstance.RegisterStructValidation(validateJob, JobConfig{})
	})
	return validateInstance
}

func validateJob(sl validator.StructLevel) {
	job := sl.Current().Interface().(JobConfig)

	seen := make(map[string]struct{}, len(job.Storages))
	for i, s := range job.Storages {
		if s.Name == "" {
			continue // covered by `validate:"required"` on Name
		}
		if _, dup := seen[s.Name]; dup {
			sl.ReportError(s.Name, fmt.Sprintf("storages[%d].name", i), "Name", "unique", s.Name)
			continue
		}
		seen[s.Name] = struct{}{}
	}

	if job.Split != nil && job.Split.ChunkSize != "" {
		if _, err := utils.ParseSize(job.Split.ChunkSize); err != nil {
			sl.ReportError(job.Split.ChunkSize, "split.chunkSize", "ChunkSize", "size", err.Error())
		}
	}

	for i, s := range job.Storages {
		if s.DownloadMode == "signed" && s.Type != "s3" {
			sl.ReportError(s.DownloadMode, fmt.Sprintf("storages[%d].downloadMode", i), "DownloadMode", "signed_only_s3", s.Type)
		}
	}
}

// resolveEnvRefs replaces "env:VAR" string values with os.Getenv("VAR") to keep secrets out of YAML.
func resolveEnvRefs(v reflect.Value) error {
	switch v.Kind() {
	case reflect.Pointer, reflect.Interface:
		if v.IsNil() {
			return nil
		}
		return resolveEnvRefs(v.Elem())
	case reflect.Struct:
		for i := 0; i < v.NumField(); i++ {
			if !v.Field(i).CanSet() {
				continue
			}
			if err := resolveEnvRefs(v.Field(i)); err != nil {
				return err
			}
		}
	case reflect.Slice, reflect.Array:
		for i := 0; i < v.Len(); i++ {
			if err := resolveEnvRefs(v.Index(i)); err != nil {
				return err
			}
		}
	case reflect.Map:
		for _, key := range v.MapKeys() {
			val := v.MapIndex(key)
			if val.Kind() != reflect.String {
				continue
			}
			resolved, err := resolveEnvString(val.String())
			if err != nil {
				return err
			}
			if resolved != val.String() {
				v.SetMapIndex(key, reflect.ValueOf(resolved))
			}
		}
	case reflect.String:
		resolved, err := resolveEnvString(v.String())
		if err != nil {
			return err
		}
		if resolved != v.String() {
			v.SetString(resolved)
		}
	}
	return nil
}

func resolveEnvString(s string) (string, error) {
	if !strings.HasPrefix(s, "env:") {
		return s, nil
	}
	name := strings.TrimPrefix(s, "env:")
	if name == "" {
		return "", fmt.Errorf("empty env var name in %q", s)
	}
	val, ok := os.LookupEnv(name)
	if !ok {
		return "", fmt.Errorf("env var %q is not set", name)
	}
	return val, nil
}

func Load(configFilePath string) (*Config, error) {
	logger := log.With().Str("component", "config").Logger()

	logger.Info().Msg("Loading configuration")

	v := viper.New()

	configFile := configFilePath
	if configFile == "" {
		configFile = os.Getenv("SNAPR_CONFIG_FILE")
	}

	if configFile != "" {
		source := "command-line flag"
		if configFilePath == "" {
			source = "SNAPR_CONFIG_FILE environment variable"
		}

		logger.Info().
			Str("config_file", configFile).
			Str("source", source).
			Msg("Using specified config file")

		if _, err := os.Stat(configFile); errors.Is(err, fs.ErrNotExist) { //nolint:gosec // configFile from user-controlled SNAPR_CONFIG_FILE env, expected
			logger.Error().
				Str("config_file", configFile).
				Str("source", source).
				Msg("Configuration file does not exist")
			return nil, fmt.Errorf("configuration file not found: %s", configFile)
		}

		v.SetConfigFile(configFile)
	} else {
		v.SetConfigName("snapr")

		searchPaths := defaultSearchPaths()
		for _, path := range searchPaths {
			v.AddConfigPath(path)
		}

		logger.Info().Strs("search_paths", searchPaths).Msg("Searching for configuration file in default paths")
	}

	v.SetEnvPrefix("SNAPR")
	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	v.SetDefault("server.enabled", true)
	v.SetDefault("server.address", "0.0.0.0:8080")
	v.SetDefault("server.defaultLanguage", "en")
	v.SetDefault("server.logLimits.jobLogs", 10000)
	v.SetDefault("server.logLimits.systemLogs", 10000)
	v.SetDefault("server.permissions.allowBackupDownload", true)
	v.SetDefault("server.permissions.allowManualRun", true)
	v.SetDefault("server.permissions.showConfig", true)
	v.SetDefault("server.ui.enabled", true)

	v.SetDefault("logs.path", "./logs")
	v.SetDefault("logs.system", true)
	v.SetDefault("logs.perJob", true)
	v.SetDefault("logs.maxSizeMB", 100)
	v.SetDefault("logs.maxBackups", 7)
	v.SetDefault("logs.maxAgeDays", 30)
	v.SetDefault("logs.compress", true)

	if err := v.ReadInConfig(); err != nil {
		var notFoundErr viper.ConfigFileNotFoundError
		if errors.As(err, &notFoundErr) {
			if configFile != "" {
				logger.Error().Str("config_file", configFile).Msg("Configuration file specified in SNAPR_CONFIG_FILE not found")
				return nil, fmt.Errorf("configuration file not found: %s", configFile)
			}
			logger.Error().
				Strs("search_paths", defaultSearchPaths()).
				Msg("Configuration file not found in any search path")
			return nil, fmt.Errorf("configuration file not found")
		}
		logger.Error().Err(err).Msg("Error reading configuration file")
		return nil, fmt.Errorf("error reading configuration: %w", err)
	}

	usedConfigFile := v.ConfigFileUsed()

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		logger.Error().Err(err).
			Str("config_file", usedConfigFile).
			Msg("Error parsing configuration file")
		return nil, fmt.Errorf("error parsing configuration: %w", err)
	}

	if err := resolveEnvRefs(reflect.ValueOf(&cfg).Elem()); err != nil {
		logger.Error().Err(err).Msg("Error resolving env references in configuration")
		return nil, err
	}

	if err := getValidator().Struct(&cfg); err != nil {
		formattedErr := formatValidationError(err)
		logger.Error().Err(formattedErr).
			Str("config_file", usedConfigFile).
			Msg("Configuration validation failed")
		return nil, formattedErr
	}

	loadEvent := logger.Info().
		Str("path", usedConfigFile).
		Bool("server_enabled", cfg.Server.Enabled).
		Int("jobs_count", len(cfg.Jobs))
	if cfg.Server.Enabled {
		loadEvent = loadEvent.
			Str("server_address", cfg.Server.Address).
			Bool("ui_enabled", cfg.Server.UI.Enabled)
	}
	loadEvent.Msg("Configuration loaded successfully")

	return &cfg, nil
}

func formatValidationError(err error) error {
	logger := log.With().Str("component", "config").Logger()

	var validationErrors validator.ValidationErrors
	if errors.As(err, &validationErrors) {
		var messages []string
		var errorDetails []map[string]interface{}

		for _, e := range validationErrors {
			fieldError := formatFieldError(e)
			messages = append(messages, fieldError)

			errorDetails = append(errorDetails, map[string]interface{}{
				"field":   strings.ToLower(e.Field()),
				"tag":     e.Tag(),
				"value":   e.Value(),
				"param":   e.Param(),
				"message": fieldError,
			})
		}

		logger.Error().
			Interface("validation_errors", errorDetails).
			Int("error_count", len(messages)).
			Msg("Detailed validation errors")

		return fmt.Errorf("validation failed:\n  %s", strings.Join(messages, "\n  "))
	}

	logger.Error().Err(err).Msg("Unexpected validation error")
	return fmt.Errorf("validation failed: %w", err)
}

func formatFieldError(e validator.FieldError) string {
	field := strings.ToLower(e.Field())

	switch e.Tag() {
	case "required":
		return fmt.Sprintf("%s is required", field)
	case "required_if":
		return fmt.Sprintf("%s is required when type is %s", field, e.Param())
	case "required_without_all":
		return fmt.Sprintf("%s is required when %s is not provided", field, e.Param())
	case "oneof":
		return fmt.Sprintf("%s must be one of: %s", field, e.Param())
	case "min":
		if e.Kind().String() == "slice" {
			return fmt.Sprintf("%s must have at least %s items", field, e.Param())
		}
		return fmt.Sprintf("%s must be at least %s", field, e.Param())
	case "max":
		return fmt.Sprintf("%s must not exceed %s", field, e.Param())
	case "unique":
		return fmt.Sprintf("%s must be unique within the job", field)
	default:
		return fmt.Sprintf("%s failed validation: %s", field, e.Tag())
	}
}
