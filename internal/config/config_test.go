package config_test

import (
	"testing"

	"github.com/artefactual-sdps/temporal-activities/bagcreate"
	"github.com/artefactual-sdps/temporal-activities/ffvalidate"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/fs"

	"github.com/artefactual-sdps/dai-enduro-workflows/internal/config"
)

const testConfig = `# Config
debug = true
verbosity = 2
[temporal]
address = "host:port"
namespace = "default"
[worker]
maxConcurrentSessions = 1
taskQueue = "dai-enduro"
[preprocessing]
workflowName = "preprocessing"
sharedPath = "/home/enduro/shared"
[preprocessing.bagCreate]
checksumAlgorithm = "md5"
[preprocessing.fileFormat]
allowlistPath = "/home/enduro/.config/allowed_file_formats.csv"
`

func TestConfig(t *testing.T) {
	t.Parallel()

	type test struct {
		name            string
		configFile      string
		toml            string
		wantFound       bool
		wantCfg         config.Configuration
		wantErr         string
		wantErrContains string
	}

	for _, tc := range []test{
		{
			name:       "Loads configuration from a TOML file",
			configFile: "dai-enduro-worker.toml",
			toml:       testConfig,
			wantFound:  true,
			wantCfg: config.Configuration{
				Debug:     true,
				Verbosity: 2,
				Temporal: config.TemporalConfig{
					Address:   "host:port",
					Namespace: "default",
				},
				Worker: config.WorkerConfig{
					MaxConcurrentSessions: 1,
					TaskQueue:             "dai-enduro",
				},
				Preprocessing: config.PreprocessingConfig{
					WorkflowName: "preprocessing",
					SharedPath:   "/home/enduro/shared",
					BagCreate: bagcreate.Config{
						ChecksumAlgorithm: "md5",
					},
					FileFormatsPath: ffvalidate.Config{
						AllowlistPath: "/home/enduro/.config/allowed_file_formats.csv",
					},
				},
			},
		},
		{
			name:       "Errors when configuration values are not valid",
			configFile: "dai-enduro-worker.toml",
			wantFound:  true,
			wantErr: `invalid configuration
Temporal.Address: missing required value
Worker.TaskQueue: missing required value
Preprocessing.SharedPath: missing required value
Preprocessing.WorkflowName: missing required value`,
		},
		{
			name:       "Errors when MaxConcurrentSessions is less than 1",
			configFile: "dai-enduro-worker.toml",
			toml: `# Config
[temporal]
address = "host:port"
[worker]
maxConcurrentSessions = -1
taskQueue = "dai-enduro"
[preprocessing]
workflowName = "preprocessing"
sharedPath = "/home/enduro/shared"
`,
			wantFound: true,
			wantErr: `invalid configuration
Worker.MaxConcurrentSessions: -1 is less than the minimum value (1)`,
		},
		{
			name:       "Errors when bagcreate checksumAlgorithm is invalid",
			configFile: "dai-enduro-worker.toml",
			toml: `# Config
[temporal]
address = "host:port"
[worker]
taskQueue = "dai-enduro"
[preprocessing]
workflowName = "preprocessing"
sharedPath = "/home/enduro/shared"
[preprocessing.bagCreate]
checksumAlgorithm = "unknown"
`,
			wantFound: true,
			wantErr: `invalid configuration
Preprocessing.BagCreate: ChecksumAlgorithm: invalid value "unknown", must be one of (md5, sha1, sha256, sha512)`,
		},
		{
			name:       "Errors when file format allowlist and disallowlist are both configured",
			configFile: "dai-enduro-worker.toml",
			toml: `# Config
[temporal]
address = "host:port"
[worker]
taskQueue = "dai-enduro"
[preprocessing]
workflowName = "preprocessing"
sharedPath = "/home/enduro/shared"
[preprocessing.fileFormat]
allowlistPath = "/home/enduro/.config/allowed_file_formats.csv"
disallowlistPath = "/home/enduro/.config/disallowed_file_formats.csv"
`,
			wantFound: true,
			wantErr: `invalid configuration
Preprocessing.FileFormat: AllowlistPath and DisallowlistPath cannot both be set`,
		},
		{
			name:       "Errors when TOML is invalid",
			configFile: "dai-enduro-worker.toml",
			toml:       "bad TOML",
			wantFound:  true,
			wantErr:    "failed to read configuration file: While parsing config: toml: expected character =",
		},
		{
			name:            "Errors when no config file is found in the default paths",
			wantFound:       false,
			wantErrContains: "Config File \"dai-enduro-worker\" Not Found in \"[",
		},
		{
			name:            "Errors when the given configFile is not found",
			configFile:      "missing.toml",
			wantFound:       false,
			wantErrContains: "configuration file not found: ",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			tmpDir := fs.NewDir(t, "dai-enduro-worker-test", fs.WithFile("dai-enduro-worker.toml", tc.toml))

			configFile := ""
			if tc.configFile != "" {
				configFile = tmpDir.Join(tc.configFile)
			}

			var c config.Configuration
			found, configFileUsed, err := config.Read(&c, configFile)
			if tc.wantErr != "" {
				assert.Equal(t, found, tc.wantFound)
				assert.Error(t, err, tc.wantErr)
				return
			}
			if tc.wantErrContains != "" {
				assert.Equal(t, found, tc.wantFound)
				assert.ErrorContains(t, err, tc.wantErrContains)
				return
			}

			assert.NilError(t, err)
			assert.Equal(t, found, true)
			assert.Equal(t, configFileUsed, configFile)
			assert.DeepEqual(t, c, tc.wantCfg)
		})
	}
}
