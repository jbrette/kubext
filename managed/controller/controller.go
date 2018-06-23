package controller

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/jbrette/kubext"
	"github.com/jbrette/kubext/errors"
	"github.com/jbrette/kubext/pkg/apis/managed"
	wfv1 "github.com/jbrette/kubext/pkg/apis/managed/v1alpha1"
	wfclientset "github.com/jbrette/kubext/pkg/client/clientset/versioned"
	unstructutil "github.com/jbrette/kubext/util/unstructured"
	"github.com/jbrette/kubext/managed/common"
	"github.com/ghodss/yaml"
	log "github.com/sirupsen/logrus"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
)

// ManagedController is the controller for managed resources
type ManagedController struct {
	// ConfigMap is the name of the config map in which to derive configuration of the controller from
	ConfigMap string
	// namespace for config map
	ConfigMapNS string
	// Config is the managed controller's configuration
	Config ManagedControllerConfig

	// restConfig is used by controller to send a SIGUSR1 to the wait sidecar using remotecommand.NewSPDYExecutor().
	restConfig    *rest.Config
	kubeclientset kubernetes.Interface
	wfclientset   wfclientset.Interface

	// datastructures to support the processing of manageds and managed pods
	wfInformer    cache.SharedIndexInformer
	podInformer   cache.SharedIndexInformer
	wfQueue       workqueue.RateLimitingInterface
	podQueue      workqueue.RateLimitingInterface
	completedPods chan string
}

// ManagedControllerConfig contain the configuration settings for the managed controller
type ManagedControllerConfig struct {
	// ExecutorImage is the image name of the executor to use when running pods
	ExecutorImage string `json:"executorImage,omitempty"`

	// ExecutorResources specifies the resource requirements that will be used for the executor sidecar
	ExecutorResources *apiv1.ResourceRequirements `json:"executorResources,omitempty"`

	// ArtifactRepository contains the default location of an artifact repository for container artifacts
	ArtifactRepository ArtifactRepository `json:"artifactRepository,omitempty"`

	// Namespace is a label selector filter to limit the controller's watch to a specific namespace
	Namespace string `json:"namespace,omitempty"`

	// InstanceID is a label selector to limit the controller's watch to a specific instance. It
	// contains an arbitrary value that is carried forward into its pod labels, under the key
	// manageds.jbrette.io/controller-instanceid, for the purposes of managed segregation. This
	// enables a controller to only receive managed and pod events that it is interested about,
	// in order to support multiple controllers in a single cluster, and ultimately allows the
	// controller itself to be bundled as part of a higher level application. If omitted, the
	// controller watches manageds and pods that *are not* labeled with an instance id.
	InstanceID string `json:"instanceID,omitempty"`

	MatchLabels map[string]string `json:"matchLabels,omitempty"`
}

const (
	managedResyncPeriod = 20 * time.Minute
	podResyncPeriod      = 30 * time.Minute
)

// ArtifactRepository represents a artifact repository in which a controller will store its artifacts
type ArtifactRepository struct {
	S3 *S3ArtifactRepository `json:"s3,omitempty"`
	// Future artifact repository support here
	Artifactory *ArtifactoryArtifactRepository `json:"artifactory,omitempty"`
}

// S3ArtifactRepository defines the controller configuration for an S3 artifact repository
type S3ArtifactRepository struct {
	wfv1.S3Bucket `json:",inline"`

	// KeyPrefix is prefix used as part of the bucket key in which the controller will store artifacts.
	KeyPrefix string `json:"keyPrefix,omitempty"`
}

// ArtifactoryArtifactRepository defines the controller configuration for an artifactory artifact repository
type ArtifactoryArtifactRepository struct {
	wfv1.ArtifactoryAuth `json:",inline"`
	// RepoURL is the url for artifactory repo.
	RepoURL string `json:"repoURL,omitempty"`
}

// NewManagedController instantiates a new ManagedController
func NewManagedController(restConfig *rest.Config, kubeclientset kubernetes.Interface, wfclientset wfclientset.Interface, configMap string) *ManagedController {
	wfc := ManagedController{
		restConfig:    restConfig,
		kubeclientset: kubeclientset,
		wfclientset:   wfclientset,
		ConfigMap:     configMap,
		wfQueue:       workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter()),
		podQueue:      workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter()),
		completedPods: make(chan string, 512),
	}
	return &wfc
}

// Run starts an Managed resource controller
func (wfc *ManagedController) Run(ctx context.Context, wfWorkers, podWorkers int) {
	defer wfc.wfQueue.ShutDown()
	defer wfc.podQueue.ShutDown()

	log.Infof("Managed Controller (version: %s) starting", kubext.GetVersion())
	log.Info("Watch Managed controller config map updates")
	_, err := wfc.watchControllerConfigMap(ctx)
	if err != nil {
		log.Errorf("Failed to register watch for controller config map: %v", err)
		return
	}

	wfc.wfInformer = wfc.newManagedInformer()
	wfc.podInformer = wfc.newPodInformer()
	go wfc.wfInformer.Run(ctx.Done())
	go wfc.podInformer.Run(ctx.Done())
	go wfc.podLabeler(ctx.Done())

	// Wait for all involved caches to be synced, before processing items from the queue is started
	for _, informer := range []cache.SharedIndexInformer{wfc.wfInformer, wfc.podInformer} {
		if !cache.WaitForCacheSync(ctx.Done(), informer.HasSynced) {
			log.Error("Timed out waiting for caches to sync")
			return
		}
	}

	for i := 0; i < wfWorkers; i++ {
		go wait.Until(wfc.runWorker, time.Second, ctx.Done())
	}
	for i := 0; i < podWorkers; i++ {
		go wait.Until(wfc.podWorker, time.Second, ctx.Done())
	}
	<-ctx.Done()
}

// podLabeler will label all pods on the controllers completedPod channel as completed
func (wfc *ManagedController) podLabeler(stopCh <-chan struct{}) {
	for {
		select {
		case <-stopCh:
			return
		case pod := <-wfc.completedPods:
			parts := strings.Split(pod, "/")
			if len(parts) != 2 {
				log.Warnf("Unexpected item on completed pod channel: %s", pod)
				continue
			}
			namespace := parts[0]
			podName := parts[1]
			err := common.AddPodLabel(wfc.kubeclientset, podName, namespace, common.LabelKeyCompleted, "true")
			if err != nil {
				log.Errorf("Failed to label pod %s/%s completed: %+v", namespace, podName, err)
			} else {
				log.Infof("Labeled pod %s/%s completed", namespace, podName)
			}
		}
	}
}

func (wfc *ManagedController) runWorker() {
	for wfc.processNextItem() {
	}
}

// processNextItem is the worker logic for handling managed updates
func (wfc *ManagedController) processNextItem() bool {
	key, quit := wfc.wfQueue.Get()
	if quit {
		return false
	}
	defer wfc.wfQueue.Done(key)

	obj, exists, err := wfc.wfInformer.GetIndexer().GetByKey(key.(string))
	if err != nil {
		log.Errorf("Failed to get managed '%s' from informer index: %+v", key, err)
		return true
	}
	if !exists {
		// This happens after a managed was labeled with completed=true
		// or was deleted, but the work queue still had an entry for it.
		return true
	}
	// The managed informer receives unstructured objects to deal with the possibility of invalid
	// managed manifests that are unable to unmarshal to managed objects
	un, ok := obj.(*unstructured.Unstructured)
	if !ok {
		log.Warnf("Key '%s' in index is not an unstructured", key)
		return true
	}
	var wf wfv1.Managed
	err = runtime.DefaultUnstructuredConverter.FromUnstructured(un.Object, &wf)
	if err != nil {
		log.Warnf("Failed to unmarshal key '%s' to managed object: %v", key, err)
		woc := newManagedOperationCtx(&wf, wfc)
		woc.markManagedFailed(fmt.Sprintf("invalid spec: %s", err.Error()))
		woc.persistUpdates()
		return true
	}

	if wf.ObjectMeta.Labels[common.LabelKeyCompleted] == "true" {
		// can get here if we already added the completed=true label,
		// but we are still draining the controller's managed workqueue
		return true
	}
	woc := newManagedOperationCtx(&wf, wfc)
	woc.operate()
	// TODO: operate should return error if it was unable to operate properly
	// so we can requeue the work for a later time
	// See: https://github.com/kubernetes/client-go/blob/master/examples/workqueue/main.go
	//c.handleErr(err, key)
	return true
}

func (wfc *ManagedController) podWorker() {
	for wfc.processNextPodItem() {
	}
}

// processNextPodItem is the worker logic for handling pod updates.
// For pods updates, this simply means to "wake up" the managed by
// adding the corresponding managed key into the managed workqueue.
func (wfc *ManagedController) processNextPodItem() bool {
	key, quit := wfc.podQueue.Get()
	if quit {
		return false
	}
	defer wfc.podQueue.Done(key)

	obj, exists, err := wfc.podInformer.GetIndexer().GetByKey(key.(string))
	if err != nil {
		log.Errorf("Failed to get pod '%s' from informer index: %+v", key, err)
		return true
	}
	if !exists {
		// we can get here if pod was queued into the pod workqueue,
		// but it was either deleted or labeled completed by the time
		// we dequeued it.
		return true
	}
	pod, ok := obj.(*apiv1.Pod)
	if !ok {
		log.Warnf("Key '%s' in index is not a pod", key)
		return true
	}
	if pod.Labels == nil {
		log.Warnf("Pod '%s' did not have labels", key)
		return true
	}
	managedName, ok := pod.Labels[common.LabelKeyManaged]
	if !ok {
		// Ignore pods unrelated to managed (this shouldn't happen unless the watch is setup incorrectly)
		log.Warnf("watch returned pod unrelated to any managed: %s", pod.ObjectMeta.Name)
		return true
	}
	// TODO: currently we reawaken the managed on *any* pod updates.
	// But this could be be much improved to become smarter by only
	// requeue the managed when there are changes that we care about.
	wfc.wfQueue.Add(pod.ObjectMeta.Namespace + "/" + managedName)
	return true
}

// ResyncConfig reloads the controller config from the configmap
func (wfc *ManagedController) ResyncConfig() error {
	namespace, _ := os.LookupEnv(common.EnvVarNamespace)
	if namespace == "" {
		namespace = common.DefaultControllerNamespace
	}
	cmClient := wfc.kubeclientset.CoreV1().ConfigMaps(namespace)
	cm, err := cmClient.Get(wfc.ConfigMap, metav1.GetOptions{})
	if err != nil {
		return errors.InternalWrapError(err)
	}
	wfc.ConfigMapNS = cm.Namespace
	return wfc.updateConfig(cm)
}

func (wfc *ManagedController) updateConfig(cm *apiv1.ConfigMap) error {
	configStr, ok := cm.Data[common.ManagedControllerConfigMapKey]
	if !ok {
		return errors.Errorf(errors.CodeBadRequest, "ConfigMap '%s' does not have key '%s'", wfc.ConfigMap, common.ManagedControllerConfigMapKey)
	}
	var config ManagedControllerConfig
	err := yaml.Unmarshal([]byte(configStr), &config)
	if err != nil {
		return errors.InternalWrapError(err)
	}
	log.Printf("managed controller configuration from %s:\n%s", wfc.ConfigMap, configStr)
	if config.ExecutorImage == "" {
		return errors.Errorf(errors.CodeBadRequest, "ConfigMap '%s' does not have executorImage", wfc.ConfigMap)
	}
	wfc.Config = config
	return nil
}

// instanceIDRequirement returns the label requirement to filter against a controller instance (or not)
func (wfc *ManagedController) instanceIDRequirement() labels.Requirement {
	var instanceIDReq *labels.Requirement
	var err error
	if wfc.Config.InstanceID != "" {
		instanceIDReq, err = labels.NewRequirement(common.LabelKeyControllerInstanceID, selection.Equals, []string{wfc.Config.InstanceID})
	} else {
		instanceIDReq, err = labels.NewRequirement(common.LabelKeyControllerInstanceID, selection.DoesNotExist, nil)
	}
	if err != nil {
		panic(err)
	}
	return *instanceIDReq
}

func (wfc *ManagedController) tweakManagedlist(options *metav1.ListOptions) {
	options.FieldSelector = fields.Everything().String()

	// completed notin (true)
	incompleteReq, err := labels.NewRequirement(common.LabelKeyCompleted, selection.NotIn, []string{"true"})
	if err != nil {
		panic(err)
	}
	labelSelector := labels.NewSelector().
		Add(*incompleteReq).
		Add(wfc.instanceIDRequirement())
	options.LabelSelector = labelSelector.String()
}

// newManagedInformer returns the managed informer used by the controller. This is actually
// a custom built UnstructuredInformer which is in actuality returning unstructured.Unstructured
// objects. We no longer return ManagedInformer due to:
// https://github.com/kubernetes/kubernetes/issues/57705
// https://github.com/jbrette/kubext/issues/632
func (wfc *ManagedController) newManagedInformer() cache.SharedIndexInformer {
	dynClientPool := dynamic.NewDynamicClientPool(wfc.restConfig)
	dclient, err := dynClientPool.ClientForGroupVersionKind(wfv1.SchemaGroupVersionKind)
	if err != nil {
		panic(err)
	}
	resource := &metav1.APIResource{
		Name:         managed.Plural,
		SingularName: managed.Singular,
		Namespaced:   true,
		Group:        managed.Group,
		Version:      "v1alpha1",
		ShortNames:   []string{"wf"},
	}
	informer := unstructutil.NewFilteredUnstructuredInformer(
		resource,
		dclient,
		wfc.Config.Namespace,
		managedResyncPeriod,
		cache.Indexers{},
		wfc.tweakManagedlist,
	)
	informer.AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				key, err := cache.MetaNamespaceKeyFunc(obj)
				if err == nil {
					wfc.wfQueue.Add(key)
				}
			},
			UpdateFunc: func(old, new interface{}) {
				key, err := cache.MetaNamespaceKeyFunc(new)
				if err == nil {
					wfc.wfQueue.Add(key)
				}
			},
			DeleteFunc: func(obj interface{}) {
				// IndexerInformer uses a delta queue, therefore for deletes we have to use this
				// key function.
				key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
				if err == nil {
					wfc.wfQueue.Add(key)
				}
			},
		},
	)
	return informer
}

func (wfc *ManagedController) watchControllerConfigMap(ctx context.Context) (cache.Controller, error) {
	source := wfc.newControllerConfigMapWatch()
	_, controller := cache.NewInformer(
		source,
		&apiv1.ConfigMap{},
		0,
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				if cm, ok := obj.(*apiv1.ConfigMap); ok {
					log.Infof("Detected ConfigMap update. Updating the controller config.")
					err := wfc.updateConfig(cm)
					if err != nil {
						log.Errorf("Update of config failed due to: %v", err)
					}
				}
			},
			UpdateFunc: func(old, new interface{}) {
				if newCm, ok := new.(*apiv1.ConfigMap); ok {
					log.Infof("Detected ConfigMap update. Updating the controller config.")
					err := wfc.updateConfig(newCm)
					if err != nil {
						log.Errorf("Update of config failed due to: %v", err)
					}
				}
			},
		})

	go controller.Run(ctx.Done())
	return controller, nil
}

func (wfc *ManagedController) newControllerConfigMapWatch() *cache.ListWatch {
	c := wfc.kubeclientset.CoreV1().RESTClient()
	resource := "configmaps"
	name := wfc.ConfigMap
	namespace := wfc.ConfigMapNS
	fieldSelector := fields.ParseSelectorOrDie(fmt.Sprintf("metadata.name=%s", name))

	listFunc := func(options metav1.ListOptions) (runtime.Object, error) {
		options.FieldSelector = fieldSelector.String()
		req := c.Get().
			Namespace(namespace).
			Resource(resource).
			VersionedParams(&options, metav1.ParameterCodec)
		return req.Do().Get()
	}
	watchFunc := func(options metav1.ListOptions) (watch.Interface, error) {
		options.Watch = true
		options.FieldSelector = fieldSelector.String()
		req := c.Get().
			Namespace(namespace).
			Resource(resource).
			VersionedParams(&options, metav1.ParameterCodec)
		return req.Watch()
	}
	return &cache.ListWatch{ListFunc: listFunc, WatchFunc: watchFunc}
}

func (wfc *ManagedController) newManagedPodWatch() *cache.ListWatch {
	c := wfc.kubeclientset.CoreV1().RESTClient()
	resource := "pods"
	namespace := wfc.Config.Namespace
	fieldSelector := fields.ParseSelectorOrDie("status.phase!=Pending")
	// completed=false
	incompleteReq, _ := labels.NewRequirement(common.LabelKeyCompleted, selection.Equals, []string{"false"})
	labelSelector := labels.NewSelector().
		Add(*incompleteReq).
		Add(wfc.instanceIDRequirement())

	listFunc := func(options metav1.ListOptions) (runtime.Object, error) {
		options.FieldSelector = fieldSelector.String()
		options.LabelSelector = labelSelector.String()
		req := c.Get().
			Namespace(namespace).
			Resource(resource).
			VersionedParams(&options, metav1.ParameterCodec)
		return req.Do().Get()
	}
	watchFunc := func(options metav1.ListOptions) (watch.Interface, error) {
		options.Watch = true
		options.FieldSelector = fieldSelector.String()
		options.LabelSelector = labelSelector.String()
		req := c.Get().
			Namespace(namespace).
			Resource(resource).
			VersionedParams(&options, metav1.ParameterCodec)
		return req.Watch()
	}
	return &cache.ListWatch{ListFunc: listFunc, WatchFunc: watchFunc}
}

func (wfc *ManagedController) newPodInformer() cache.SharedIndexInformer {
	source := wfc.newManagedPodWatch()
	informer := cache.NewSharedIndexInformer(source, &apiv1.Pod{}, podResyncPeriod, cache.Indexers{})
	informer.AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				key, err := cache.MetaNamespaceKeyFunc(obj)
				if err == nil {
					wfc.podQueue.Add(key)
				}
			},
			UpdateFunc: func(old, new interface{}) {
				key, err := cache.MetaNamespaceKeyFunc(new)
				if err == nil {
					wfc.podQueue.Add(key)
				}
			},
			DeleteFunc: func(obj interface{}) {
				// IndexerInformer uses a delta queue, therefore for deletes we have to use this
				// key function.
				key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
				if err == nil {
					wfc.podQueue.Add(key)
				}
			},
		},
	)
	return informer
}
