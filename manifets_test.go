package arseeding

import "testing"

func Test_getRawById(t *testing.T) {
	data, contentType, err := getRawById("arDRw5qt51v4pOV9TrQXKJM2iLK-c39dvs2K-7b3oDk")

	if err != nil {
		t.Error(err)
	}

	t.Log(err)
	t.Log(contentType)
	t.Log(string(data))
}
