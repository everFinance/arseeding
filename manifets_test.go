package arseeding

import (
	"encoding/json"
	"github.com/everFinance/arseeding/schema"
	"testing"
)

func Test_getRawById(t *testing.T) {
	data, contentType, err := getRawById("U1FqvR_xTuL2qxrJDw20oIghpGt1eTumJ9ZfCczc5_M")

	if err != nil {
		t.Error(err)
	}

	t.Log(err)
	t.Log(contentType)
	t.Log(string(data))

	mani := schema.ManifestData{}
	err = json.Unmarshal(data, &mani)
	t.Log(err)
}

func TestNewS3Store(t *testing.T) {
	err := syncManifestData("U1FqvR_xTuL2qxrJDw20oIghpGt1eTumJ9ZfCczc5_M", nil)
	// err := syncManifestData("WUg9McRBT1i_F6utVYjFYS_pP6dzKcrC1FRccjH6inE", nil)
	t.Log(err)
}
