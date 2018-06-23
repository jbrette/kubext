package commands

import (
	"fmt"
	"log"
	"os"

	"github.com/jbrette/kubext/managed/common"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NewDeleteCommand returns a new instance of an `kubextcd repo` command
func NewDeleteCommand() *cobra.Command {
	var (
		all       bool
		completed bool
	)

	var command = &cobra.Command{
		Use:   "delete WORKFLOW",
		Short: "delete a managed and its associated pods",
		Run: func(cmd *cobra.Command, args []string) {
			wfClient = InitManagedClient()
			if all {
				deleteManageds(metav1.ListOptions{})
				return
			} else if completed {
				options := metav1.ListOptions{
					LabelSelector: fmt.Sprintf("%s=true", common.LabelKeyCompleted),
				}
				deleteManageds(options)
				return
			}
			if len(args) == 0 {
				cmd.HelpFunc()(cmd, args)
				os.Exit(1)
			}
			for _, wfName := range args {
				deleteManaged(wfName)
			}
		},
	}

	command.Flags().BoolVar(&all, "all", false, "Delete all manageds")
	command.Flags().BoolVar(&completed, "completed", false, "Delete completed manageds")
	return command
}

func deleteManaged(wfName string) {
	err := wfClient.Delete(wfName, &metav1.DeleteOptions{})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Managed '%s' deleted\n", wfName)
}

func deleteManageds(options metav1.ListOptions) {
	wfList, err := wfClient.List(options)
	if err != nil {
		log.Fatal(err)
	}
	for _, wf := range wfList.Items {
		deleteManaged(wf.ObjectMeta.Name)
	}
}
