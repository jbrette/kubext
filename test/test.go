package test

import (
	wfv1 "github.com/jbrette/kubext/pkg/apis/managed/v1alpha1"
	"github.com/ghodss/yaml"
	"github.com/gobuffalo/packr"
)

var (
	// Manifests is a packr box to the test manifests
	Manifests = packr.NewBox("e2e")
)

// GetManaged returns a test managed by it's path
func GetManaged(path string) *wfv1.Managed {
	var wf wfv1.Managed
	err := yaml.Unmarshal(Manifests.Bytes(path), &wf)
	if err != nil {
		panic(err)
	}
	// Set the managed name explicitly since generateName doesn't work in unit tests
	if wf.Name == "" {
		wf.Name = wf.GenerateName
	}
	return &wf
}
