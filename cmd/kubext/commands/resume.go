package commands

import (
	"fmt"
	"log"
	"os"

	"github.com/jbrette/kubext/managed/common"
	"github.com/spf13/cobra"
)

func NewResumeCommand() *cobra.Command {
	var command = &cobra.Command{
		Use:   "resume WORKFLOW1 WORKFLOW2...",
		Short: "resume a managed",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 {
				cmd.HelpFunc()(cmd, args)
				os.Exit(1)
			}
			InitManagedClient()
			for _, wfName := range args {
				err := common.ResumeManaged(wfClient, wfName)
				if err != nil {
					log.Fatalf("Failed to resume %s: %+v", wfName, err)
				}
				fmt.Printf("managed %s resumed\n", wfName)
			}
		},
	}
	return command
}
