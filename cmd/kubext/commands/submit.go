package commands

import (
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"

	wfv1 "github.com/jbrette/kubext/pkg/apis/managed/v1alpha1"
	cmdutil "github.com/jbrette/kubext/util/cmd"
	"github.com/jbrette/kubext/managed/common"
	"github.com/spf13/cobra"
)

type submitFlags struct {
	name           string   // --name
	generateName   string   // --generate-name
	instanceID     string   // --instanceid
	entrypoint     string   // --entrypoint
	parameters     []string // --parameter
	output         string   // --output
	wait           bool     // --wait
	serviceAccount string   // --serviceaccount
}

func NewSubmitCommand() *cobra.Command {
	var (
		submitArgs submitFlags
	)
	var command = &cobra.Command{
		Use:   "submit FILE1 FILE2...",
		Short: "submit a managed",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 {
				cmd.HelpFunc()(cmd, args)
				os.Exit(1)
			}
			SubmitManageds(args, &submitArgs)
		},
	}
	command.Flags().StringVar(&submitArgs.name, "name", "", "override metadata.name")
	command.Flags().StringVar(&submitArgs.generateName, "generate-name", "", "override metadata.generateName")
	command.Flags().StringVar(&submitArgs.entrypoint, "entrypoint", "", "override entrypoint")
	command.Flags().StringArrayVarP(&submitArgs.parameters, "parameter", "p", []string{}, "pass an input parameter")
	command.Flags().StringVarP(&submitArgs.output, "output", "o", "", "Output format. One of: name|json|yaml|wide")
	command.Flags().BoolVarP(&submitArgs.wait, "wait", "w", false, "wait for the managed to complete")
	command.Flags().StringVar(&submitArgs.serviceAccount, "serviceaccount", "", "run all pods in the managed using specified serviceaccount")
	command.Flags().StringVar(&submitArgs.instanceID, "instanceid", "", "submit with a specific controller's instance id label")
	return command
}

func SubmitManageds(filePaths []string, submitArgs *submitFlags) {
	InitManagedClient()
	var managedNames []string
	for _, filePath := range filePaths {
		var body []byte
		var err error
		if cmdutil.IsURL(filePath) {
			response, err := http.Get(filePath)
			if err != nil {
				log.Fatal(err)
			}
			body, err = ioutil.ReadAll(response.Body)
			_ = response.Body.Close()
			if err != nil {
				log.Fatal(err)
			}
		} else {
			body, err = ioutil.ReadFile(filePath)
			if err != nil {
				log.Fatal(err)
			}
		}
		manageds, err := splitYAMLFile(body)
		if err != nil {
			log.Fatalf("%s failed to parse: %v", filePath, err)
		}
		for _, wf := range manageds {
			wfName, err := submitManaged(&wf, submitArgs)
			if err != nil {
				log.Fatalf("Managed manifest %s failed submission: %v", filePath, err)
			}

			managedNames = append(managedNames, wfName)
		}
	}

	if submitArgs.wait {
		wsp := NewManagedStatusPoller(wfClient, false, submitArgs.output == "json")
		wsp.WaitManageds(managedNames)
	}
}

// submitManaged is a helper to validate and submit a single managed and override the entrypoint/params supplied from command line
func submitManaged(wf *wfv1.Managed, submitArgs *submitFlags) (string, error) {
	if submitArgs.entrypoint != "" {
		wf.Spec.Entrypoint = submitArgs.entrypoint
	}
	if submitArgs.serviceAccount != "" {
		wf.Spec.ServiceAccountName = submitArgs.serviceAccount
	}
	if submitArgs.instanceID != "" {
		labels := wf.GetLabels()
		if labels == nil {
			labels = make(map[string]string)
		}
		labels[common.LabelKeyControllerInstanceID] = submitArgs.instanceID
		wf.SetLabels(labels)
	}
	if len(submitArgs.parameters) > 0 {
		newParams := make([]wfv1.Parameter, 0)
		passedParams := make(map[string]bool)
		for _, paramStr := range submitArgs.parameters {
			parts := strings.SplitN(paramStr, "=", 2)
			if len(parts) == 1 {
				log.Fatalf("Expected parameter of the form: NAME=VALUE. Received: %s", paramStr)
			}
			param := wfv1.Parameter{
				Name:  parts[0],
				Value: &parts[1],
			}
			newParams = append(newParams, param)
			passedParams[param.Name] = true
		}
		for _, param := range wf.Spec.Arguments.Parameters {
			if _, ok := passedParams[param.Name]; ok {
				// this parameter was overridden via command line
				continue
			}
			newParams = append(newParams, param)
		}
		wf.Spec.Arguments.Parameters = newParams
	}
	if submitArgs.generateName != "" {
		wf.ObjectMeta.GenerateName = submitArgs.generateName
	}
	if submitArgs.name != "" {
		wf.ObjectMeta.Name = submitArgs.name
	}
	err := common.ValidateManaged(wf)
	if err != nil {
		return "", err
	}
	created, err := wfClient.Create(wf)
	if err != nil {
		return "", err
	}
	printManaged(created, submitArgs.output)
	return created.Name, nil
}
