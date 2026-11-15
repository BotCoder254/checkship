package config

import (
	"errors"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"text/template"

	"github.com/checkship-deploy/checkship/cmd/serve"
	"github.com/checkship-deploy/checkship/cmd/version"
	"github.com/checkship-deploy/checkship/internal/deployment/domain"
	"github.com/checkship-deploy/checkship/pkg/bus"
	"github.com/checkship-deploy/checkship/pkg/bytesize"
	"github.com/checkship-deploy/checkship/pkg/config"
	"github.com/checkship-deploy/checkship/pkg/crypto"
	"github.com/checkship-deploy/checkship/pkg/id"
	"github.com/checkship-deploy/checkship/pkg/log"
	"github.com/checkship-deploy/checkship/pkg/monad"
	"github.com/checkship-deploy/checkship/pkg/must"
	"github.com/checkship-deploy/checkship/pkg/ostools"
	"github.com/checkship-deploy/checkship/pkg/validate"
)

var (
	userConfigDir                = must.Panic(os.UserConfigDir())
	generatedSecretKey           = must.Panic(crypto.RandomKey[string](64))
	defaultDataDirectory         = filepath.Join(userConfigDir, "checkship")
	DefaultConfigPath            = filepath.Join(defaultDataDirectory, defaultConfigFilename) // Default configuration path
	errBackupHelperImageRequired = errors.New("required")
	noticeSecretKeyGenerated     = `a default secret key has been generated. If you want to override it, you can set the HTTP_SECRET environment variable.`
)

const (
	databaseConnectionString      = "checkship.db?_journal=WAL&_timeout=5000&_foreign_keys=yes&_txlock=immediate"
	defaultConfigFilename         = "conf.yml"
	defaultPort                   = 8080
	defaultHost                   = ""
	defaultRunnersPollInterval    = "4s"
	defaultRunnersDeploymentCount = 4
	defaultRunnersOtherCount      = 2
	defaultRunnersBackupCount     = 1
	defaultBalancerDomain         = "http://docker.localhost"
	defaultDeploymentDirTemplate  = "{{ .Environment }}"
	defaultBackupHelperImage      = "busybox:1.37"
	defaultBackupMaxUploadSize    = "5gb"
	defaultBackupStorageImage     = "versity/versitygw:v1.11.0"
	defaultSourceArchiveMaxSize   = "32mb"
)

type (
	// Configuration used to configure checkship commands.
	Configuration interface {
		serve.Options

		Initialize(log.ConfigurableLogger, string) error // Initialize the configuration by loading it (from config file, env vars, etc.)
	}

	// Configuration builder function used to initialize the configuration object (mostly used in tests).
	ConfigurationBuilder func(*configuration)

	configuration struct {
		Log     logConfiguration
		Data    dataConfiguration
		Source  sourceConfiguration
		Http    httpConfiguration
		Runners runnersConfiguration `env:"RUNNERS_CONFIGURATION"`
		Backups backupsConfiguration
		Private internalConfiguration `yaml:"-"`

		LegacyRunners legacyRunnersEnvironment `yaml:"-"` // Deprecated RUNNERS_* variables

		appExposedUrl         monad.Maybe[domain.Url]
		warnings              []string // Logged once the logger is configured
		deploymentDirTemplate *template.Template
		backupMaxUploadSize   int64
		sourceArchiveMaxSize  int64
		logLevel              log.Level
		logFormat             log.OutputFormat
	}

	logConfiguration struct {
		Level  string `env:"LOG_LEVEL"`
		Format string `env:"LOG_FORMAT"`
	}

	httpConfiguration struct {
		Host   string            `env:"HTTP_HOST" yaml:",omitempty"`
		Port   int               `env:"HTTP_PORT,PORT"`
		Secure monad.Maybe[bool] `env:"HTTP_SECURE" yaml:",omitempty"`
		Secret string            `env:"HTTP_SECRET"`
	}

	// Contains configuration related to where files produced by checkship will be stored.
	dataConfiguration struct {
		Path                  string `env:"DATA_PATH"`
		DeploymentDirTemplate string `env:"DEPLOYMENT_DIR_TEMPLATE" yaml:"deployment_dir_template"`
	}

	// Volume backups. Where archives are copied besides the data folder is set from
	// the dashboard (Backups › Storage).
	backupsConfiguration struct {
		HelperImage   string `env:"BACKUPS_HELPER_IMAGE" yaml:"helper_image"`       // Image that mounts each volume while it is copied
		MaxUploadSize string `env:"BACKUPS_MAX_UPLOAD_SIZE" yaml:"max_upload_size"` // Largest archive accepted by Upload backup
		BeforeCleanup bool   `env:"BACKUPS_BEFORE_CLEANUP" yaml:"before_cleanup"`   // Back up volumes before checkship removes them
		StorageImage  string `env:"BACKUPS_STORAGE_IMAGE" yaml:"storage_image"`     // Image of the built-in S3 storage
	}

	// Deployment sources
	sourceConfiguration struct {
		Archive sourceArchiveConfiguration
	}

	sourceArchiveConfiguration struct {
		MaxSize string `env:"SOURCE_ARCHIVE_MAX_SIZE" yaml:"max_size"` // Largest archive accepted for a deployment
	}

	// internalConfiguration fields not read from the configuration file and use only during specific steps
	internalConfiguration struct {
		Email     string `env:"CHECKSHIP_ADMIN_EMAIL,ADMIN_EMAIL"`
		Password  string `env:"CHECKSHIP_ADMIN_PASSWORD,ADMIN_PASSWORD"`
		ExposedOn string `env:"EXPOSED_ON"` // Container name and default target url (ie. http://checkship@docker.localhost)
	}
)

// Instantiate the default checkship configuration.
func Default(builders ...ConfigurationBuilder) Configuration {
	conf := &configuration{
		Log: logConfiguration{
			Level:  "info",
			Format: "console",
		},
		Data: dataConfiguration{
			Path:                  defaultDataDirectory,
			DeploymentDirTemplate: defaultDeploymentDirTemplate,
		},
		Source: sourceConfiguration{
			Archive: sourceArchiveConfiguration{MaxSize: defaultSourceArchiveMaxSize},
		},
		Http: httpConfiguration{
			Host:   defaultHost,
			Port:   defaultPort,
			Secret: generatedSecretKey,
		},
		Runners: runnersConfiguration{
			list: defaultRunners(defaultRunnersPollInterval, defaultRunnersDeploymentCount, defaultRunnersOtherCount),
		},
		Backups: backupsConfiguration{
			HelperImage:   defaultBackupHelperImage,
			MaxUploadSize: defaultBackupMaxUploadSize,
			BeforeCleanup: true,
			StorageImage:  defaultBackupStorageImage,
		},
	}

	for _, builder := range builders {
		builder(conf)
	}

	if err := conf.PostLoad(); err != nil {
		panic(err) // Should never happen since the default config is managed by us
	}

	return conf
}

func (c *configuration) Initialize(logger log.ConfigurableLogger, path string) error {
	exists, err := config.Load(path, c)

	if err != nil {
		return err
	}

	// Make sure the data path exists
	if err = ostools.MkdirAll(c.Data.Path); err != nil {
		return err
	}

	// Update logger based on loaded configuration
	if err = logger.Configure(c.logFormat, c.logLevel); err != nil {
		return err
	}

	if exists {
		logger.Infow("configuration loaded",
			"path", path)
	} else {
		logger.Infow("configuration not found, saving current configuration",
			"path", path)

		// Save the config file only if it doesn't exist yet (to preserve the secret generated for example)
		if err := config.Save(path, c); err != nil {
			return err
		}
	}

	for _, warning := range c.warnings {
		logger.Warn(warning)
	}

	if c.Http.Secret == generatedSecretKey {
		logger.Info(noticeSecretKeyGenerated)
	}

	return nil
}

func (c *configuration) DataDir() string                           { return c.Data.Path }
func (c *configuration) DeploymentDirTemplate() *template.Template { return c.deploymentDirTemplate }
func (c *configuration) AppExposedUrl() monad.Maybe[domain.Url]    { return c.appExposedUrl }
func (c *configuration) DefaultEmail() string                      { return c.Private.Email }
func (c *configuration) DefaultPassword() string                   { return c.Private.Password }
func (c *configuration) Secret() []byte                            { return []byte(c.Http.Secret) }
func (c *configuration) IsDebug() bool                             { return c.logLevel == log.DebugLevel }
func (c *configuration) BackupHelperImage() string                 { return c.Backups.HelperImage }
func (c *configuration) BackupsMaxUploadSize() int64               { return c.backupMaxUploadSize }
func (c *configuration) BuiltinStorageImage() string               { return c.Backups.StorageImage }
func (c *configuration) BackupBeforeCleanup() bool                 { return c.Backups.BeforeCleanup }
func (c *configuration) CheckshipVersion() string                  { return version.Current() }
func (c *configuration) MaxDeploymentArchiveFileSize() int64       { return c.sourceArchiveMaxSize }

// Builds the background runners from the configuration, given the names of every job
// type registered by the application (bus.Marshallable.Keys()).
func (c *configuration) RunnersDefinitions(jobNames []string, logger log.Logger) ([]bus.RunnerDefinition, error) {
	return c.Runners.definitions(jobNames, logger)
}

func (c *configuration) IsSecure() bool {
	// If secure has been explicitly isSet, returns it
	if secure, isSet := c.Http.Secure.TryGet(); isSet {
		return secure
	}

	if defaultTargetUrl, isSet := c.appExposedUrl.TryGet(); isSet {
		return defaultTargetUrl.UseSSL()
	}

	return false
}

// Gets the connection string to be used.
func (c *configuration) ConnectionString() string {
	return "file:" + path.Join(c.Data.Path, databaseConnectionString)
}

// Returns the address to bind the HTTP server to.
func (c *configuration) ListenAddress() string {
	return c.Http.Host + ":" + strconv.Itoa(c.Http.Port)
}

func (c *configuration) PostLoad() error {
	warnings, err := c.Runners.applyLegacy(c.LegacyRunners)

	if err != nil {
		return err
	}

	c.warnings = warnings

	return validate.Struct(validate.Of{
		"log.level":                    validate.Value(c.Log.Level, &c.logLevel, log.ParseLevel),
		"log.format":                   validate.Value(c.Log.Format, &c.logFormat, log.ParseFormat),
		"source.archive.max_size":      validate.Value(c.Source.Archive.MaxSize, &c.sourceArchiveMaxSize, bytesize.Parse),
		"data.deployment_dir_template": validate.Value(c.Data.DeploymentDirTemplate, &c.deploymentDirTemplate, template.New("").Parse),
		"runners":                      c.Runners.validate(),
		"backups.helper_image":         validate.If(strings.TrimSpace(c.Backups.HelperImage) == "", func() error { return errBackupHelperImageRequired }),
		"backups.max_upload_size":      validate.Value(c.Backups.MaxUploadSize, &c.backupMaxUploadSize, bytesize.Parse),
		"exposed_as": validate.If(c.Private.ExposedOn != "", func() error {
			url, err := domain.UrlFrom(c.Private.ExposedOn)

			if err != nil {
				return err
			}

			c.appExposedUrl.Set(url)

			return nil
		}),
	})
}

// Configuration builder used to set some tests sensible defaults.
// Generates a random data directory path to avoid conflicts with other tests.
func WithTestDefaults() ConfigurationBuilder {
	return func(c *configuration) {
		c.Data.Path = "__testdata_" + id.New[string]()
	}
}
