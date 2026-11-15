package config_test

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/checkship-deploy/checkship/cmd/config"
	"github.com/checkship-deploy/checkship/pkg/assert"
	"github.com/checkship-deploy/checkship/pkg/bus"
	"github.com/checkship-deploy/checkship/pkg/log"
	"github.com/checkship-deploy/checkship/pkg/must"
	"github.com/checkship-deploy/checkship/pkg/validate"
	"github.com/checkship-deploy/checkship/pkg/validate/numbers"
)

const (
	deployJob          = "deployment.command.deploy"
	cleanupAppJob      = "deployment.command.cleanup_app"
	deleteAppJob       = "deployment.command.delete_app"
	configureTargetJob = "deployment.command.configure_target"
	cleanupTargetJob   = "deployment.command.cleanup_target"
	deleteTargetJob    = "deployment.command.delete_target"
	backupJob          = "deployment.command.backup_volumes"
	restoreJob         = "deployment.command.restore_volumes"
	storeBackupJob     = "deployment.command.store_backup"
)

// Sorted, as bus.Marshallable.Keys() returns them.
var knownJobs = []string{backupJob, cleanupAppJob, cleanupTargetJob, configureTargetJob, deleteAppJob, deleteTargetJob, deployJob, restoreJob, storeBackupJob}

var backupJobs = []string{backupJob, restoreJob, storeBackupJob}

// Every job but deployments, sorted.
var nonDeployJobs = []string{backupJob, cleanupAppJob, cleanupTargetJob, configureTargetJob, deleteAppJob, deleteTargetJob, restoreJob, storeBackupJob}

var otherJobs = []string{cleanupAppJob, cleanupTargetJob, configureTargetJob, deleteAppJob, deleteTargetJob}

func Test_Runners(t *testing.T) {
	t.Run("should default to a deployment runner, a volume copy runner and a runner for every other job", func(t *testing.T) {
		conf, logger, err := load(t, "", nil)

		assert.Nil(t, err)
		assert.HasLength(t, 0, logger.warnings())

		runners := definitions(t, conf)

		assert.DeepEqual(t, []bus.RunnerDefinition{
			{PollInterval: 4 * time.Second, Workers: 4, Messages: []string{deployJob}},
			{PollInterval: 4 * time.Second, Workers: 1, Messages: backupJobs},
			{PollInterval: 4 * time.Second, Workers: 2, Messages: otherJobs},
		}, runners)
	})

	t.Run("should write the default runners as a list in a new configuration file", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "conf.yml")

		_, _, err := loadFile(t, path, map[string]string{"DATA_PATH": dir})
		assert.Nil(t, err)

		content := string(must.Panic(os.ReadFile(path)))
		assert.Match(t, `runners:\n    - poll_interval: 4s\n      count: 4\n      jobs:\n        - deployment.command.deploy\n    - poll_interval: 4s\n      count: 1\n      jobs:\n        - deployment.command.backup_volumes\n        - deployment.command.restore_volumes\n        - deployment.command.store_backup\n    - poll_interval: 4s\n      count: 2\n`, content)

		// And read it back
		conf, logger, err := loadFile(t, path, map[string]string{"DATA_PATH": dir})
		assert.Nil(t, err)
		assert.HasLength(t, 0, logger.warnings())
		assert.HasLength(t, 3, definitions(t, conf))
	})

	t.Run("should read runners from the configuration file", func(t *testing.T) {
		conf, logger, err := load(t, `
runners:
  - poll_interval: 1s
    count: 8
    jobs:
      - deployment.command.deploy
  - poll_interval: 2s
    count: 1
    jobs:
      - deployment.command.configure_target
  - poll_interval: 10s
    count: 3
`, nil)

		assert.Nil(t, err)
		assert.HasLength(t, 0, logger.warnings())
		assert.DeepEqual(t, []bus.RunnerDefinition{
			{PollInterval: time.Second, Workers: 8, Messages: []string{deployJob}},
			{PollInterval: 2 * time.Second, Workers: 1, Messages: []string{configureTargetJob}},
			{PollInterval: 10 * time.Second, Workers: 3, Messages: []string{backupJob, cleanupAppJob, cleanupTargetJob, deleteAppJob, deleteTargetJob, restoreJob, storeBackupJob}},
		}, definitions(t, conf))
	})

	t.Run("should read the old configuration file format and warn about it", func(t *testing.T) {
		conf, logger, err := load(t, `
runners:
  poll_interval: 10s
  deployment: 5
  cleanup: 3
`, nil)

		assert.Nil(t, err)
		assert.HasLength(t, 1, logger.warnings())
		assert.Match(t, "old format", logger.warnings()[0])
		assert.DeepEqual(t, []bus.RunnerDefinition{
			{PollInterval: 10 * time.Second, Workers: 5, Messages: []string{deployJob}},
			{PollInterval: 10 * time.Second, Workers: 1, Messages: backupJobs},
			{PollInterval: 10 * time.Second, Workers: 3, Messages: otherJobs},
		}, definitions(t, conf))
	})

	t.Run("should report a list error when the runners section is neither format", func(t *testing.T) {
		_, _, err := load(t, `
runners: 42
`, nil)

		assert.NotNil(t, err)
	})

	t.Run("should read RUNNERS_CONFIGURATION over the configuration file", func(t *testing.T) {
		conf, _, err := load(t, `
runners:
  - poll_interval: 1s
    count: 1
`, map[string]string{
			"RUNNERS_CONFIGURATION": " 4s;2;deployment.command.deploy, deployment.command.cleanup_app | 5s;3; ",
		})

		assert.Nil(t, err)
		assert.DeepEqual(t, []bus.RunnerDefinition{
			{PollInterval: 4 * time.Second, Workers: 2, Messages: []string{deployJob, cleanupAppJob}},
			{PollInterval: 5 * time.Second, Workers: 3, Messages: []string{backupJob, cleanupTargetJob, configureTargetJob, deleteAppJob, deleteTargetJob, restoreJob, storeBackupJob}},
		}, definitions(t, conf))
	})

	t.Run("should ignore an empty RUNNERS_CONFIGURATION", func(t *testing.T) {
		conf, _, err := load(t, "", map[string]string{"RUNNERS_CONFIGURATION": "  "})

		assert.Nil(t, err)
		assert.HasLength(t, 3, definitions(t, conf))
	})

	t.Run("should say which runner of RUNNERS_CONFIGURATION could not be parsed", func(t *testing.T) {
		_, _, err := load(t, "", map[string]string{"RUNNERS_CONFIGURATION": "4s;2;deployment.command.deploy|4s;x;"})

		assert.NotNil(t, err)
		assert.Match(t, `runner 1 count "x" is not a number`, err.Error())

		_, _, err = load(t, "", map[string]string{"RUNNERS_CONFIGURATION": "4s;2"})

		assert.NotNil(t, err)
		assert.Match(t, "runner 0 has 2 parts, expected 3", err.Error())
	})

	t.Run("should apply the deprecated environment variables and warn about them", func(t *testing.T) {
		conf, logger, err := load(t, "", map[string]string{
			"RUNNERS_DEPLOYMENT_COUNT": "8",
		})

		assert.Nil(t, err)
		assert.HasLength(t, 1, logger.warnings())
		assert.Match(t, "deprecated, use RUNNERS_CONFIGURATION", logger.warnings()[0])
		assert.DeepEqual(t, []bus.RunnerDefinition{
			{PollInterval: 4 * time.Second, Workers: 8, Messages: []string{deployJob}},
			{PollInterval: 4 * time.Second, Workers: 1, Messages: backupJobs},
			{PollInterval: 4 * time.Second, Workers: 2, Messages: otherJobs},
		}, definitions(t, conf))
	})

	t.Run("should apply the deprecated environment variables over the old file format", func(t *testing.T) {
		conf, logger, err := load(t, `
runners:
  poll_interval: 10s
  deployment: 5
  cleanup: 3
`, map[string]string{
			"RUNNERS_POLL_INTERVAL": "2s",
			"RUNNERS_CLEANUP_COUNT": "6",
		})

		assert.Nil(t, err)
		assert.HasLength(t, 2, logger.warnings())
		assert.DeepEqual(t, []bus.RunnerDefinition{
			{PollInterval: 2 * time.Second, Workers: 5, Messages: []string{deployJob}},
			{PollInterval: 2 * time.Second, Workers: 1, Messages: backupJobs},
			{PollInterval: 2 * time.Second, Workers: 6, Messages: otherJobs},
		}, definitions(t, conf))
	})

	t.Run("should ignore the deprecated environment variables when RUNNERS_CONFIGURATION is set", func(t *testing.T) {
		conf, logger, err := load(t, "", map[string]string{
			"RUNNERS_CONFIGURATION":    "1s;1;",
			"RUNNERS_DEPLOYMENT_COUNT": "8",
		})

		assert.Nil(t, err)
		assert.HasLength(t, 1, logger.warnings())
		assert.Match(t, "ignored because RUNNERS_CONFIGURATION is set", logger.warnings()[0])
		assert.DeepEqual(t, []bus.RunnerDefinition{
			{PollInterval: time.Second, Workers: 1, Messages: knownJobs},
		}, definitions(t, conf))
	})

	t.Run("should fail on a deprecated count that is not a number", func(t *testing.T) {
		_, _, err := load(t, "", map[string]string{"RUNNERS_CLEANUP_COUNT": "two"})

		assert.NotNil(t, err)
		assert.Match(t, `RUNNERS_CLEANUP_COUNT: "two" is not a number`, err.Error())
	})

	t.Run("should validate every runner", func(t *testing.T) {
		_, _, err := load(t, "", map[string]string{
			"RUNNERS_CONFIGURATION": "0s;0;deployment.command.deploy|1s;65;|1h;-1;deployment.command.deploy,deployment.command.cleanup_app",
		})

		assert.ValidationError(t, validate.FieldErrors{
			"runners.0.poll_interval": config.ErrPollIntervalTooShort,
			"runners.0.count":         numbers.ErrMin,
			"runners.1.count":         numbers.ErrMax,
			"runners.1.jobs":          config.ErrRunnerJobsRequired,
			"runners.2.count":         numbers.ErrMin,
			"runners.2.jobs":          config.ErrJobInSeveralRunners,
		}, err)
	})

	t.Run("should require at least one runner", func(t *testing.T) {
		_, _, err := load(t, "runners: []\n", nil)

		assert.ValidationError(t, validate.FieldErrors{
			"runners": config.ErrNoRunners,
		}, err)
	})

	t.Run("should fail on unknown job names", func(t *testing.T) {
		conf, _, err := load(t, "", map[string]string{
			"RUNNERS_CONFIGURATION": "4s;2;deployment.command.deploy,deployment.command.nope|4s;2;",
		})
		assert.Nil(t, err)

		_, err = conf.RunnersDefinitions(knownJobs, must.Panic(log.NewLogger()))

		assert.NotNil(t, err)
		assert.Match(t, "unknown job names in the runners configuration: deployment.command.nope, known jobs are deployment.command.backup_volumes, deployment.command.cleanup_app, ", err.Error())
	})

	t.Run("should fail when a job is handled by no runner", func(t *testing.T) {
		conf, _, err := load(t, "", map[string]string{
			"RUNNERS_CONFIGURATION": "4s;2;deployment.command.deploy",
		})
		assert.Nil(t, err)

		_, err = conf.RunnersDefinitions(knownJobs, must.Panic(log.NewLogger()))

		assert.NotNil(t, err)
		assert.Equal(t, "no runner handles these jobs: "+strings.Join(nonDeployJobs, ", ")+", list them in a runner or leave the jobs of the last runner empty", err.Error())
	})

	t.Run("should drop a last runner left without jobs", func(t *testing.T) {
		conf, _, err := load(t, "", map[string]string{
			"RUNNERS_CONFIGURATION": fmt.Sprintf("4s;2;%s|4s;1;%s|4s;1;", deployJob, strings.Join(nonDeployJobs, ",")),
		})
		assert.Nil(t, err)

		logger := &testLogger{ConfigurableLogger: must.Panic(log.NewLogger())}
		runners, err := conf.RunnersDefinitions(knownJobs, logger)

		assert.Nil(t, err)
		assert.HasLength(t, 2, runners)
		assert.HasLength(t, 1, logger.warnings())
		assert.True(t, slices.Equal(nonDeployJobs, runners[1].Messages))
	})
}

type testLogger struct {
	log.ConfigurableLogger

	mu   sync.Mutex
	warn []string
}

func (l *testLogger) Warn(args ...any) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.warn = append(l.warn, fmt.Sprint(args...))
}

func (l *testLogger) Warnw(msg string, _ ...any) { l.Warn(msg) }

func (l *testLogger) warnings() []string {
	l.mu.Lock()
	defer l.mu.Unlock()
	return slices.Clone(l.warn)
}

// Loads a configuration from the given YAML content (none if empty) and environment.
// Runner variables not given are set to empty, which means unset.
func load(t *testing.T, yaml string, env map[string]string) (config.Configuration, *testLogger, error) {
	dir := t.TempDir()
	path := filepath.Join(dir, "conf.yml")

	if yaml != "" {
		assert.Nil(t, os.WriteFile(path, []byte(yaml), 0o600))
	}

	if env == nil {
		env = map[string]string{}
	}

	env["DATA_PATH"] = dir

	return loadFile(t, path, env)
}

func loadFile(t *testing.T, path string, env map[string]string) (config.Configuration, *testLogger, error) {
	for _, key := range []string{"RUNNERS_CONFIGURATION", "RUNNERS_POLL_INTERVAL", "RUNNERS_DEPLOYMENT_COUNT", "RUNNERS_CLEANUP_COUNT"} {
		t.Setenv(key, "")
	}

	for key, value := range env {
		t.Setenv(key, value)
	}

	logger := &testLogger{ConfigurableLogger: must.Panic(log.NewLogger())}
	conf := config.Default()
	err := conf.Initialize(logger, path)

	return conf, logger, err
}

func definitions(t *testing.T, conf config.Configuration) []bus.RunnerDefinition {
	t.Helper()

	runners, err := conf.RunnersDefinitions(knownJobs, must.Panic(log.NewLogger()))
	assert.Nil(t, err)

	return runners
}
