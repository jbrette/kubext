package commands

import (
	"os"

	"github.com/jbrette/kubext/managed/common"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func NewRetryCommand() *cobra.Command {
	var (
		submitArgs submitFlags
	)
	var command = &cobra.Command{
		Use:   "retry WORKFLOW",
		Short: "retry a managed",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 {
				cmd.HelpFunc()(cmd, args)
				os.Exit(1)
			}
			kubeClient := initKubeClient()
			wfClient := InitManagedClient()
			wf, err := wfClient.Get(args[0], metav1.GetOptions{})
			if err != nil {
				log.Fatal(err)
			}
			wf, err = common.RetryManaged(kubeClient, wfClient, wf)
			if err != nil {
				log.Fatal(err)
			}
			printManaged(wf, submitArgs.output)
			if submitArgs.wait {
				wsp := NewManagedStatusPoller(wfClient, false, submitArgs.output == "json")
				wsp.WaitManageds([]string{wf.ObjectMeta.Name})
			}
		},
	}
	command.Flags().StringVarP(&submitArgs.output, "output", "o", "", "Output format. One of: name|json|yaml|wide")
	command.Flags().BoolVarP(&submitArgs.wait, "wait", "w", false, "wait for the managed to complete")
	return command
}
