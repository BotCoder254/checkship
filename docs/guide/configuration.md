# Configuration

**checkship** can be configured from a **yaml** file and/or **environment variables**. Environment variables will always take precedence over the configuration file.

When configured using a file, you can provide it with the `-c <your_file.yml>` or `--config <your_file.yml>`.

If the configuration file does not exist, **it will be created** with the initial configuration.

::: info
Environment variables can also be defined in a `.env` or `.env.local` file in the working directory when launching checkship.
:::

## Reference

| yaml path / env name(s)                                 | Description                                                                                                                                                                                                                                                 | Default value                         |
| ------------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------- |
| log.level<br>LOG_LEVEL                                  | Log level to use (info, warn or error)                                                                                                                                                                                                                      | info                                  |
| log.format<br>LOG_FORMAT                                | Format of the logs (json, console)                                                                                                                                                                                                                          | console                               |
| data.path<br>DATA_PATH                                  | Where data produced by checkship will be saved (deployment artifacts, logs, local db, …)                                                                                                                                                                        | ~/.config/checkship                       |
| data.deployment_dir_template<br>DEPLOYMENT_DIR_TEMPLATE | [Go template](https://pkg.go.dev/text/template) determining the directory where the build will occur (use <code v-pre>{{ .Number }}-{{ .Environment }}</code> if you want to keep all application deployment sources for example)                           | <code v-pre>{{ .Environment }}</code> |
| source.archive.max_size<br>SOURCE_ARCHIVE_MAX_SIZE      | Largest archive accepted when deploying from an uploaded file, with a unit: b, kb, mb, gb or tb (powers of 1024), such as `64mb`                                                                                                                    | 32mb                                  |
| http.host<br>HTTP_HOST                                  | Host to listen to                                                                                                                                                                                                                                           | 0.0.0.0                               |
| http.port<br>HTTP_PORT,PORT                             | Port to listen to                                                                                                                                                                                                                                           | 8080                                  |
| http.secure<br>HTTP_SECURE                              | Wether or not the web server is served over https. If omitted, determine this information from the `EXPOSED_ON` variable. It controls wether or not cookie are set with the `Secure` flag and the scheme used on the `Location` header of created resources | false                                 |
| http.secret<br>HTTP_SECRET                              | Secret key to use when signing cookies                                                                                                                                                                                                                      | &lt;generated if empty&gt;            |
| runners<br>RUNNERS_CONFIGURATION                        | Runners processing the [background jobs](/reference/jobs), [see below](#runners)                                                                                                                                                                            | [see below](#runners)                 |
| -<br>ADMIN_EMAIL                                        | Email of the first user account to create (ignored if a user already exists)                                                                                                                                                                                |                                       |
| -<br>ADMIN_PASSWORD                                     | Password of the first user account to create (ignored if a user already exists)                                                                                                                                                                             |                                       |
| -<br>EXPOSED_ON                                         | Url at which the checkship container [will be exposed](/guide/installation#exposing-checkship) and default target url. In the form `<url scheme>://<container name>@<default target url>`                                                                           |                                       |

## Runners {#runners}

[Background jobs](/reference/jobs) are processed by **runners**. Each runner checks for new jobs at a regular interval and runs up to `count` of them at the same time. Runners work independently, so a long queue of deployments doesn't hold up cleanups or target configurations.

A job also starts as soon as it is queued, or as soon as a worker of its runner becomes free. The poll interval is a fallback.

The default configuration has one runner for deployments and one for every other job:

```yml
runners:
  - poll_interval: 4s
    count: 4
    jobs:
      - deployment.command.deploy
  - poll_interval: 4s
    count: 2
```

| yaml path     | Description                                                                                                                                                             |
| ------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| poll_interval | Interval at which the runner checks for jobs, parsed by [time.ParseDuration](https://pkg.go.dev/time#ParseDuration). At least `500ms`                                  |
| count         | Number of jobs the runner can process at the same time, between 1 and 64                                                                                                |
| jobs          | Names of the jobs handled by this runner. Required on every runner except the last one: when the last runner has no `jobs`, it handles every job not listed elsewhere |

Job names:

- `deployment.command.deploy`
- `deployment.command.cleanup_app`
- `deployment.command.delete_app`
- `deployment.command.configure_target`
- `deployment.command.cleanup_target`
- `deployment.command.delete_target`

Every job must be handled by exactly one runner. **checkship** refuses to start if a job is listed twice, if a name is unknown, or if a job is handled by no runner. If every job is already listed and the last runner has no `jobs`, that runner is not started and a warning is logged.

For example, to deploy up to 8 applications at once and configure targets apart from the cleanups:

```yml
runners:
  - poll_interval: 4s
    count: 8
    jobs:
      - deployment.command.deploy
  - poll_interval: 4s
    count: 1
    jobs:
      - deployment.command.configure_target
  - poll_interval: 10s
    count: 2
```

### With an environment variable

`RUNNERS_CONFIGURATION` replaces the `runners` section of the file. Each runner is written `<poll_interval>;<count>;<job names separated by commas>` and runners are separated by `|`. The default configuration is:

```sh
RUNNERS_CONFIGURATION="4s;4;deployment.command.deploy|4s;2;" checkship serve
```

An empty `RUNNERS_CONFIGURATION` is ignored.

### Older settings

Before runners could be listed, the configuration had three settings. They still work, and **checkship** logs a warning when it finds them:

- In the file, `runners.poll_interval`, `runners.deployment` and `runners.cleanup` are read as the default pair of runners.
- The `RUNNERS_POLL_INTERVAL`, `RUNNERS_DEPLOYMENT_COUNT` and `RUNNERS_CLEANUP_COUNT` variables replace the runners with the default pair, built from their values. They are ignored when `RUNNERS_CONFIGURATION` is set.

The configuration file is not rewritten. Move to the list format when convenient.
