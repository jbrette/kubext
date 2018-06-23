package commands

import (
	"fmt"
	"log"
	"os"

	"github.com/jbrette/kubext/managed/common"
	"github.com/spf13/cobra"
)

func NewSuspendCommand() *cobra.Command {
	var command = &cobra.Command{
		Use:   "suspend WORKFLOW1 WORKFLOW2...",
		Short: "suspend a managed",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 {
				cmd.HelpFunc()(cmd, args)
				os.Exit(1)
			}
			InitManagedClient()
			for _, wfName := range args {
				err := common.SuspendManaged(wfClient, wfName)
				if err != nil {
					log.Fatalf("Failed to suspend %s: %v", wfName, err)
				}
				fmt.Printf("managed %s suspended\n", wfName)
			}
		},
	}
	return command
}
