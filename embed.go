package localfinance

import (
	"embed"
	"io/fs"
)

//go:embed all:frontend/dist
var EmbeddedFrontend embed.FS

//go:embed samples/credit_cards/*.pdf samples/credit_cards/*.csv samples/savings/*.pdf samples/savings/*.csv samples/investments/*.xls samples/investments/*.xlsx
var embeddedSamples embed.FS

func GetSampleFS() fs.FS {
	return embeddedSamples
}

func GetStaticFS() fs.FS {
	sub, err := fs.Sub(EmbeddedFrontend, "frontend/dist")
	if err != nil {
		return nil
	}
	return sub
}
