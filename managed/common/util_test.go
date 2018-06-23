package common

import (
	"testing"

	wfv1 "github.com/jbrette/kubext/pkg/apis/managed/v1alpha1"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TestResubmitManagedWithOnExit ensures we do not carry over the onExit node even if successful
func TestResubmitManagedWithOnExit(t *testing.T) {
	wfName := "test-wf"
	onExitName := wfName + ".onExit"
	wf := wfv1.Managed{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-wf",
		},
		Status: wfv1.ManagedStatus{
			Phase: wfv1.NodeFailed,
			Nodes: map[string]wfv1.NodeStatus{},
		},
	}
	onExitID := wf.NodeID(onExitName)
	wf.Status.Nodes[onExitID] = wfv1.NodeStatus{
		Name:  onExitName,
		Phase: wfv1.NodeSucceeded,
	}
	newWF, err := FormulateResubmitManaged(&wf, true)
	assert.Nil(t, err)
	newWFOnExitName := newWF.ObjectMeta.Name + ".onExit"
	newWFOneExitID := newWF.NodeID(newWFOnExitName)
	_, ok := newWF.Status.Nodes[newWFOneExitID]
	assert.False(t, ok)
}
