# dai-enduro-workflows

**dai-enduro-workflows** provides one preprocessing workflow for DAI.

## Configuration
The worker needs to share the filesystem with Enduro's a3m or Archivematica
workers, connect to the same Temporal server, and be related to Enduro with
the correct namespace, task queue and workflow names.


### Worker configuration
An example configuration for the worker binary:


```toml
debug = false
verbosity = 0

[temporal]
address = "temporal-frontend.enduro-sdps:7233"
namespace = "default"

[worker]
taskQueue = "dai-enduro"
maxConcurrentSessions = 1

[preprocessing]
workflowName = "preprocessing"
sharedPath = "/home/enduro/shared"

[preprocessing.bagCreate]
checksumAlgorithm = "sha512"

[preprocessing.fileFormat]
allowlistPath = "/home/enduro/.config/allowed_file_formats.csv"
```

The worker requires a configuration file. It searches the working directory,
`$HOME/.config`, and `/etc` for `dai-enduro-worker.*`, or accepts a path
through `--config`. Environment variables override values from that file. They
use the `DAI_ENDURO_WORKER` prefix and replace dots with underscores.

Configuration validation checks the Temporal address and namespace, task
queue, session limit, workflow name, shared path, and activity settings.

Logs are written to standard error as JSON by default. Set `debug = true` for
readable text and use `verbosity` to control the log level. The Temporal client
and worker use the same logger.

### Enduro
The child workflow sections in Enduro's configuration file:

```toml
[[childWorkflows]]
type = "preprocessing"
taskQueue = "dai-enduro"
workflowName = "preprocessing"
sharedPath = "/home/enduro/preprocessing"
extract = false
```

## Local Development
This project provides a child workflow for the Enduro development environment.
The supported development workflow is to run `tilt up` from the Enduro repository
and load this repository through Enduro's CHILD_WORKFLOW_PATHS mechanism in the
`.tilt.env` file.

Bring up the Enduro environment by following the [Enduro development manual].
```shell
tilt up
```

### Set up
The specific requirements for dai-enduro-workflows are:
- clone this repository as a sibling of the Enduro repository
- configure CHILD_WORKFLOW_PATHS=../dai-enduro-workflows
- configure MOUNT_PREPROCESSING_VOLUME=true
- *optionally:* set TRIGGER_MODE_AUTO=true (for live reloading capabilities)
- run tilt up from the Enduro repository

All other development workflow details, including .tilt.env, live updates, starting, stopping, and clearing the environment, are documented in Enduro. This repository can also provide local overrides through its own .tilt.env file, including settings such as TRIGGER_MODE_AUTO.

### Requirements for development
While we run the services inside a Kubernetes cluster we recommend installing Go and other tools locally to ease the development process.
- Go (1.26+)
- GNU Make and GCC

## Makefile
Run `make help` to list every Makefile target. Common targets are:

```shell
make tools
make fmt
make lint
make test
make test-race
make validate-tilt
make pre-commit
```

## Available Activities
* [Validate SIP Size](#validate-sip-size)
* [Validate file formats](#validate-file-formats)

### Validate SIP Size
Ensures the SIP is no bigger than 1 Terabyte.

### Validate file formats
Identifies each file in the SIP and checks it against an allowed-formats CSV.
The CSV must include a `PRONOM PUID` column. Kubernetes builds `dai-enduro-secret`
from `hack/kube/allowed_file_formats.csv` and mounts it at
`/home/enduro/.config/allowed_file_formats.csv`.

### Other activities
The preprocessing child workflow also uses a
number of other more general Enduro temporal activities, including:
- `bagcreate`
- `bagextract`
- `ffvalidate`

[Enduro development manual]: https://enduro.readthedocs.io/dev-manual/devel/
[go]: https://go.dev/doc/install
[make]: https://www.gnu.org/software/make/
[gcc]: https://gcc.gnu.org/
