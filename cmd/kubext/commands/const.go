package commands

import rbacv1 "k8s.io/api/rbac/v1"

const (
	// Kubext controller resource constants
	KubextControllerServiceAccount     = "kubext"
	KubextControllerClusterRole        = "kubext-cluster-role"
	KubextControllerClusterRoleBinding = "kubext-binding"

	// Kubext UI resource constants
	KubextUIServiceAccount     = "kubext-ui"
	KubextUIClusterRole        = "kubext-ui-cluster-role"
	KubextUIClusterRoleBinding = "kubext-ui-binding"
	KubextUIDeploymentName     = "kubext-ui"
	KubextUIServiceName        = "kubext-ui"
)

var (
	KubextControllerPolicyRules = []rbacv1.PolicyRule{
		{
			APIGroups: []string{""},
			// TODO(jesse): remove exec privileges when issue #499 is resolved
			Resources: []string{"pods", "pods/exec"},
			Verbs:     []string{"create", "get", "list", "watch", "update", "patch"},
		},
		{
			APIGroups: []string{""},
			Resources: []string{"configmaps"},
			Verbs:     []string{"get", "watch", "list"},
		},
		{
			APIGroups: []string{""},
			Resources: []string{"persistentvolumeclaims"},
			Verbs:     []string{"create", "delete"},
		},
		{
			APIGroups: []string{"jbrette.io"},
			Resources: []string{"manageds"},
			Verbs:     []string{"get", "list", "watch", "update", "patch"},
		},
	}

	KubextUIPolicyRules = []rbacv1.PolicyRule{
		{
			APIGroups: []string{""},
			Resources: []string{"pods", "pods/exec", "pods/log"},
			Verbs:     []string{"get", "list", "watch"},
		},
		{
			APIGroups: []string{""},
			Resources: []string{"secrets"},
			Verbs:     []string{"get"},
		},
		{
			APIGroups: []string{"jbrette.io"},
			Resources: []string{"manageds"},
			Verbs:     []string{"get", "list", "watch"},
		},
	}
)
