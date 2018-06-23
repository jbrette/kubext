package install

import (
	"fmt"
	"strconv"

	"github.com/jbrette/kubext"
	"github.com/jbrette/kubext-cd/util/diff"
	"github.com/jbrette/kubext-cd/util/kube"
	"github.com/jbrette/kubext/errors"
	"github.com/jbrette/kubext/managed/common"
	"github.com/jbrette/kubext/managed/controller"
	"github.com/ghodss/yaml"
	"github.com/gobuffalo/packr"
	goversion "github.com/hashicorp/go-version"
	log "github.com/sirupsen/logrus"
	"github.com/yudai/gojsondiff/formatter"
	appsv1beta2 "k8s.io/api/apps/v1beta2"
	apiv1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apiextensionsv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	apierr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type InstallOptions struct {
	Upgrade          bool   // --upgrade
	DryRun           bool   // --dry-run
	Namespace        string // --namespace
	InstanceID       string // --instanceid
	ConfigMap        string // --configmap
	ControllerImage  string // --controller-image
	ServiceAccount   string // --service-account
	ExecutorImage    string // --executor-image
	UIImage          string // --ui-image
	UIBaseHref       string // --ui-base-href
	UIServiceAccount string // --ui-service-account
	EnableWebConsole bool   // --enable-web-console
	ImagePullPolicy  string // --image-pull-policy
}

type Installer struct {
	InstallOptions
	box           packr.Box
	config        *rest.Config
	dynClientPool dynamic.ClientPool
	disco         discovery.DiscoveryInterface
	rbacSupported *bool
	clientset     *kubernetes.Clientset
}

func NewInstaller(config *rest.Config, opts InstallOptions) (*Installer, error) {
	shallowCopy := *config
	inst := Installer{
		InstallOptions: opts,
		box:            packr.NewBox("./manifests"),
		config:         &shallowCopy,
	}
	var err error
	inst.dynClientPool = dynamic.NewDynamicClientPool(inst.config)
	inst.disco, err = discovery.NewDiscoveryClientForConfig(inst.config)
	if err != nil {
		return nil, err
	}
	inst.clientset, err = kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	return &inst, nil
}

// Install installs the Argo controller and UI in the given Namespace
func (i *Installer) Install() {
	if !i.DryRun {
		fmt.Printf("Installing Argo %s into namespace '%s'\n", kubext.GetVersion(), i.Namespace)
		kubernetesVersionCheck(i.clientset)
	}
	i.InstallManagedCRD()
	i.InstallManagedController()
	i.InstallArgoUI()
}

func kubernetesVersionCheck(clientset *kubernetes.Clientset) {
	// Check if the Kubernetes version is >= 1.8
	versionInfo, err := clientset.ServerVersion()
	if err != nil {
		log.Fatalf("Failed to get Kubernetes version: %v", err)
	}

	serverVersion, err := goversion.NewVersion(versionInfo.String())
	if err != nil {
		log.Fatalf("Failed to create version: %v", err)
	}

	minVersion, err := goversion.NewVersion("1.8")
	if err != nil {
		log.Fatalf("Failed to create minimum version: %v", err)
	}

	if serverVersion.LessThan(minVersion) {
		log.Fatalf("Server version %v < %v. Installation won't proceed...\n", serverVersion, minVersion)
	}

	fmt.Printf("Proceeding with Kubernetes version %v\n", serverVersion)
}

// IsRBACSupported returns whether or not RBAC is supported on the cluster
func (i *Installer) IsRBACSupported() bool {
	if i.rbacSupported != nil {
		return *i.rbacSupported
	}
	// TODO: figure out the proper way to test if RBAC is enabled
	clusterRoles := i.clientset.RbacV1().ClusterRoles()
	_, err := clusterRoles.Get("cluster-admin", metav1.GetOptions{})
	if err != nil {
		if apierr.IsNotFound(err) {
			f := false
			i.rbacSupported = &f
			return false
		}
		log.Fatalf("Failed to lookup 'cluster-admin' role: %v", err)
	}
	t := true
	i.rbacSupported = &t
	return true

}

func (i *Installer) InstallManagedCRD() {
	var managedCRD apiextensionsv1beta1.CustomResourceDefinition
	i.unmarshalManifest("01_managed-crd.yaml", &managedCRD)
	obj := kube.MustToUnstructured(&managedCRD)
	i.MustInstallResource(obj)
}

func (i *Installer) InstallManagedController() {
	var managedControllerServiceAccount apiv1.ServiceAccount
	var managedControllerClusterRole rbacv1.ClusterRole
	var managedControllerClusterRoleBinding rbacv1.ClusterRoleBinding
	//var managedControllerConfigMap apiv1.ConfigMap
	var managedControllerDeployment appsv1beta2.Deployment
	i.unmarshalManifest("02a_managed-controller-sa.yaml", &managedControllerServiceAccount)
	i.unmarshalManifest("02b_managed-controller-cluster-role.yaml", &managedControllerClusterRole)
	i.unmarshalManifest("02c_managed-controller-cluster-rolebinding.yaml", &managedControllerClusterRoleBinding)
	//i.unmarshalManifest("02d_managed-controller-configmap.yaml", &managedControllerConfigMap)
	i.unmarshalManifest("02e_managed-controller-deployment.yaml", &managedControllerDeployment)
	managedControllerDeployment.Spec.Template.Spec.Containers[0].Image = i.ControllerImage
	managedControllerDeployment.Spec.Template.Spec.Containers[0].ImagePullPolicy = apiv1.PullPolicy(i.ImagePullPolicy)
	if i.ServiceAccount == "" {
		i.MustInstallResource(kube.MustToUnstructured(&managedControllerServiceAccount))
		if i.IsRBACSupported() {
			managedControllerClusterRoleBinding.Subjects[0].Namespace = i.Namespace
			i.MustInstallResource(kube.MustToUnstructured(&managedControllerClusterRole))
			i.MustInstallResource(kube.MustToUnstructured(&managedControllerClusterRoleBinding))
		}
	} else {
		managedControllerDeployment.Spec.Template.Spec.ServiceAccountName = i.ServiceAccount
	}
	//i.MustInstallResource(kube.MustToUnstructured(&managedControllerConfigMap))
	i.installConfigMap(i.clientset)
	i.MustInstallResource(kube.MustToUnstructured(&managedControllerDeployment))
}

func (i *Installer) InstallArgoUI() {
	var kubextUIServiceAccount apiv1.ServiceAccount
	var kubextUIClusterRole rbacv1.ClusterRole
	var kubextUIClusterRoleBinding rbacv1.ClusterRoleBinding
	var kubextUIDeployment appsv1beta2.Deployment
	var kubextUIService apiv1.Service
	i.unmarshalManifest("03a_kubext-ui-sa.yaml", &kubextUIServiceAccount)
	i.unmarshalManifest("03b_kubext-ui-cluster-role.yaml", &kubextUIClusterRole)
	i.unmarshalManifest("03c_kubext-ui-cluster-rolebinding.yaml", &kubextUIClusterRoleBinding)
	i.unmarshalManifest("03d_kubext-ui-deployment.yaml", &kubextUIDeployment)
	i.unmarshalManifest("03e_kubext-ui-service.yaml", &kubextUIService)
	kubextUIDeployment.Spec.Template.Spec.Containers[0].Image = i.UIImage
	kubextUIDeployment.Spec.Template.Spec.Containers[0].ImagePullPolicy = apiv1.PullPolicy(i.ImagePullPolicy)
	setEnv(&kubextUIDeployment, "ENABLE_WEB_CONSOLE", strconv.FormatBool(i.EnableWebConsole))
	setEnv(&kubextUIDeployment, "BASE_HREF", i.UIBaseHref)
	if i.UIServiceAccount == "" {
		i.MustInstallResource(kube.MustToUnstructured(&kubextUIServiceAccount))
		if i.IsRBACSupported() {
			kubextUIClusterRoleBinding.Subjects[0].Namespace = i.Namespace
			i.MustInstallResource(kube.MustToUnstructured(&kubextUIClusterRole))
			i.MustInstallResource(kube.MustToUnstructured(&kubextUIClusterRoleBinding))
		}
	} else {
		kubextUIDeployment.Spec.Template.Spec.ServiceAccountName = i.UIServiceAccount
	}
	i.MustInstallResource(kube.MustToUnstructured(&kubextUIDeployment))
	i.MustInstallResource(kube.MustToUnstructured(&kubextUIService))
}

func setEnv(dep *appsv1beta2.Deployment, key, val string) {
	ctr := dep.Spec.Template.Spec.Containers[0]
	for i, env := range ctr.Env {
		if env.Name == key {
			env.Value = val
			ctr.Env[i] = env
			return
		}
	}
	ctr.Env = append(ctr.Env, apiv1.EnvVar{Name: key, Value: val})
}

func (i *Installer) unmarshalManifest(fileName string, obj interface{}) {
	yamlBytes, err := i.box.MustBytes(fileName)
	checkError(err)
	err = yaml.Unmarshal(yamlBytes, obj)
	checkError(err)
}

func (i *Installer) MustInstallResource(obj *unstructured.Unstructured) *unstructured.Unstructured {
	obj, err := i.InstallResource(obj)
	checkError(err)
	return obj
}

func isNamespaced(obj *unstructured.Unstructured) bool {
	switch obj.GetKind() {
	case "Namespace", "ClusterRole", "ClusterRoleBinding", "CustomResourceDefinition":
		return false
	}
	return true
}

// InstallResource creates or updates a resource. If installed resource is up-to-date, does nothing
func (i *Installer) InstallResource(obj *unstructured.Unstructured) (*unstructured.Unstructured, error) {
	if isNamespaced(obj) {
		obj.SetNamespace(i.Namespace)
	}
	// remove 'creationTimestamp' and 'status' fields from object so that the diff will not be modified
	obj.SetCreationTimestamp(metav1.Time{})
	delete(obj.Object, "status")
	if i.DryRun {
		printYAML(obj)
		return nil, nil
	}
	gvk := obj.GroupVersionKind()
	dclient, err := i.dynClientPool.ClientForGroupVersionKind(gvk)
	if err != nil {
		return nil, err
	}
	apiResource, err := kube.ServerResourceForGroupVersionKind(i.disco, gvk)
	if err != nil {
		return nil, err
	}
	reIf := dclient.Resource(apiResource, i.Namespace)
	liveObj, err := reIf.Create(obj)
	if err == nil {
		fmt.Printf("%s '%s' created\n", liveObj.GetKind(), liveObj.GetName())
		return liveObj, nil
	}
	if !apierr.IsAlreadyExists(err) {
		return nil, err
	}
	liveObj, err = reIf.Get(obj.GetName(), metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	diffRes := diff.Diff(obj, liveObj)
	if !diffRes.Modified {
		fmt.Printf("%s '%s' up-to-date\n", liveObj.GetKind(), liveObj.GetName())
		return liveObj, nil
	}
	if !i.Upgrade {
		log.Println(diffRes.ASCIIFormat(obj, formatter.AsciiFormatterConfig{}))
		return nil, fmt.Errorf("%s '%s' already exists. Rerun with --upgrade to update", obj.GetKind(), obj.GetName())
	}
	liveObj, err = reIf.Update(obj)
	if err != nil {
		return nil, err
	}
	fmt.Printf("%s '%s' updated\n", liveObj.GetKind(), liveObj.GetName())
	return liveObj, nil
}

func printYAML(obj interface{}) {
	objBytes, err := yaml.Marshal(obj)
	if err != nil {
		log.Fatalf("Failed to marshal %v", obj)
	}
	fmt.Printf("---\n%s\n", string(objBytes))
}

// checkError is a convenience function to exit if an error is non-nil and exit if it was
func checkError(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func (i *Installer) installConfigMap(clientset *kubernetes.Clientset) {
	cmClient := clientset.CoreV1().ConfigMaps(i.Namespace)
	wfConfig := controller.ManagedControllerConfig{
		ExecutorImage: i.ExecutorImage,
		InstanceID:    i.InstanceID,
	}
	configBytes, err := yaml.Marshal(wfConfig)
	if err != nil {
		log.Fatalf("%+v", errors.InternalWrapError(err))
	}
	wfConfigMap := apiv1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ConfigMap",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      i.ConfigMap,
			Namespace: i.Namespace,
		},
		Data: map[string]string{
			common.ManagedControllerConfigMapKey: string(configBytes),
		},
	}
	if i.DryRun {
		printYAML(wfConfigMap)
		return
	}
	_, err = cmClient.Create(&wfConfigMap)
	if err != nil {
		if !apierr.IsAlreadyExists(err) {
			log.Fatalf("Failed to create ConfigMap '%s' in namespace '%s': %v", i.ConfigMap, i.Namespace, err)
		}
		// Configmap already exists. Check if existing configmap needs an update to a new executor image
		existingCM, err := cmClient.Get(i.ConfigMap, metav1.GetOptions{})
		if err != nil {
			log.Fatalf("Failed to retrieve ConfigMap '%s' in namespace '%s': %v", i.ConfigMap, i.Namespace, err)
		}
		configStr, ok := existingCM.Data[common.ManagedControllerConfigMapKey]
		if !ok {
			log.Fatalf("ConfigMap '%s' missing key '%s'", i.ConfigMap, common.ManagedControllerConfigMapKey)
		}
		var existingConfig controller.ManagedControllerConfig
		err = yaml.Unmarshal([]byte(configStr), &existingConfig)
		if err != nil {
			log.Fatalf("Failed to load controller configuration: %v", err)
		}
		if existingConfig.ExecutorImage == wfConfig.ExecutorImage {
			fmt.Printf("Existing ConfigMap '%s' up-to-date\n", i.ConfigMap)
			return
		}
		if !i.Upgrade {
			log.Fatalf("ConfigMap '%s' requires upgrade. Rerun with --upgrade to update the configuration", i.ConfigMap)
		}
		existingConfig.ExecutorImage = i.ExecutorImage
		configBytes, err := yaml.Marshal(existingConfig)
		if err != nil {
			log.Fatalf("%+v", errors.InternalWrapError(err))
		}
		existingCM.Data = map[string]string{
			common.ManagedControllerConfigMapKey: string(configBytes),
		}
		_, err = cmClient.Update(existingCM)
		if err != nil {
			log.Fatalf("Failed to update ConfigMap '%s' in namespace '%s': %v", i.ConfigMap, i.Namespace, err)
		}
		fmt.Printf("ConfigMap '%s' updated\n", i.ConfigMap)
	}
}
