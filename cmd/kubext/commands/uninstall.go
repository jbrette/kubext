package commands

import (
	"fmt"

	"github.com/jbrette/kubext/pkg/apis/managed"
	"github.com/jbrette/kubext/managed/common"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	apierr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type uninstallFlags struct {
	controllerName string // --controller-name
	uiName         string // --ui-name
	configMap      string // --configmap
	namespace      string // --namespace
}

func NewUninstallCommand() *cobra.Command {
	var (
		uninstallArgs uninstallFlags
	)
	var command = &cobra.Command{
		Use:   "uninstall",
		Short: "uninstall Kubext",
		Run: func(cmd *cobra.Command, args []string) {
			uninstallArgs.namespace = InstallNamespace()
			uninstall(&uninstallArgs)
		},
	}
	command.Flags().StringVar(&uninstallArgs.controllerName, "controller-name", common.DefaultControllerDeploymentName, "name of controller deployment")
	command.Flags().StringVar(&uninstallArgs.uiName, "ui-name", KubextUIDeploymentName, "name of ui deployment")
	command.Flags().StringVar(&uninstallArgs.configMap, "configmap", common.DefaultConfigMapName(common.DefaultControllerDeploymentName), "name of configmap to uninstall")
	return command
}

func uninstall(uninstallArgs *uninstallFlags) {
	clientset = initKubeClient()
	fmt.Printf("Uninstalling from namespace '%s'\n", uninstallArgs.namespace)
	// Delete the UI service
	svcClient := clientset.CoreV1().Services(uninstallArgs.namespace)
	err := svcClient.Delete(KubextUIServiceName, &metav1.DeleteOptions{})
	if err != nil {
		if !apierr.IsNotFound(err) {
			log.Fatalf("Failed to delete service '%s': %v", KubextUIServiceName, err)
		}
		fmt.Printf("Service '%s' in namespace '%s' not found\n", KubextUIServiceName, uninstallArgs.namespace)
	} else {
		fmt.Printf("Service '%s' deleted\n", KubextUIServiceName)
	}

	// Delete the UI and managed-controller deployment
	deploymentsClient := clientset.AppsV1beta2().Deployments(uninstallArgs.namespace)
	deletePolicy := metav1.DeletePropagationForeground
	for _, depName := range []string{uninstallArgs.uiName, uninstallArgs.controllerName} {
		err := deploymentsClient.Delete(depName, &metav1.DeleteOptions{PropagationPolicy: &deletePolicy})
		if err != nil {
			if !apierr.IsNotFound(err) {
				log.Fatalf("Failed to delete deployment '%s': %v", depName, err)
			}
			fmt.Printf("Deployment '%s' in namespace '%s' not found\n", depName, uninstallArgs.namespace)
		} else {
			fmt.Printf("Deployment '%s' deleted\n", depName)
		}
	}

	// Delete the configmap
	cmClient := clientset.CoreV1().ConfigMaps(uninstallArgs.namespace)
	err = cmClient.Delete(uninstallArgs.configMap, &metav1.DeleteOptions{})
	if err != nil {
		if !apierr.IsNotFound(err) {
			log.Fatalf("Failed to delete ConfigMap '%s': %v", uninstallArgs.configMap, err)
		}
		fmt.Printf("ConfigMap '%s' in namespace '%s' not found\n", uninstallArgs.configMap, uninstallArgs.namespace)
	} else {
		fmt.Printf("ConfigMap '%s' deleted\n", uninstallArgs.configMap)
	}

	// Delete controller and UI role binding
	for _, bindingName := range []string{KubextControllerClusterRoleBinding, KubextUIClusterRoleBinding} {
		if err := clientset.RbacV1().ClusterRoleBindings().Delete(bindingName, &metav1.DeleteOptions{}); err != nil {
			if !apierr.IsNotFound(err) {
				log.Fatalf("Failed to delete ClusterRoleBinding: %v\n", err)
			}
			fmt.Printf("ClusterRoleBinding '%s' not found\n", bindingName)
		} else {
			fmt.Printf("ClusterRoleBinding '%s' deleted\n", bindingName)
		}
	}

	// Delete controller and UI the cluster role
	for _, roleName := range []string{KubextControllerClusterRole, KubextUIClusterRole} {
		if err := clientset.RbacV1().ClusterRoles().Delete(roleName, &metav1.DeleteOptions{}); err != nil {
			if !apierr.IsNotFound(err) {
				log.Fatalf("Failed to delete ClusterRole: %v\n", err)
			}
			fmt.Printf("ClusterRole '%s' not found\n", roleName)
		} else {
			fmt.Printf("ClusterRole '%s' deleted\n", roleName)
		}
	}

	// Delete controller and UI service account
	for _, serviceAccount := range []string{KubextControllerServiceAccount, KubextUIServiceAccount} {
		if err := clientset.CoreV1().ServiceAccounts(uninstallArgs.namespace).Delete(serviceAccount, &metav1.DeleteOptions{}); err != nil {
			if !apierr.IsNotFound(err) {
				log.Fatalf("Failed to delete ServiceAccount: %v\n", err)
			}
			fmt.Printf("ServiceAccount '%s' in namespace '%s' not found\n", serviceAccount, uninstallArgs.namespace)
		} else {
			fmt.Printf("ServiceAccount '%s' deleted\n", serviceAccount)
		}
	}

	// Delete the managed CRD
	apiextensionsclientset := apiextensionsclient.NewForConfigOrDie(restConfig)
	crdClient := apiextensionsclientset.Apiextensions().CustomResourceDefinitions()
	err = crdClient.Delete(managed.FullName, nil)
	if err != nil {
		if !apierr.IsNotFound(err) {
			log.Fatalf("Failed to delete CustomResourceDefinition '%s': %v", managed.FullName, err)
		}
		fmt.Printf("CustomResourceDefinition '%s' not found\n", managed.FullName)
	} else {
		fmt.Printf("CustomResourceDefinition '%s' deleted\n", managed.FullName)
	}
}
