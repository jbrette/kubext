package main

import (
	"fmt"
	"os"

	"github.com/jbrette/kubext/cmd/kubext/commands"
	// load the gcp plugin (required to authenticate against GKE clusters).
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	// load the oidc plugin (required to authenticate with OpenID Connect).
	_ "k8s.io/client-go/plugin/pkg/client/auth/oidc"
)

func main() {
	if err := commands.NewCommand().Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
