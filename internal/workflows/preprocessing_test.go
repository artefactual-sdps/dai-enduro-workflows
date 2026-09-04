package workflows_test

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/artefactual-sdps/enduro/pkg/childwf"
	"github.com/artefactual-sdps/temporal-activities/bagcreate"
	"github.com/artefactual-sdps/temporal-activities/bagextract"
	"github.com/artefactual-sdps/temporal-activities/ffvalidate"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	temporalsdk_activity "go.temporal.io/sdk/activity"
	temporalsdk_testsuite "go.temporal.io/sdk/testsuite"
	temporalsdk_worker "go.temporal.io/sdk/worker"

	"github.com/artefactual-sdps/dai-enduro-workflows/internal/activities"
	"github.com/artefactual-sdps/dai-enduro-workflows/internal/config"
	"github.com/artefactual-sdps/dai-enduro-workflows/internal/csvs"
	"github.com/artefactual-sdps/dai-enduro-workflows/internal/workflows"
)

const (
	sharedPath   = "/shared/path/"
	validSIPName = "SIP_2025-10-20_IANUS1234_ABT"
)

type PreprocessingTestSuite struct {
	suite.Suite
	temporalsdk_testsuite.WorkflowTestSuite

	env      *temporalsdk_testsuite.TestWorkflowEnvironment
	workflow *workflows.PreprocessingWorkflow
}

func (s *PreprocessingTestSuite) SetupTest(cfg config.Configuration) {
	s.env = s.NewTestWorkflowEnvironment()
	s.env.SetWorkerOptions(temporalsdk_worker.Options{EnableSessionWorker: true})

	// Register activities.
	s.env.RegisterActivityWithOptions(
		bagextract.New().Execute,
		temporalsdk_activity.RegisterOptions{Name: bagextract.Name},
	)
	s.env.RegisterActivityWithOptions(
		bagcreate.New(cfg.Preprocessing.BagCreate).Execute,
		temporalsdk_activity.RegisterOptions{Name: bagcreate.Name},
	)

	s.env.RegisterActivityWithOptions(
		activities.NewCheckSIPInfo().Execute,
		temporalsdk_activity.RegisterOptions{Name: activities.CheckSIPInfoName},
	)

	s.env.RegisterActivityWithOptions(
		activities.NewValidateFileAndFolder().Execute,
		temporalsdk_activity.RegisterOptions{Name: activities.ValidateFileAndFolderName},
	)

	s.env.RegisterActivityWithOptions(
		activities.NewValidateSIPStructure().Execute,
		temporalsdk_activity.RegisterOptions{Name: activities.ValidateSIPStructureName},
	)

	s.env.RegisterActivityWithOptions(
		activities.NewValidateSIPMetadata(csvs.NewCSVValidatorCmd()).Execute,
		temporalsdk_activity.RegisterOptions{Name: activities.ValidateSIPMetadataName},
	)

	s.env.RegisterActivityWithOptions(
		ffvalidate.New(cfg.Preprocessing.FileFormat).Execute,
		temporalsdk_activity.RegisterOptions{Name: ffvalidate.Name},
	)

	cfg.Preprocessing.SharedPath = sharedPath
	s.workflow = workflows.NewPreprocessingWorkflow(cfg.Preprocessing)
}

func (s *PreprocessingTestSuite) AfterTest(suiteName, testName string) {
	s.env.AssertExpectations(s.T())
}

func TestPreprocessingWorkflow(t *testing.T) {
	suite.Run(t, new(PreprocessingTestSuite))
}

func (s *PreprocessingTestSuite) TestSuccess() {
	relPath := validSIPName
	s.SetupTest(config.Configuration{
		Preprocessing: config.PreprocessingConfig{
			CSVSchemaPath: "/schema/dai-relaxed-schema.csvs",
		},
	})

	// Mock activities.
	sessionCtx := mock.AnythingOfType("*context.timerCtx")
	srcPath := filepath.Join(sharedPath, relPath)
	s.env.OnActivity(
		bagextract.Name,
		sessionCtx,
		&bagextract.Params{Path: srcPath, Keep: []string{"metadata"}},
	).Return(
		&bagextract.Result{Path: srcPath},
		nil,
	)
	s.env.OnActivity(
		activities.CheckSIPInfoName,
		sessionCtx,
		&activities.CheckSIPInfoParams{Path: filepath.Join(sharedPath, relPath)},
	).Return(
		&activities.CheckSIPInfoResult{
			SizeHuman:           "1.0 kB",
			SizeInBytes:         1024,
			NumberOfFiles:       1,
			NumberOfDirectories: 1,
		},
		nil,
	)
	s.env.OnActivity(
		activities.ValidateFileAndFolderName,
		sessionCtx,
		&activities.ValidateFileAndFolderParams{Path: filepath.Join(sharedPath, relPath)},
	).Return(
		&activities.ValidateFileAndFolderResult{},
		nil,
	)
	s.env.OnActivity(
		activities.ValidateSIPStructureName,
		sessionCtx,
		&activities.ValidateSIPStructureParams{Path: filepath.Join(sharedPath, relPath)},
	).Return(
		&activities.ValidateSIPStructureResult{HasMetadataDirectory: true},
		nil,
	)
	s.env.OnActivity(
		ffvalidate.Name,
		sessionCtx,
		&ffvalidate.Params{Path: srcPath},
	).Return(
		&ffvalidate.Result{},
		nil,
	)
	s.env.OnActivity(
		activities.ValidateSIPMetadataName,
		sessionCtx,
		&activities.ValidateSIPMetadataParams{
			SIPSourcePath: srcPath,
			CSVSchemaPath: "/schema/dai-relaxed-schema.csvs",
		},
	).Return(
		&activities.ValidateSIPMetadataResult{},
		nil,
	)
	s.env.OnActivity(
		bagcreate.Name,
		sessionCtx,
		&bagcreate.Params{SourcePath: srcPath},
	).Return(
		&bagcreate.Result{BagPath: srcPath},
		nil,
	)

	s.env.ExecuteWorkflow(
		s.workflow.Execute,
		&childwf.PreprocessingParams{RelativePath: relPath},
	)

	s.True(s.env.IsWorkflowCompleted())

	var result childwf.PreprocessingResult
	err := s.env.GetWorkflowResult(&result)
	s.NoError(err)
	s.Equal(
		&childwf.PreprocessingResult{
			Outcome:      childwf.OutcomeSuccess,
			RelativePath: relPath,
			Tasks: []*childwf.Task{
				{
					Name:        "Extract the SIP bag",
					Message:     "The SIP bag was extracted.",
					Outcome:     childwf.TaskOutcomeSuccess,
					StartedAt:   s.env.Now().UTC(),
					CompletedAt: s.env.Now().UTC(),
				},
				{
					Name:        "Validate the SIP name",
					Message:     "The SIP name is valid: " + validSIPName,
					Outcome:     childwf.TaskOutcomeSuccess,
					StartedAt:   s.env.Now().UTC(),
					CompletedAt: s.env.Now().UTC(),
				},
				{
					Name:        "Validate the SIP size",
					Message:     "SIP size checked: 1.0 kB",
					Outcome:     childwf.TaskOutcomeSuccess,
					StartedAt:   s.env.Now().UTC(),
					CompletedAt: s.env.Now().UTC(),
				},
				{
					Name:        "Validate the SIP payload",
					Message:     "SIP payload size checked. Files: 1 - Directories: 1",
					Outcome:     childwf.TaskOutcomeSuccess,
					StartedAt:   s.env.Now().UTC(),
					CompletedAt: s.env.Now().UTC(),
				},
				{
					Name:        "Validate file and folder names",
					Message:     "File and folder names are valid",
					Outcome:     childwf.TaskOutcomeSuccess,
					StartedAt:   s.env.Now().UTC(),
					CompletedAt: s.env.Now().UTC(),
				},
				{
					Name:        "Validate the SIP structure",
					Message:     "The SIP structure is valid",
					Outcome:     childwf.TaskOutcomeSuccess,
					StartedAt:   s.env.Now().UTC(),
					CompletedAt: s.env.Now().UTC(),
				},
				{
					Name:        "Validate file formats",
					Message:     "File formats are valid",
					Outcome:     childwf.TaskOutcomeSuccess,
					StartedAt:   s.env.Now().UTC(),
					CompletedAt: s.env.Now().UTC(),
				},
				{
					Name:        "Validate the SIP metadata",
					Message:     "The SIP metadata is valid",
					Outcome:     childwf.TaskOutcomeSuccess,
					StartedAt:   s.env.Now().UTC(),
					CompletedAt: s.env.Now().UTC(),
				},
				{
					Name:        "Bag SIP",
					Message:     "SIP has been bagged",
					Outcome:     childwf.TaskOutcomeSuccess,
					StartedAt:   s.env.Now().UTC(),
					CompletedAt: s.env.Now().UTC(),
				},
			},
		},
		&result,
	)
}

func (s *PreprocessingTestSuite) TestSystemError() {
	relPath := validSIPName
	s.SetupTest(config.Configuration{})

	// Mock activities.
	sessionCtx := mock.AnythingOfType("*context.timerCtx")
	srcPath := filepath.Join(sharedPath, relPath)
	s.env.OnActivity(
		bagextract.Name,
		sessionCtx,
		&bagextract.Params{Path: srcPath, Keep: []string{"metadata"}},
	).Return(
		&bagextract.Result{Path: srcPath},
		nil,
	)
	s.env.OnActivity(
		activities.CheckSIPInfoName,
		sessionCtx,
		&activities.CheckSIPInfoParams{Path: filepath.Join(sharedPath, relPath)},
	).Return(
		&activities.CheckSIPInfoResult{
			SizeHuman:   "1.0 kB",
			SizeInBytes: 1024,
		},
		nil,
	)
	s.env.OnActivity(
		activities.ValidateFileAndFolderName,
		sessionCtx,
		&activities.ValidateFileAndFolderParams{Path: filepath.Join(sharedPath, relPath)},
	).Return(
		&activities.ValidateFileAndFolderResult{},
		nil,
	)
	s.env.OnActivity(
		activities.ValidateSIPStructureName,
		sessionCtx,
		&activities.ValidateSIPStructureParams{Path: filepath.Join(sharedPath, relPath)},
	).Return(
		&activities.ValidateSIPStructureResult{},
		nil,
	)
	s.env.OnActivity(
		ffvalidate.Name,
		sessionCtx,
		&ffvalidate.Params{Path: srcPath},
	).Return(
		&ffvalidate.Result{},
		nil,
	)
	s.env.OnActivity(
		bagcreate.Name,
		sessionCtx,
		&bagcreate.Params{SourcePath: srcPath},
	).Return(
		nil,
		fmt.Errorf(
			"bagcreate: failed to open %s: permission denied",
			srcPath,
		),
	)

	s.env.ExecuteWorkflow(
		s.workflow.Execute,
		&childwf.PreprocessingParams{RelativePath: relPath},
	)

	s.True(s.env.IsWorkflowCompleted())

	var result childwf.PreprocessingResult
	err := s.env.GetWorkflowResult(&result)
	s.NoError(err)
	s.Equal(
		&childwf.PreprocessingResult{
			Outcome:      childwf.OutcomeSystemError,
			RelativePath: relPath,
			Tasks: []*childwf.Task{
				{
					Name:        "Extract the SIP bag",
					Message:     "The SIP bag was extracted.",
					Outcome:     childwf.TaskOutcomeSuccess,
					StartedAt:   s.env.Now().UTC(),
					CompletedAt: s.env.Now().UTC(),
				},
				{
					Name:        "Validate the SIP name",
					Message:     "The SIP name is valid: " + validSIPName,
					Outcome:     childwf.TaskOutcomeSuccess,
					StartedAt:   s.env.Now().UTC(),
					CompletedAt: s.env.Now().UTC(),
				},
				{
					Name:        "Validate the SIP size",
					Message:     "SIP size checked: 1.0 kB",
					Outcome:     childwf.TaskOutcomeSuccess,
					StartedAt:   s.env.Now().UTC(),
					CompletedAt: s.env.Now().UTC(),
				},
				{
					Name:        "Validate the SIP payload",
					Message:     "SIP payload size checked. Files: 0 - Directories: 0",
					Outcome:     childwf.TaskOutcomeSuccess,
					StartedAt:   s.env.Now().UTC(),
					CompletedAt: s.env.Now().UTC(),
				},
				{
					Name:        "Validate file and folder names",
					Message:     "File and folder names are valid",
					Outcome:     childwf.TaskOutcomeSuccess,
					StartedAt:   s.env.Now().UTC(),
					CompletedAt: s.env.Now().UTC(),
				},
				{
					Name:        "Validate the SIP structure",
					Message:     "The SIP structure is valid",
					Outcome:     childwf.TaskOutcomeSuccess,
					StartedAt:   s.env.Now().UTC(),
					CompletedAt: s.env.Now().UTC(),
				},
				{
					Name:        "Validate file formats",
					Message:     "File formats are valid",
					Outcome:     childwf.TaskOutcomeSuccess,
					StartedAt:   s.env.Now().UTC(),
					CompletedAt: s.env.Now().UTC(),
				},
				{
					Name:        "Bag SIP",
					Message:     "System error: bagging has failed",
					Outcome:     childwf.TaskOutcomeSystemFailure,
					StartedAt:   s.env.Now().UTC(),
					CompletedAt: s.env.Now().UTC(),
				},
			},
		},
		&result,
	)
}

func (s *PreprocessingTestSuite) TestSIPSizeSystemError() {
	relPath := validSIPName
	s.SetupTest(config.Configuration{})

	// Mock activities.
	sessionCtx := mock.AnythingOfType("*context.timerCtx")
	srcPath := filepath.Join(sharedPath, relPath)
	s.env.OnActivity(
		bagextract.Name,
		sessionCtx,
		&bagextract.Params{Path: srcPath, Keep: []string{"metadata"}},
	).Return(
		&bagextract.Result{Path: srcPath},
		nil,
	)
	s.env.OnActivity(
		activities.CheckSIPInfoName,
		sessionCtx,
		&activities.CheckSIPInfoParams{Path: filepath.Join(sharedPath, relPath)},
	).Return(
		nil,
		fmt.Errorf("lstat %s: no such file or directory", srcPath),
	)

	s.env.ExecuteWorkflow(
		s.workflow.Execute,
		&childwf.PreprocessingParams{RelativePath: relPath},
	)

	s.True(s.env.IsWorkflowCompleted())

	var result childwf.PreprocessingResult
	err := s.env.GetWorkflowResult(&result)
	s.NoError(err)
	s.Equal(
		&childwf.PreprocessingResult{
			Outcome:      childwf.OutcomeSystemError,
			RelativePath: relPath,
			Tasks: []*childwf.Task{
				{
					Name:        "Extract the SIP bag",
					Message:     "The SIP bag was extracted.",
					Outcome:     childwf.TaskOutcomeSuccess,
					StartedAt:   s.env.Now().UTC(),
					CompletedAt: s.env.Now().UTC(),
				},
				{
					Name:        "Validate the SIP name",
					Message:     "The SIP name is valid: " + validSIPName,
					Outcome:     childwf.TaskOutcomeSuccess,
					StartedAt:   s.env.Now().UTC(),
					CompletedAt: s.env.Now().UTC(),
				},
				{
					Name:        "Validate the SIP size",
					Message:     "System error: SIP validation has failed",
					Outcome:     childwf.TaskOutcomeSystemFailure,
					StartedAt:   s.env.Now().UTC(),
					CompletedAt: s.env.Now().UTC(),
				},
			},
		},
		&result,
	)
}

func (s *PreprocessingTestSuite) TestSIPTooLarge() {
	relPath := "transfer"
	s.SetupTest(config.Configuration{})

	// Mock activities.
	sessionCtx := mock.AnythingOfType("*context.timerCtx")
	sourcePath := filepath.Join(sharedPath, relPath)
	s.env.OnActivity(
		bagextract.Name,
		sessionCtx,
		&bagextract.Params{Path: sourcePath, Keep: []string{"metadata"}},
	).Return(
		&bagextract.Result{Path: sourcePath},
		nil,
	)

	s.env.OnActivity(
		activities.CheckSIPInfoName,
		sessionCtx,
		&activities.CheckSIPInfoParams{Path: filepath.Join(sharedPath, relPath)},
	).Return(
		&activities.CheckSIPInfoResult{
			SizeHuman:   "1.0 TB",
			SizeInBytes: workflows.MAX_BYTES + 1,
		},
		nil,
	)

	s.env.ExecuteWorkflow(
		s.workflow.Execute,
		&childwf.PreprocessingParams{RelativePath: relPath},
	)

	s.True(s.env.IsWorkflowCompleted())

	var result childwf.PreprocessingResult
	err := s.env.GetWorkflowResult(&result)
	s.NoError(err)
	s.Equal(
		&childwf.PreprocessingResult{
			Outcome:      childwf.OutcomeContentError,
			RelativePath: relPath,
			Tasks: []*childwf.Task{
				{
					Name:        "Extract the SIP bag",
					Message:     "The SIP bag was extracted.",
					Outcome:     childwf.TaskOutcomeSuccess,
					StartedAt:   s.env.Now().UTC(),
					CompletedAt: s.env.Now().UTC(),
				},
				{
					Name:        "Validate the SIP name",
					Message:     "Content error: Invalid SIP name 'transfer':\n- expected 4 sections divided by '_', got: 1",
					Outcome:     childwf.TaskOutcomeValidationFailure,
					StartedAt:   s.env.Now().UTC(),
					CompletedAt: s.env.Now().UTC(),
				},
				{
					Name:        "Validate the SIP size",
					Message:     "Content error: SIP is bigger than 1 Terabyte",
					Outcome:     childwf.TaskOutcomeValidationFailure,
					StartedAt:   s.env.Now().UTC(),
					CompletedAt: s.env.Now().UTC(),
				},
				{
					Name:        "Validate the SIP payload",
					Message:     "SIP payload size checked. Files: 0 - Directories: 0",
					Outcome:     childwf.TaskOutcomeSuccess,
					StartedAt:   s.env.Now().UTC(),
					CompletedAt: s.env.Now().UTC(),
				},
			},
		},
		&result,
	)
}

func (s *PreprocessingTestSuite) TestValidationErrors() {
	relPath := "transfer"
	s.SetupTest(config.Configuration{
		Preprocessing: config.PreprocessingConfig{
			CSVSchemaPath: "/schema/dai-relaxed-schema.csvs",
		},
	})

	sessionCtx := mock.AnythingOfType("*context.timerCtx")
	srcPath := filepath.Join(sharedPath, relPath)
	s.env.OnActivity(
		bagextract.Name,
		sessionCtx,
		&bagextract.Params{Path: srcPath, Keep: []string{"metadata"}},
	).Return(
		&bagextract.Result{Path: srcPath},
		nil,
	)
	s.env.OnActivity(
		activities.CheckSIPInfoName,
		sessionCtx,
		&activities.CheckSIPInfoParams{Path: srcPath},
	).Return(
		&activities.CheckSIPInfoResult{
			SizeHuman:           "1.0 kB",
			SizeInBytes:         1024,
			NumberOfFiles:       1,
			NumberOfDirectories: 1,
		},
		nil,
	)
	s.env.OnActivity(
		activities.ValidateFileAndFolderName,
		sessionCtx,
		&activities.ValidateFileAndFolderParams{Path: srcPath},
	).Return(
		&activities.ValidateFileAndFolderResult{
			ValidationErrors: []string{
				`"bad folder" has disallowed characters, allowed: a-z A-Z 0-9 dash (-) and underscore (_)`,
				`folder "other/data" has a duplicate name "data" in the SIP`,
			},
		},
		nil,
	)
	s.env.OnActivity(
		activities.ValidateSIPStructureName,
		sessionCtx,
		&activities.ValidateSIPStructureParams{Path: srcPath},
	).Return(
		&activities.ValidateSIPStructureResult{
			HasMetadataDirectory: true,
			ValidationErrors:     []string{"Metadata directory must include a README.md file"},
		},
		nil,
	)
	s.env.OnActivity(
		ffvalidate.Name,
		sessionCtx,
		&ffvalidate.Params{Path: srcPath},
	).Return(
		&ffvalidate.Result{
			Failures: []string{`file format "fmt/11" not allowed: "payload.bin"`},
		},
		nil,
	)
	s.env.OnActivity(
		activities.ValidateSIPMetadataName,
		sessionCtx,
		&activities.ValidateSIPMetadataParams{
			SIPSourcePath: srcPath,
			CSVSchemaPath: "/schema/dai-relaxed-schema.csvs",
		},
	).Return(
		&activities.ValidateSIPMetadataResult{
			ValidationErrors: []string{
				`notEmpty fails for line: 1, column: filename, value: ""`,
				`notEmpty fails for line: 1, column: identifier, value: ""`,
				`notEmpty fails for line: 1, column: identifier.ianus, value: ""`,
			},
		},
		nil,
	)

	s.env.ExecuteWorkflow(
		s.workflow.Execute,
		&childwf.PreprocessingParams{RelativePath: relPath},
	)

	s.True(s.env.IsWorkflowCompleted())

	var result childwf.PreprocessingResult
	err := s.env.GetWorkflowResult(&result)
	s.NoError(err)
	s.Equal(
		&childwf.PreprocessingResult{
			Outcome:      childwf.OutcomeContentError,
			RelativePath: relPath,
			Tasks: []*childwf.Task{
				{
					Name:        "Extract the SIP bag",
					Message:     "The SIP bag was extracted.",
					Outcome:     childwf.TaskOutcomeSuccess,
					StartedAt:   s.env.Now().UTC(),
					CompletedAt: s.env.Now().UTC(),
				},
				{
					Name:        "Validate the SIP name",
					Message:     "Content error: Invalid SIP name 'transfer':\n- expected 4 sections divided by '_', got: 1",
					Outcome:     childwf.TaskOutcomeValidationFailure,
					StartedAt:   s.env.Now().UTC(),
					CompletedAt: s.env.Now().UTC(),
				},
				{
					Name:        "Validate the SIP size",
					Message:     "SIP size checked: 1.0 kB",
					Outcome:     childwf.TaskOutcomeSuccess,
					StartedAt:   s.env.Now().UTC(),
					CompletedAt: s.env.Now().UTC(),
				},
				{
					Name:        "Validate the SIP payload",
					Message:     "SIP payload size checked. Files: 1 - Directories: 1",
					Outcome:     childwf.TaskOutcomeSuccess,
					StartedAt:   s.env.Now().UTC(),
					CompletedAt: s.env.Now().UTC(),
				},
				{
					Name: "Validate file and folder names",
					Message: "Content error: Invalid file and folder names:\n" +
						"- \"bad folder\" has disallowed characters, allowed: a-z A-Z 0-9 dash (-) and underscore (_)\n" +
						"- folder \"other/data\" has a duplicate name \"data\" in the SIP",
					Outcome:     childwf.TaskOutcomeValidationFailure,
					StartedAt:   s.env.Now().UTC(),
					CompletedAt: s.env.Now().UTC(),
				},
				{
					Name:        "Validate the SIP structure",
					Message:     "Content error: Invalid SIP structure:\n- Metadata directory must include a README.md file",
					Outcome:     childwf.TaskOutcomeValidationFailure,
					StartedAt:   s.env.Now().UTC(),
					CompletedAt: s.env.Now().UTC(),
				},
				{
					Name:        "Validate file formats",
					Message:     "Content error: Invalid file formats:\n- file format \"fmt/11\" not allowed: \"payload.bin\"",
					Outcome:     childwf.TaskOutcomeValidationFailure,
					StartedAt:   s.env.Now().UTC(),
					CompletedAt: s.env.Now().UTC(),
				},
				{
					Name: "Validate the SIP metadata",
					Message: "Content error: Invalid SIP metadata:\n" +
						"- notEmpty fails for line: 1, column: filename, value: \"\"\n" +
						"- notEmpty fails for line: 1, column: identifier, value: \"\"\n" +
						"- notEmpty fails for line: 1, column: identifier.ianus, value: \"\"",
					Outcome:     childwf.TaskOutcomeValidationFailure,
					StartedAt:   s.env.Now().UTC(),
					CompletedAt: s.env.Now().UTC(),
				},
			},
		},
		&result,
	)
}

func (s *PreprocessingTestSuite) TestSIPPayloadTooLarge() {
	relPath := validSIPName

	type test struct {
		result  activities.CheckSIPInfoResult
		message string
	}

	testCases := map[string]test{
		"Too many files": {
			result: activities.CheckSIPInfoResult{
				SizeHuman:     "1.0 kB",
				SizeInBytes:   1024,
				NumberOfFiles: workflows.MAX_FILES + 1,
			},
			message: fmt.Sprintf("Content error: SIP payload has more than %d files", workflows.MAX_FILES),
		},
		"Too many directories": {
			result: activities.CheckSIPInfoResult{
				SizeHuman:           "1.0 kB",
				SizeInBytes:         1024,
				NumberOfDirectories: workflows.MAX_DIRECTORIES + 1,
			},
			message: fmt.Sprintf(
				"Content error: SIP payload has more than %d directories",
				workflows.MAX_DIRECTORIES,
			),
		},
		"Too many files and directories": {
			result: activities.CheckSIPInfoResult{
				SizeHuman:           "1.0 kB",
				SizeInBytes:         1024,
				NumberOfFiles:       workflows.MAX_FILES + 1,
				NumberOfDirectories: workflows.MAX_DIRECTORIES + 1,
			},
			message: fmt.Sprintf(
				"Content error: SIP payload has more than %d files - SIP payload has more than %d directories",
				workflows.MAX_FILES,
				workflows.MAX_DIRECTORIES,
			),
		},
	}

	for name, tc := range testCases {
		s.Run(name, func() {
			s.SetupTest(config.Configuration{})

			sessionCtx := mock.AnythingOfType("*context.timerCtx")
			srcPath := filepath.Join(sharedPath, relPath)
			s.env.OnActivity(
				bagextract.Name,
				sessionCtx,
				&bagextract.Params{Path: srcPath, Keep: []string{"metadata"}},
			).Return(
				&bagextract.Result{Path: srcPath},
				nil,
			)
			s.env.OnActivity(
				activities.CheckSIPInfoName,
				sessionCtx,
				&activities.CheckSIPInfoParams{Path: srcPath},
			).Return(
				&tc.result,
				nil,
			)

			s.env.ExecuteWorkflow(
				s.workflow.Execute,
				&childwf.PreprocessingParams{RelativePath: relPath},
			)

			s.True(s.env.IsWorkflowCompleted())

			var result childwf.PreprocessingResult
			err := s.env.GetWorkflowResult(&result)
			s.NoError(err)
			s.Equal(
				&childwf.PreprocessingResult{
					Outcome:      childwf.OutcomeContentError,
					RelativePath: relPath,
					Tasks: []*childwf.Task{
						{
							Name:        "Extract the SIP bag",
							Message:     "The SIP bag was extracted.",
							Outcome:     childwf.TaskOutcomeSuccess,
							StartedAt:   s.env.Now().UTC(),
							CompletedAt: s.env.Now().UTC(),
						},
						{
							Name:        "Validate the SIP name",
							Message:     "The SIP name is valid: " + validSIPName,
							Outcome:     childwf.TaskOutcomeSuccess,
							StartedAt:   s.env.Now().UTC(),
							CompletedAt: s.env.Now().UTC(),
						},
						{
							Name:        "Validate the SIP size",
							Message:     "SIP size checked: 1.0 kB",
							Outcome:     childwf.TaskOutcomeSuccess,
							StartedAt:   s.env.Now().UTC(),
							CompletedAt: s.env.Now().UTC(),
						},
						{
							Name:        "Validate the SIP payload",
							Message:     tc.message,
							Outcome:     childwf.TaskOutcomeValidationFailure,
							StartedAt:   s.env.Now().UTC(),
							CompletedAt: s.env.Now().UTC(),
						},
					},
				},
				&result,
			)
		})
	}
}
