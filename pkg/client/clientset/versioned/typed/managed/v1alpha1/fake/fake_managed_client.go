package fake

import (
	v1alpha1 "github.com/jbrette/kubext/pkg/client/clientset/versioned/typed/managed/v1alpha1"
	rest "k8s.io/client-go/rest"
	testing "k8s.io/client-go/testing"
)

type FakeKubextprojV1alpha1 struct {
	*testing.Fake
}

func (c *FakeKubextprojV1alpha1) Manageds(namespace string) v1alpha1.ManagedInterface {
	return &FakeManageds{c, namespace}
}

// RESTClient returns a RESTClient that is used to communicate
// with API server by this client implementation.
func (c *FakeKubextprojV1alpha1) RESTClient() rest.Interface {
	var ret *rest.RESTClient
	return ret
}
