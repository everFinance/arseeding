package schema

const (
	ManifestType = "application/x.arweave-manifest+json"
	ContentType  = "Content-Type"
	ManiData     = `{
  "manifest": "arweave/paths",
  "version": "0.1.0",
  "index": {
    "path": "index.html"
  },
  "paths": {
    "index.html": {
      "id": "cG7Hdi_iTQPoEYgQJFqJ8NMpN4KoZ-vH_j7pG4iP7NI"
    },
    "js/style.css": {
      "id": "fZ4d7bkCAUiXSfo3zFsPiQvpLVKVtXUKB6kiLNt2XVQ"
    },
    "css/style.css": {
      "id": "fZ4d7bkCAUiXSfo3zFsPiQvpLVKVtXUKB6kiLNt2XVQ"
    },
    "css/mobile.css": {
      "id": "fZ4d7bkCAUiXSfo3zFsPiQvpLVKVtXUKB6kiLNt2XVQ"
    },
    "assets/img/logo.png": {
      "id": "QYWh-QsozsYu2wor0ZygI5Zoa_fRYFc8_X1RkYmw_fU"
    },
    "assets/img/icon.png": {
      "id": "0543SMRGYuGKTaqLzmpOyK4AxAB96Fra2guHzYxjRGo"
    }
  }
}`
)

type ManifestData struct {
	Manifest string              `json:"manifest"` // must be "arweave/paths"
	Version  string              `json:"version"`  // currently "0.1.0"
	Index    IndexPath           `json:"index"`
	Paths    map[string]Resource `json:"paths"`
}

type IndexPath struct {
	Path string `json:"path"`
}

type Resource struct {
	TxId string `json:"id"`
}

type Manifest struct {
	ID          uint   `gorm:"primarykey" json:"-"`
	ManifestUrl string `gorm:"index:idxMani0,unique" json:"manifestUrl"`
	ManifestId  string `json:"manifestId"` // arId
}
