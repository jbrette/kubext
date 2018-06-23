package commands

import (
	"os"

	"github.com/jbrette/kubext/util/cmd"
	"github.com/spf13/cobra"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	// CLIName is the name of the CLI
	CLIName = "kubext"
)

// NewCommand returns a new instance of an kubextcd command
func NewCommand() *cobra.Command {
	var command = &cobra.Command{
		Use:   CLIName,
		Short: "kubext is the command line interface to Argo",
		Run: func(cmd *cobra.Command, args []string) {
			cmd.HelpFunc()(cmd, args)
		},
	}

	command.AddCommand(NewCompletionCommand())
	command.AddCommand(NewDeleteCommand())
	command.AddCommand(NewGetCommand())
	command.AddCommand(NewInstallCommand())
	command.AddCommand(NewLintCommand())
	command.AddCommand(NewListCommand())
	command.AddCommand(NewLogsCommand())
	command.AddCommand(NewResubmitCommand())
	command.AddCommand(NewResumeCommand())
	command.AddCommand(NewRetryCommand())
	command.AddCommand(NewSubmitCommand())
	command.AddCommand(NewSuspendCommand())
	command.AddCommand(NewUninstallCommand())
	command.AddCommand(NewWaitCommand())
	command.AddCommand(cmd.NewVersionCmd(CLIName))

	addKubectlFlagsToCmd(command)
	return command
}

func addKubectlFlagsToCmd(cmd *cobra.Command) {
	// The "usual" clientcmd/kubectl flags
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	loadingRules.DefaultClientConfig = &clientcmd.DefaultClientConfig
	overrides := clientcmd.ConfigOverrides{}
	kflags := clientcmd.RecommendedConfigOverrideFlags("")
	cmd.PersistentFlags().StringVar(&loadingRules.ExplicitPath, "kubeconfig", "", "Path to a kube config. Only required if out-of-cluster")
	clientcmd.BindOverrideFlags(&overrides, cmd.PersistentFlags(), kflags)
	clientConfig = clientcmd.NewInteractiveDeferredLoadingClientConfig(loadingRules, &overrides, os.Stdin)
}
