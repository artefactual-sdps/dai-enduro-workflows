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

func TestAddPREMISEvent(t *testing.T) {
	t.Parallel()

	const objectsXML = `<?xml version="1.0" encoding="UTF-8"?>
<premis:premis xmlns:premis="http://www.loc.gov/premis/v3" xmlns:xlink="http://www.w3.org/1999/xlink" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="http://www.loc.gov/premis/v3 https://www.loc.gov/standards/premis/premis.xsd" version="3.0">
  <premis:object xsi:type="premis:file">
    <premis:objectIdentifier>
      <premis:objectIdentifierType>UUID</premis:objectIdentifierType>
      <premis:objectIdentifierValue>52fdfc07-2182-454f-963f-5f0f9a621d72</premis:objectIdentifierValue>
    </premis:objectIdentifier>
    <premis:objectCharacteristics>
      <premis:format>
        <premis:formatDesignation>
          <premis:formatName/>
        </premis:formatDesignation>
      </premis:format>
    </premis:objectCharacteristics>
    <premis:originalName>data/cat.jpg</premis:originalName>
  </premis:object>
</premis:premis>
`

	td := fs.NewDir(t, "", fs.WithFile("premis.xml", objectsXML))
	path := td.Join("premis.xml")

	ts := &temporalsdk_testsuite.WorkflowTestSuite{}
	env := ts.NewTestActivityEnvironment()
	env.RegisterActivityWithOptions(
		activities.NewAddPREMISEvent().Execute,
		temporalsdk_activity.RegisterOptions{Name: activities.AddPREMISEventName},
	)

	future, err := env.ExecuteActivity(activities.AddPREMISEventName, &activities.AddPREMISEventParams{
		PREMISFilePath: path,
		Agent:          premis.AgentDefault(),
		Type:           "validation",
		Detail:         `name="Validate the SIP structure"`,
		OutcomeDetail:  "The SIP structure is valid",
	})
	assert.NilError(t, err)

	var res activities.AddPREMISEventResult
	assert.NilError(t, future.Get(&res))

	doc, err := premis.ParseFile(path)
	assert.NilError(t, err)

	idValueEl := doc.FindElement("/premis:premis/premis:event/premis:eventIdentifier/premis:eventIdentifierValue")
	assert.Assert(t, idValueEl != nil)
	assert.Equal(t, len(idValueEl.Text()), 36)

	linkEl := doc.FindElement(
		"/premis:premis/premis:object/premis:linkingEventIdentifier/premis:linkingEventIdentifierValue",
	)
	assert.Equal(t, linkEl.Text(), idValueEl.Text())

	detail := doc.FindElement("/premis:premis/premis:event/premis:eventDetailInformation/premis:eventDetail")
	assert.Equal(t, detail.Text(), `name="Validate the SIP structure"`)

	outcome := doc.FindElement("/premis:premis/premis:event/premis:eventOutcomeInformation/premis:eventOutcome")
	assert.Equal(t, outcome.Text(), "valid")

	_, err = os.Stat(path)
	assert.NilError(t, err)
}
