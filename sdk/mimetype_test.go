package sdk

import (
	"fmt"
	"mime"
	"path/filepath"
	"testing"
)

func TestCSSFile(T *testing.T) {

	/*	mtype, err := mimetype.DetectFile("../tmpFile/styles.8d84f8ee.css")

		fmt.Println(mtype.String(), mtype.Extension(), err)*/

	cssFile := "../tmpFile/styles.8d84f8ee.css"
	jsFile := "../tmpFile/1f391b9e.847dfa56.js"

	cssExt := filepath.Ext(cssFile) // ".css"
	jsExt := filepath.Ext(jsFile)   // ".js"

	cssType := mime.TypeByExtension(cssExt)
	jsType := mime.TypeByExtension(jsExt)

	fmt.Printf("MIME type of %s: %s\n", cssFile, cssType)
	fmt.Printf("MIME type of %s: %s\n", jsFile, jsType)

}
