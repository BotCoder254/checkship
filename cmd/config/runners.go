package config

import (
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/checkship-deploy/checkship/internal/deployment/app/deploy"
	"github.com/checkship-deploy/checkship/pkg/apperr"
	"github.com/checkship-deploy/checkship/pkg/bus"
	"github.com/checkship-deploy/checkship/pkg/log"
	"github.com/checkship-deploy/checkship/pkg/validate"
	"github.com/checkship-deploy/checkship/pkg/validate/numbers"
	"gopkg.in/yaml.v3"
)

const (
	runnersEnvPartsCount      = 3
	runnersEnvPartSeparator   = ";"
	runnersEnvJobSeparator    = ","
	runnersEnvRunnerSeparator = "|"

	minRunnerPollInterval = 500 * time.Millisecond
	maxRunnerWorkers      = 64
)

var (
	ErrNoRunners             = apperr.New("at_least_one_runner_required")
	ErrPollIntervalTooShort  = apperr.New("poll_interval_below_500ms")
	ErrRunnerJobsRequired    = apperr.New("jobs_required_except_on_last_runner")
	ErrJobInSeveralRunners   = apperr.New("job_already_handled_by_another_runner")
	errRunnersEnvParseFailed = errors.New("could not parse RUNNERS_CONFIGURATION, expected <poll_interval>;<count>;<job,job> with runners separated by |")
)

type (
	// One background runner: it polls the jobs listed in Jobs every PollInterval and
	// runs up to Count of them at once. The last runner may leave Jobs empty to take
	// every job the other runners don't list.
	runnerConfiguration struct {
		PollInterval string   `yaml:"poll_interval"`
		Count        int      `yaml:"count"`
		Jobs         []string `yaml:"jobs,omitempty"`

		pollInterval time.Duration
	}

	// List of runners, written as a YAML sequence. It still reads the format used
	// before runners were configurable (see legacyRunnersFile).
	runnersConfiguration struct {
		list    []runnerConfiguration
		fromEnv bool               // Set from RUNNERS_CONFIGURATION, which takes precedence over legacy variables
		legacy  *legacyRunnersFile // Set when the YAML file uses the old format
	}

	// Runners section of configuration files written before runners were configurable.
	legacyRunnersFile struct {
		PollInterval string `yaml:"poll_interval"`
		Deployment   int    `yaml:"deployment"`
		Cleanup      int    `yaml:"cleanup"`
	}

	// Environment variables used before RUNNERS_CONFIGURATION. Strings so an unset
	// variable can be told apart from a zero.
	legacyRunnersEnvironment struct {
		PollInterval string `env:"RUNNERS_POLL_INTERVAL"`
		Deployment   string `env:"RUNNERS_DEPLOYMENT_COUNT"`
		Cleanup      string `env:"RUNNERS_CLEANUP_COUNT"`
	}
)

// Two runners: one for deployments, one for every other job.
func defaultRunners(pollInterval string, deploymentCount, otherCount int) []runnerConfiguration {
	return []runnerConfiguration{
		{
			PollInterval: pollInterval,
			Count:        deploymentCount,
			Jobs:         []string{deploy.Command{}.Name_()},
		},
		{
			PollInterval: pollInterval,
			Count:        otherCount,
		},
	}
}

func (r runnersConfiguration) MarshalYAML() (any, error) { return r.list, nil }

func (r *runnersConfiguration) UnmarshalYAML(value *yaml.Node) error {
	var list []runnerConfiguration

	listErr := value.Decode(&list)

	if listErr == nil {
		r.list = list
		r.legacy = nil
		return nil
	}

	var legacy legacyRunnersFile

	// Not the old format either, the list error is the one worth showing
	if value.Kind != yaml.MappingNode || value.Decode(&legacy) != nil {
		return listErr
	}

	r.legacy = &legacy
	r.list = defaultRunners(legacy.PollInterval, legacy.Deployment, legacy.Cleanup)

	return nil
}

// Parses RUNNERS_CONFIGURATION: <poll_interval>;<count>;<job,job> with runners
// separated by |, for example "4s;4;deployment.command.deploy|4s;2;".
// An empty value is treated as unset.
func (r *runnersConfiguration) UnmarshalEnvironmentValue(data string) error {
	data = strings.TrimSpace(data)

	if data == "" {
		return nil
	}

	parts := strings.Split(data, runnersEnvRunnerSeparator)
	list := make([]runnerConfiguration, len(parts))

	for i, part := range parts {
		fields := strings.Split(part, runnersEnvPartSeparator)

		if len(fields) != runnersEnvPartsCount {
			return fmt.Errorf("%w: runner %d has %d parts, expected %d", errRunnersEnvParseFailed, i, len(fields), runnersEnvPartsCount)
		}

		count, err := strconv.Atoi(strings.TrimSpace(fields[1]))

		if err != nil {
			return fmt.Errorf("%w: runner %d count %q is not a number", errRunnersEnvParseFailed, i, fields[1])
		}

		runner := runnerConfiguration{
			PollInterval: strings.TrimSpace(fields[0]),
			Count:        count,
		}

		for _, job := range strings.Split(fields[2], runnersEnvJobSeparator) {
			if job = strings.TrimSpace(job); job != "" {
				runner.Jobs = append(runner.Jobs, job)
			}
		}

		list[i] = runner
	}

	r.list = list
	r.fromEnv = true
	r.legacy = nil

	return nil
}

func (e legacyRunnersEnvironment) isSet() bool {
	return e.PollInterval != "" || e.Deployment != "" || e.Cleanup != ""
}

// Applies the deprecated RUNNERS_* variables and returns the warnings to log.
// They replace the runners with the default pair, starting from the values of an
// old-format file if there is one, unless RUNNERS_CONFIGURATION is set.
func (r *runnersConfiguration) applyLegacy(env legacyRunnersEnvironment) ([]string, error) {
	var warnings []string

	if r.legacy != nil {
		warnings = append(warnings, "the runners section of the configuration file uses the old format (poll_interval, deployment, cleanup), it still works but is deprecated, see the runners section of the configuration documentation for the list format")
	}

	if !env.isSet() {
		return warnings, nil
	}

	if r.fromEnv {
		return append(warnings, "RUNNERS_POLL_INTERVAL, RUNNERS_DEPLOYMENT_COUNT and RUNNERS_CLEANUP_COUNT are ignored because RUNNERS_CONFIGURATION is set"), nil
	}

	base := legacyRunnersFile{
		PollInterval: defaultRunnersPollInterval,
		Deployment:   defaultRunnersDeploymentCount,
		Cleanup:      defaultRunnersOtherCount,
	}

	if r.legacy != nil {
		base = *r.legacy
	}

	if env.PollInterval != "" {
		base.PollInterval = env.PollInterval
	}

	for _, v := range []struct {
		name   string
		value  string
		target *int
	}{
		{"RUNNERS_DEPLOYMENT_COUNT", env.Deployment, &base.Deployment},
		{"RUNNERS_CLEANUP_COUNT", env.Cleanup, &base.Cleanup},
	} {
		if v.value == "" {
			continue
		}

		count, err := strconv.Atoi(strings.TrimSpace(v.value))

		if err != nil {
			return warnings, fmt.Errorf("%s: %q is not a number", v.name, v.value)
		}

		*v.target = count
	}

	r.list = defaultRunners(base.PollInterval, base.Deployment, base.Cleanup)

	return append(warnings, "RUNNERS_POLL_INTERVAL, RUNNERS_DEPLOYMENT_COUNT and RUNNERS_CLEANUP_COUNT are deprecated, use RUNNERS_CONFIGURATION instead"), nil
}

// Validates every runner and parses their poll intervals.
func (r *runnersConfiguration) validate() error {
	if len(r.list) == 0 {
		return ErrNoRunners
	}

	var (
		lastIdx = len(r.list) - 1
		seenIn  = make(map[string]int)
	)

	return validate.Array(r.list, func(runner runnerConfiguration, idx int) error {
		return validate.Struct(validate.Of{
			"poll_interval": validate.Value(runner.PollInterval, &r.list[idx].pollInterval, parsePollInterval),
			"count":         validate.Field(runner.Count, numbers.Min(1), numbers.Max(maxRunnerWorkers)),
			"jobs": validate.Field(runner.Jobs, func(jobs []string) error {
				if len(jobs) == 0 && idx != lastIdx {
					return ErrRunnerJobsRequired
				}

				for _, job := range jobs {
					if _, seen := seenIn[job]; seen {
						return ErrJobInSeveralRunners
					}

					seenIn[job] = idx
				}

				return nil
			}),
		})
	})
}

func parsePollInterval(value string) (time.Duration, error) {
	interval, err := time.ParseDuration(value)

	if err != nil {
		return 0, err
	}

	if interval < minRunnerPollInterval {
		return 0, ErrPollIntervalTooShort
	}

	return interval, nil
}

// Builds the runner definitions given the names of every job type the application
// registered. Fails on unknown names and on job types no runner handles. A last runner
// without jobs takes the remaining ones and is dropped if nothing remains.
func (r *runnersConfiguration) definitions(known []string, logger log.Logger) ([]bus.RunnerDefinition, error) {
	var (
		definitions = make([]bus.RunnerDefinition, 0, len(r.list))
		unhandled   = slices.Clone(known)
		unknown     []string
	)

	for _, runner := range r.list {
		for _, job := range runner.Jobs {
			if !slices.Contains(known, job) {
				unknown = append(unknown, job)
				continue
			}

			unhandled = slices.DeleteFunc(unhandled, func(name string) bool { return name == job })
		}
	}

	if len(unknown) > 0 {
		return nil, fmt.Errorf("unknown job names in the runners configuration: %s, known jobs are %s",
			strings.Join(unknown, ", "), strings.Join(known, ", "))
	}

	for idx, runner := range r.list {
		jobs := runner.Jobs

		// Validation makes sure only the last runner can have no jobs
		if len(jobs) == 0 {
			if len(unhandled) == 0 {
				logger.Warnw("the last runner has no job left to handle, every job is already listed by another runner, it will not be started",
					"runner", idx)
				continue
			}

			jobs, unhandled = unhandled, nil
		}

		definitions = append(definitions, bus.RunnerDefinition{
			PollInterval: runner.pollInterval,
			Workers:      runner.Count,
			Messages:     slices.Clone(jobs),
		})
	}

	if len(unhandled) > 0 {
		return nil, fmt.Errorf("no runner handles these jobs: %s, list them in a runner or leave the jobs of the last runner empty",
			strings.Join(unhandled, ", "))
	}

	return definitions, nil
}
