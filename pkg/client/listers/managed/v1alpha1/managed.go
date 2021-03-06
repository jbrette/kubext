// This file was automatically generated by lister-gen

package v1alpha1

import (
	v1alpha1 "github.com/jbrette/kubext/pkg/apis/managed/v1alpha1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/cache"
)

// ManagedLister helps list Manageds.
type ManagedLister interface {
	// List lists all Manageds in the indexer.
	List(selector labels.Selector) (ret []*v1alpha1.Managed, err error)
	// Manageds returns an object that can list and get Manageds.
	Manageds(namespace string) ManagedNamespaceLister
	ManagedListerExpansion
}

// managedLister implements the ManagedLister interface.
type managedLister struct {
	indexer cache.Indexer
}

// NewManagedLister returns a new ManagedLister.
func NewManagedLister(indexer cache.Indexer) ManagedLister {
	return &managedLister{indexer: indexer}
}

// List lists all Manageds in the indexer.
func (s *managedLister) List(selector labels.Selector) (ret []*v1alpha1.Managed, err error) {
	err = cache.ListAll(s.indexer, selector, func(m interface{}) {
		ret = append(ret, m.(*v1alpha1.Managed))
	})
	return ret, err
}

// Manageds returns an object that can list and get Manageds.
func (s *managedLister) Manageds(namespace string) ManagedNamespaceLister {
	return managedNamespaceLister{indexer: s.indexer, namespace: namespace}
}

// ManagedNamespaceLister helps list and get Manageds.
type ManagedNamespaceLister interface {
	// List lists all Manageds in the indexer for a given namespace.
	List(selector labels.Selector) (ret []*v1alpha1.Managed, err error)
	// Get retrieves the Managed from the indexer for a given namespace and name.
	Get(name string) (*v1alpha1.Managed, error)
	ManagedNamespaceListerExpansion
}

// managedNamespaceLister implements the ManagedNamespaceLister
// interface.
type managedNamespaceLister struct {
	indexer   cache.Indexer
	namespace string
}

// List lists all Manageds in the indexer for a given namespace.
func (s managedNamespaceLister) List(selector labels.Selector) (ret []*v1alpha1.Managed, err error) {
	err = cache.ListAllByNamespace(s.indexer, s.namespace, selector, func(m interface{}) {
		ret = append(ret, m.(*v1alpha1.Managed))
	})
	return ret, err
}

// Get retrieves the Managed from the indexer for a given namespace and name.
func (s managedNamespaceLister) Get(name string) (*v1alpha1.Managed, error) {
	obj, exists, err := s.indexer.GetByKey(s.namespace + "/" + name)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(v1alpha1.Resource("managed"), name)
	}
	return obj.(*v1alpha1.Managed), nil
}
