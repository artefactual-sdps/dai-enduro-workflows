package activities_test

import (
	pseudorand "math/rand"
	"os"
	"testing"

	temporalsdk_activity "go.temporal.io/sdk/activity"
	temporalsdk_testsuite "go.temporal.io/sdk/testsuite"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/fs"

	"github.com/artefactual-sdps/dai-enduro-workflows/internal/activities"
	"github.com/artefactual-sdps/dai-enduro-workflows/internal/premis"
)

func TestAddPREMISObjects(t *testing.T) {
	t.Parallel()

	sipDir := fs.NewDir(t, "",
		fs.WithFile("cat.jpg", ""),
		fs.WithDir("metadata",
			fs.WithFile("README.md", ""),
			fs.WithFile("metadata.csv", ""),
			fs.WithFile("premis.xml", premis.EmptyXML),
		),
	)

	premisFilePath := sipDir.Join("metadata", "premis.xml")

	ts := &temporalsdk_testsuite.WorkflowTestSuite{}
	env := ts.NewTestActivityEnvironment()
	rng := pseudorand.New(pseudorand.NewSource(1)) // #nosec G404
	env.RegisterActivityWithOptions(
		activities.NewAddPREMISObjects(rng).Execute,
		temporalsdk_activity.RegisterOptions{Name: activities.AddPREMISObjectsName},
	)

	future, err := env.ExecuteActivity(activities.AddPREMISObjectsName, &activities.AddPREMISObjectsParams{
		SIPPath:        sipDir.Path(),
		PREMISFilePath: premisFilePath,
	})
	assert.NilError(t, err)

	var res activities.AddPREMISObjectsResult
	assert.NilError(t, future.Get(&res))
	assert.DeepEqual(t, res, activities.AddPREMISObjectsResult{})

	b, err := os.ReadFile(premisFilePath)
	assert.NilError(t, err)
	assert.Equal(t, string(b), `<?xml version="1.0" encoding="UTF-8"?>
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
  <premis:object xsi:type="premis:file">
    <premis:objectIdentifier>
      <premis:objectIdentifierType>UUID</premis:objectIdentifierType>
      <premis:objectIdentifierValue>9566c74d-1003-4c4d-bbbb-0407d1e2c649</premis:objectIdentifierValue>
    </premis:objectIdentifier>
    <premis:objectCharacteristics>
      <premis:format>
        <premis:formatDesignation>
          <premis:formatName/>
        </premis:formatDesignation>
      </premis:format>
    </premis:objectCharacteristics>
    <premis:originalName>data/metadata/README.md</premis:originalName>
  </premis:object>
  <premis:object xsi:type="premis:file">
    <premis:objectIdentifier>
      <premis:objectIdentifierType>UUID</premis:objectIdentifierType>
      <premis:objectIdentifierValue>81855ad8-681d-4d86-91e9-1e00167939cb</premis:objectIdentifierValue>
    </premis:objectIdentifier>
    <premis:objectCharacteristics>
      <premis:format>
        <premis:formatDesignation>
          <premis:formatName/>
        </premis:formatDesignation>
      </premis:format>
    </premis:objectCharacteristics>
    <premis:originalName>data/metadata/metadata.csv</premis:originalName>
  </premis:object>
</premis:premis>
`)
}
