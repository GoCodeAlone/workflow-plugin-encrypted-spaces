package main

import (
	"github.com/GoCodeAlone/workflow-plugin-encrypted-spaces/internal"
	"github.com/GoCodeAlone/workflow/plugin/external/sdk"
)

func main() {
	sdk.Serve(internal.NewEncryptedSpacesProvider(),
		sdk.WithBuildVersion(sdk.ResolveBuildVersion(internal.Version)),
	)
}
