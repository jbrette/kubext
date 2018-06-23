package commands

import (
	"os"

	"github.com/jbrette/kubext"
	"github.com/jbrette/kubext/util/cmd"
	"github.com/jbrette/kubext/managed/common"
	"github.com/jbrette/kubext/managed/executor"
	"github.com/jbrette/kubext/managed/executor/docker"
	"github.com/ghodss/yaml"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	// CLIName is the name of the CLI
	CLIName = "kubextexec"
)

var (
	// GlobalArgs hold global CLI flags
	GlobalArgs globalFlags
)

type globalFlags struct {
	podAnnotationsPath string // --pod-annotations
	kubeConfig         string // --kubeconfig
}

func init() {
	RootCmd.PersistentFlags().StringVar(&GlobalArgs.kubeConfig, "kubeconfig", "", "Kubernetes config (used when running outside of cluster)")
	RootCmd.PersistentFlags().StringVar(&GlobalArgs.podAnnotationsPath, "pod-annotations", common.PodMetadataAnnotationsPath, "Pod annotations file from k8s downward API")
	RootCmd.AddCommand(cmd.NewVersionCmd(CLIName))
}

// RootCmd is the kubext root level command
var RootCmd = &cobra.Command{
	Use:   CLIName,
	Short: "kubextexec is the executor sidecar to managed containers",
	Run: func(cmd *cobra.Command, args []string) {
		cmd.HelpFunc()(cmd, args)
	},
}

// getClientConfig return rest config, if path not specified, assume in cluster config
func getClientConfig(kubeconfig string) (*rest.Config, error) {
	if kubeconfig != "" {
		return clientcmd.BuildConfigFromFlags("", kubeconfig)
	}
	return rest.InClusterConfig()
}

func initExecutor() *executor.ManagedExecutor {
	podAnnotationsPath := common.PodMetadataAnnotationsPath

	// Use the path specified from the flag
	if GlobalArgs.podAnnotationsPath != "" {
		podAnnotationsPath = GlobalArgs.podAnnotationsPath
	}

	config, err := getClientConfig(GlobalArgs.kubeConfig)
	if err != nil {
		panic(err.Error())
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}
	podName, ok := os.LookupEnv(common.EnvVarPodName)
	if !ok {
		log.Fatalf("Unable to determine pod name from environment variable %s", common.EnvVarPodName)
	}
	namespace, ok := os.LookupEnv(common.EnvVarNamespace)
	if !ok {
		log.Fatalf("Unable to determine pod namespace from environment variable %s", common.EnvVarNamespace)
	}

	wfExecutor := executor.NewExecutor(clientset, podName, namespace, podAnnotationsPath, &docker.DockerExecutor{})
	err = wfExecutor.LoadTemplate()
	if err != nil {
		panic(err.Error())
	}
	yamlBytes, _ := yaml.Marshal(&wfExecutor.Template)
	log.Infof("Executor (version: %s) initialized with template:\n%s", kubext.GetVersion(), string(yamlBytes))
	return &wfExecutor
}
