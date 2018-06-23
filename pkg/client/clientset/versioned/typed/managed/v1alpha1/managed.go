package v1alpha1

import (
	v1alpha1 "github.com/jbrette/kubext/pkg/apis/managed/v1alpha1"
	scheme "github.com/jbrette/kubext/pkg/client/clientset/versioned/scheme"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	rest "k8s.io/client-go/rest"
)

// ManagedsGetter has a method to return a ManagedInterface.
// A group's client should implement this interface.
type ManagedsGetter interface {
	Manageds(namespace string) ManagedInterface
}

// ManagedInterface has methods to work with Managed resources.
type ManagedInterface interface {
	Create(*v1alpha1.Managed) (*v1alpha1.Managed, error)
	Update(*v1alpha1.Managed) (*v1alpha1.Managed, error)
	Delete(name string, options *v1.DeleteOptions) error
	DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error
	Get(name string, options v1.GetOptions) (*v1alpha1.Managed, error)
	List(opts v1.ListOptions) (*v1alpha1.ManagedList, error)
	Watch(opts v1.ListOptions) (watch.Interface, error)
	Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1alpha1.Managed, err error)
	ManagedExpansion
}

// manageds implements ManagedInterface
type manageds struct {
	client rest.Interface
	ns     string
}

// newManageds returns a Manageds
func newManageds(c *ArgoprojV1alpha1Client, namespace string) *manageds {
	return &manageds{
		client: c.RESTClient(),
		ns:     namespace,
	}
}

// Get takes name of the managed, and returns the corresponding managed object, and an error if there is any.
func (c *manageds) Get(name string, options v1.GetOptions) (result *v1alpha1.Managed, err error) {
	result = &v1alpha1.Managed{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("manageds").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of Manageds that match those selectors.
func (c *manageds) List(opts v1.ListOptions) (result *v1alpha1.ManagedList, err error) {
	result = &v1alpha1.ManagedList{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("manageds").
		VersionedParams(&opts, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested manageds.
func (c *manageds) Watch(opts v1.ListOptions) (watch.Interface, error) {
	opts.Watch = true
	return c.client.Get().
		Namespace(c.ns).
		Resource("manageds").
		VersionedParams(&opts, scheme.ParameterCodec).
		Watch()
}

// Create takes the representation of a managed and creates it.  Returns the server's representation of the managed, and an error, if there is any.
func (c *manageds) Create(managed *v1alpha1.Managed) (result *v1alpha1.Managed, err error) {
	result = &v1alpha1.Managed{}
	err = c.client.Post().
		Namespace(c.ns).
		Resource("manageds").
		Body(managed).
		Do().
		Into(result)
	return
}

// Update takes the representation of a managed and updates it. Returns the server's representation of the managed, and an error, if there is any.
func (c *manageds) Update(managed *v1alpha1.Managed) (result *v1alpha1.Managed, err error) {
	result = &v1alpha1.Managed{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("manageds").
		Name(managed.Name).
		Body(managed).
		Do().
		Into(result)
	return
}

// Delete takes name of the managed and deletes it. Returns an error if one occurs.
func (c *manageds) Delete(name string, options *v1.DeleteOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("manageds").
		Name(name).
		Body(options).
		Do().
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *manageds) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("manageds").
		VersionedParams(&listOptions, scheme.ParameterCodec).
		Body(options).
		Do().
		Error()
}

// Patch applies the patch and returns the patched managed.
func (c *manageds) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1alpha1.Managed, err error) {
	result = &v1alpha1.Managed{}
	err = c.client.Patch(pt).
		Namespace(c.ns).
		Resource("manageds").
		SubResource(subresources...).
		Name(name).
		Body(data).
		Do().
		Into(result)
	return
}
