package activities_test

import (
	"os"
	"testing"

	temporalsdk_activity "go.temporal.io/sdk/activity"
	temporalsdk_testsuite "go.temporal.io/sdk/testsuite"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/fs"

	"github.com/artefactual-sdps/dai-enduro-workflows/internal/activities"
	"github.com/artefactual-sdps/dai-enduro-workflows/internal/premis"
)

func TestAddPREMISAgent(t *testing.T) {
	t.Parallel()

	td := fs.NewDir(t, "")
	path := td.Join("premis.xml")

	ts := &temporalsdk_testsuite.WorkflowTestSuite{}
	env := ts.NewTestActivityEnvironment()
	env.RegisterActivityWithOptions(
		activities.NewAddPREMISAgent().Execute,
		temporalsdk_activity.RegisterOptions{Name: activities.AddPREMISAgentName},
	)

	future, err := env.ExecuteActivity(activities.AddPREMISAgentName, &activities.AddPREMISAgentParams{
		PREMISFilePath: path,
		Agent:          premis.AgentDefault(),
	})
	assert.NilError(t, err)

	var res activities.AddPREMISAgentResult
	assert.NilError(t, future.Get(&res))

	b, err := os.ReadFile(path)
	assert.NilError(t, err)
	assert.Equal(t, string(b), `<?xml version="1.0" encoding="UTF-8"?>
<premis:premis xmlns:premis="http://www.loc.gov/premis/v3" xmlns:xlink="http://www.w3.org/1999/xlink" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="http://www.loc.gov/premis/v3 https://www.loc.gov/standards/premis/premis.xsd" version="3.0">
  <premis:agent>
    <premis:agentIdentifier>
      <premis:agentIdentifierType valueURI="http://id.loc.gov/vocabulary/identifiers/local">url</premis:agentIdentifierType>
      <premis:agentIdentifierValue>https://github.com/artefactual-sdps/dai-enduro-workflows</premis:agentIdentifierValue>
    </premis:agentIdentifier>
    <premis:agentName>Enduro</premis:agentName>
    <premis:agentType>software</premis:agentType>
  </premis:agent>
</premis:premis>
`)
}
