package common

import (
	"time"

	"github.com/jbrette/kubext/pkg/apis/managed"
)

const (
	// DefaultControllerDeploymentName is the default deployment name of the managed controller
	DefaultControllerDeploymentName = "managed-controller"
	// DefaultControllerNamespace is the default namespace where the managed controller is installed
	DefaultControllerNamespace = "kube-system"

	// ManagedControllerConfigMapKey is the key in the configmap to retrieve managed configuration from.
	// Content encoding is expected to be YAML.
	ManagedControllerConfigMapKey = "config"

	// Container names used in the managed pod
	MainContainerName = "main"
	InitContainerName = "init"
	WaitContainerName = "wait"

	// PodMetadataVolumeName is the volume name defined in a managed pod spec to expose pod metadata via downward API
	PodMetadataVolumeName = "podmetadata"

	// PodMetadataAnnotationsVolumePath is volume path for metadata.annotations in the downward API
	PodMetadataAnnotationsVolumePath = "annotations"
	// PodMetadataMountPath is the directory mount location for DownwardAPI volume containing pod metadata
	PodMetadataMountPath = "/kubext/" + PodMetadataVolumeName
	// PodMetadataAnnotationsPath is the file path containing pod metadata annotations. Examined by executor
	PodMetadataAnnotationsPath = PodMetadataMountPath + "/" + PodMetadataAnnotationsVolumePath

	// DockerLibVolumeName is the volume name for the /var/lib/docker host path volume
	DockerLibVolumeName = "docker-lib"
	// DockerLibHostPath is the host directory path containing docker runtime state
	DockerLibHostPath = "/var/lib/docker"
	// DockerSockVolumeName is the volume name for the /var/run/docker.sock host path volume
	DockerSockVolumeName = "docker-sock"

	// AnnotationKeyNodeName is the pod metadata annotation key containing the managed node name
	AnnotationKeyNodeName = managed.FullName + "/node-name"
	// AnnotationKeyNodeMessage is the pod metadata annotation key the executor will use to
	// communicate errors encountered by the executor during artifact load/save, etc...
	AnnotationKeyNodeMessage = managed.FullName + "/node-message"
	// AnnotationKeyTemplate is the pod metadata annotation key containing the container template as JSON
	AnnotationKeyTemplate = managed.FullName + "/template"
	// AnnotationKeyOutputs is the pod metadata annotation key containing the container outputs
	AnnotationKeyOutputs = managed.FullName + "/outputs"
	// AnnotationKeyExecutionControl is the pod metadata annotation key containing execution control parameters
	// set by the controller and obeyed by the executor. For example, the controller will use this annotation to
	// signal the executors of daemoned containers that it should terminate.
	AnnotationKeyExecutionControl = managed.FullName + "/execution"

	// LabelKeyControllerInstanceID is the label the controller will carry forward to manageds/pod labels
	// for the purposes of managed segregation
	LabelKeyControllerInstanceID = managed.FullName + "/controller-instanceid"
	// LabelKeyCompleted is the metadata label applied on worfklows and managed pods to indicates if resource is completed
	// Manageds and pods with a completed=true label will be ignored by the controller
	LabelKeyCompleted = managed.FullName + "/completed"
	// LabelKeyManaged is the pod metadata label to indicate the associated managed name
	LabelKeyManaged = managed.FullName + "/managed"
	// LabelKeyPhase is a label applied to manageds to indicate the current phase of the managed (for filtering purposes)
	LabelKeyPhase = managed.FullName + "/phase"

	// ExecutorArtifactBaseDir is the base directory in the init container in which artifacts will be copied to.
	// Each artifact will be named according to its input name (e.g: /kubext/inputs/artifacts/CODE)
	ExecutorArtifactBaseDir = "/kubext/inputs/artifacts"

	// InitContainerMainFilesystemDir is a path made available to the init container such that the init container
	// can access the same volume mounts used in the main container. This is used for the purposes of artifact loading
	// (when there is overlapping paths between artifacts and volume mounts)
	InitContainerMainFilesystemDir = "/mainctrfs"

	// ExecutorStagingEmptyDir is the path of the emptydir which is used as a staging area to transfer a file between init/main container for script/resource templates
	ExecutorStagingEmptyDir = "/kubext/staging"
	// ExecutorScriptSourcePath is the path which init will write the script source file to for script templates
	ExecutorScriptSourcePath = "/kubext/staging/script"
	// ExecutorResourceManifestPath is the path which init will write the a manifest file to for resource templates
	ExecutorResourceManifestPath = "/tmp/manifest.yaml"

	// Various environment variables containing pod information exposed to the executor container(s)

	// EnvVarPodIP contains the IP of the pod (currently unused)
	EnvVarPodIP = "ARGO_POD_IP"
	// EnvVarPodName contains the name of the pod (currently unused)
	EnvVarPodName = "ARGO_POD_NAME"
	// EnvVarNamespace contains the namespace of the pod (currently unused)
	EnvVarNamespace = "ARGO_NAMESPACE"

	// These are global variables that are added to the scope during template execution and can be referenced using {{}} syntax

	// GlobalVarManagedName is a global managed variable referencing the managed's metadata.name field
	GlobalVarManagedName = "managed.name"
	// GlobalVarManagedNamespace is a global managed variable referencing the managed's metadata.namespace field
	GlobalVarManagedNamespace = "managed.namespace"
	// GlobalVarManagedUID is a global managed variable referencing the managed's metadata.uid field
	GlobalVarManagedUID = "managed.uid"
	// GlobalVarManagedStatus is a global managed variable referencing the managed's status.phase field
	GlobalVarManagedStatus = "managed.status"
)

// ExecutionControl contains execution control parameters for executor to decide how to execute the container
type ExecutionControl struct {
	// Deadline is a max timestamp in which an executor can run the container before terminating it
	// It is used to signal the executor to terminate a daemoned container. In the future it will be
	// used to support managed or steps/dag level timeouts.
	Deadline *time.Time `json:"deadline,omitempty"`
}
