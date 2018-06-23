package fake

import (
	v1alpha1 "github.com/jbrette/kubext/pkg/apis/managed/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"
)

// FakeManageds implements ManagedInterface
type FakeManageds struct {
	Fake *FakeArgoprojV1alpha1
	ns   string
}

var managedsResource = schema.GroupVersionResource{Group: "jbrette.io", Version: "v1alpha1", Resource: "manageds"}

var managedsKind = schema.GroupVersionKind{Group: "jbrette.io", Version: "v1alpha1", Kind: "Managed"}

// Get takes name of the managed, and returns the corresponding managed object, and an error if there is any.
func (c *FakeManageds) Get(name string, options v1.GetOptions) (result *v1alpha1.Managed, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewGetAction(managedsResource, c.ns, name), &v1alpha1.Managed{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.Managed), err
}

// List takes label and field selectors, and returns the list of Manageds that match those selectors.
func (c *FakeManageds) List(opts v1.ListOptions) (result *v1alpha1.ManagedList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewListAction(managedsResource, managedsKind, c.ns, opts), &v1alpha1.ManagedList{})

	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &v1alpha1.ManagedList{}
	for _, item := range obj.(*v1alpha1.ManagedList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested manageds.
func (c *FakeManageds) Watch(opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewWatchAction(managedsResource, c.ns, opts))

}

// Create takes the representation of a managed and creates it.  Returns the server's representation of the managed, and an error, if there is any.
func (c *FakeManageds) Create(managed *v1alpha1.Managed) (result *v1alpha1.Managed, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewCreateAction(managedsResource, c.ns, managed), &v1alpha1.Managed{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.Managed), err
}

// Update takes the representation of a managed and updates it. Returns the server's representation of the managed, and an error, if there is any.
func (c *FakeManageds) Update(managed *v1alpha1.Managed) (result *v1alpha1.Managed, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateAction(managedsResource, c.ns, managed), &v1alpha1.Managed{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.Managed), err
}

// Delete takes name of the managed and deletes it. Returns an error if one occurs.
func (c *FakeManageds) Delete(name string, options *v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewDeleteAction(managedsResource, c.ns, name), &v1alpha1.Managed{})

	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeManageds) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	action := testing.NewDeleteCollectionAction(managedsResource, c.ns, listOptions)

	_, err := c.Fake.Invokes(action, &v1alpha1.ManagedList{})
	return err
}

// Patch applies the patch and returns the patched managed.
func (c *FakeManageds) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1alpha1.Managed, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceAction(managedsResource, c.ns, name, data, subresources...), &v1alpha1.Managed{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.Managed), err
}
