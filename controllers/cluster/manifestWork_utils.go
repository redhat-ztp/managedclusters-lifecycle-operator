package cluster

import (
	"fmt"
	configv1 "github.com/openshift/api/config/v1"
	clusterv1beta1 "github.com/redhat-ztp/managedclusters-lifecycle-operator/apis/cluster/v1beta1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	workv1 "open-cluster-management.io/api/work/v1"
)

const (
	ClusterUpgradeManifestName  = "clusterUpgrade-"
	OperatorUpgradeManifestName = "operatorsUpgrade-"
)

func CreateOperatorUpgradeManifestWork(clusterName string, operatorConfig *clusterv1beta1.OcpOperatorsSpec) (*workv1.ManifestWork, error) {
	if clusterName == "" {
		return nil, fmt.Errorf("Invalid clusterName")
	}

	if operatorConfig == nil {
		return nil, fmt.Errorf("Invalid operatorConfig")
	}

	if !operatorConfig.ApproveAllUpgrades && len(operatorConfig.Include) == 0 && len(operatorConfig.Exclude) == 0 {
		return nil, nil
	}

	manifests := []workv1.Manifest{
		workv1.Manifest{
			RawExtension: runtime.RawExtension{Raw: []byte(NS)},
		},
		//workv1.Manifest{
		//	RawExtension: runtime.RawExtension{Raw: []byte(ClusterRole)},
		//},
		//workv1.Manifest{
		//	RawExtension: runtime.RawExtension{Raw: []byte(ClusterRoleBinding)},
		//},
		//workv1.Manifest{
		//	RawExtension: runtime.RawExtension{Raw: []byte(ServiceAccount)},
		//},
		//workv1.Manifest{
		//	RawExtension: runtime.RawExtension{Raw: []byte(ApproverJob)},
		//},
	}

	manifestWork := &workv1.ManifestWork{
		ObjectMeta: metav1.ObjectMeta{
			Name:      OperatorUpgradeManifestName + clusterName,
			Namespace: clusterName,
		},
		Spec: workv1.ManifestWorkSpec{
			DeleteOption: &workv1.DeleteOption{
				PropagationPolicy: workv1.DeletePropagationPolicyTypeForeground,
			},
			Workload: workv1.ManifestsTemplate{
				Manifests: manifests,
			},
		},
	}
	return manifestWork, nil
}

func CreateClusterVersionUpgradeManifestWork(clusterName string, clusterID string, versionConfig *clusterv1beta1.ClusterVersionSpec) (*workv1.ManifestWork, error) {
	if clusterName == "" {
		return nil, fmt.Errorf("Invalid clusterName")
	}

	if clusterID == "" {
		return nil, fmt.Errorf("Invalid clusterId")
	}

	if versionConfig == nil {
		return nil, fmt.Errorf("Invalid ClusterVersion")
	}

	if versionConfig.Version == "" {
		return nil, fmt.Errorf("Invalid version")
	}

	clusterVersion := &configv1.ClusterVersion{
		ObjectMeta: metav1.ObjectMeta{
			Name: "version",
		},
		Spec: configv1.ClusterVersionSpec{
			ClusterID:     configv1.ClusterID(clusterID),
			DesiredUpdate: &configv1.Update{Version: versionConfig.Version},
		},
	}

	if versionConfig.Channel != "" {
		clusterVersion.Spec.Channel = versionConfig.Channel
	}
	if versionConfig.Image != "" {
		clusterVersion.Spec.DesiredUpdate.Image = versionConfig.Image
	}
	if versionConfig.Upstream != "" {
		clusterVersion.Spec.Upstream = configv1.URL(versionConfig.Upstream)
	}

	versionManifest := workv1.Manifest{
		RawExtension: runtime.RawExtension{Object: clusterVersion},
	}

	roleBinding := &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: "admin-ocm",
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     "cluster-admin",
		},
		Subjects: []rbacv1.Subject{{
			Kind:      "ServiceAccount",
			Name:      "klusterlet-work-sa",
			Namespace: "open-cluster-management-agent",
		}},
	}

	roleBindingManifest := workv1.Manifest{
		RawExtension: runtime.RawExtension{Object: roleBinding},
	}

	manifests := []workv1.Manifest{versionManifest, roleBindingManifest}

	// Create rule for orphaning the clusterVersion CR
	rule := workv1.OrphaningRule{
		Group:     "",
		Resource:  "ClusterVersion.config.openshift.io",
		Namespace: "",
		Name:      "version",
	}
	orphaningRules := []workv1.OrphaningRule{rule}

	manifestWork := &workv1.ManifestWork{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ClusterUpgradeManifestName + clusterName,
			Namespace: clusterName,
		},
		Spec: workv1.ManifestWorkSpec{
			DeleteOption: &workv1.DeleteOption{
				PropagationPolicy: workv1.DeletePropagationPolicyTypeSelectivelyOrphan,
				SelectivelyOrphan: &workv1.SelectivelyOrphan{
					OrphaningRules: orphaningRules,
				},
			},
			Workload: workv1.ManifestsTemplate{
				Manifests: manifests,
			},
		},
	}
	return manifestWork, nil
}

const NS = `
apiVersion: v1
kind: Namespace
metadata:
  name: installplan-approver
`

const ClusterRole = `
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: installplan-approver
rules:
  -
    apiGroups:
      - operators.coreos.com
    resources:
      - installplans
      - subscriptions
    verbs:
      - get
      - list
      - patch
`
const ClusterRoleBinding = `
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: installplan-approver
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: installplan-approver
subjects:
  -
    kind: ServiceAccount
    name: installplan-approver
    namespace: installplan-approver
`
const ServiceAccount = `
apiVersion: v1
kind: ServiceAccount
metadata:
  name: installplan-approver
  namespace: installplan-approver
`
const ApproverJob = `
kind: Job
metadata:
  name: installplan-approver
  namespace: installplan-approver
spec:
  manualSelector: true
  selector:
    matchLabels:
      job-name: installplan-approver
  template:
    metadata:
      labels:
        job-name: installplan-approver
    spec:
      containers:
        -
          command:
            - /bin/bash
            - "-c"
            - |
                echo "Continue approving operator installPlan for ${WAITTIME}sec ."
                end=$((SECONDS+$WAITTIME))
                while [ $SECONDS -lt $end ]; do
                  echo "continue for " $SECONDS " - " $end
                  oc get subscriptions.operators.coreos.com -A
                  for subscription in $(oc get subscriptions.operators.coreos.com -A -o jsonpath='{range .items[*]}{.metadata.name}{","}{.metadata.namespace}{"\n"}')
                  do
                    if [ $subscription == "," ]; then
                      continue
                    fi
                    echo "Processing subscription '$subscription'"
                    n=$(echo $subscription | cut -f1 -d,)
                    ns=$(echo $subscription | cut -f2 -d,)
                    installplan=$(oc get subscriptions.operators.coreos.com -n ${ns} --field-selector metadata.name=${n} -o jsonpath='{.items[0].status.installPlanRef.name}')
                    installplanNs=$(oc get subscriptions.operators.coreos.com -n ${ns} --field-selector metadata.name=${n} -o jsonpath='{.items[0].status.installPlanRef.namespace}')
                    echo "Check installplan approved status"
                    oc get installplan $installplan -n $installplanNs -o jsonpath="{.spec.approved}"
                    if [ $(oc get installplan $installplan -n $installplanNs -o jsonpath="{.spec.approved}") == "false" ]; then
                      echo "Approving Subscription $subscription with install plan $installplan"
                      oc patch installplan $installplan -n $installplanNs --type=json -p='[{"op":"replace","path": "/spec/approved", "value": true}]'
                    else
                      echo "Install Plan '$installplan' already approved"
                    fi
                  done
                done
          env:
            -
              name: WAITTIME
              value: "600"
          image: registry.redhat.io/openshift4/ose-cli:latest
          imagePullPolicy: IfNotPresent
          name: installplan-approver
      dnsPolicy: ClusterFirst
      restartPolicy: OnFailure
      serviceAccount: installplan-approver
      serviceAccountName: installplan-approver
      terminationGracePeriodSeconds: 60
`
