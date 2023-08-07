package arseeding

import "testing"

func Test_getRawById(t *testing.T) {
	data, contentType, err := getRawById("wQk7txuMvlrlYlVozj6aeF7E9dlwar8nNtfs3iNTpbQ")

	if err != nil {
		t.Error(err)
	}

	t.Log(string(data))
	t.Log(contentType)
}
