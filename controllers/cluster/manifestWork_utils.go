package cluster

import (
	"context"
	"fmt"
	configv1 "github.com/openshift/api/config/v1"
	clusterv1beta1 "github.com/redhat-ztp/managedclusters-lifecycle-operator/apis/cluster/v1beta1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	//"k8s.io/klog"
	workv1 "open-cluster-management.io/api/work/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"
	"time"
)

const (
	ClusterUpgradeManifestName     = "-cluster-upgrade"
	OperatorUpgradeManifestName    = "-operators-upgrade"
	OperatorUpgradeSucceededState  = "succeeded"
	OperatorUpgradeActiveState     = "active"
	OperatorUpgradeFailedState     = "failed"
	OperatorUpgradeValidStateValue = 1
	OseCliDefaultImage             = "registry.redhat.io/openshift4/ose-cli:latest"
)

func CreateOperatorUpgradeManifestWork(clusterName string, operatorConfig *clusterv1beta1.OcpOperatorsSpec, timeout string, image string) (*workv1.ManifestWork, error) {
	if clusterName == "" {
		return nil, fmt.Errorf("Invalid clusterName")
	}

	if operatorConfig == nil {
		return nil, fmt.Errorf("Invalid operatorConfig")
	}

	if !operatorConfig.ApproveAllUpgrades && len(operatorConfig.Include) == 0 {
		return nil, fmt.Errorf("Invalid operatorConfig at least an operator should be selected to upgrade")
	}

	job, err := getInstallPlanApproverJob(operatorConfig.Include, operatorConfig.Exclude, timeout, image)
	if err != nil {
		return nil, err
	}

	manifests := []workv1.Manifest{
		workv1.Manifest{
			RawExtension: runtime.RawExtension{Raw: getJsonFromYaml(NS)},
		},
		workv1.Manifest{
			RawExtension: runtime.RawExtension{Raw: getJsonFromYaml(ClusterRole)},
		},
		workv1.Manifest{
			RawExtension: runtime.RawExtension{Raw: getJsonFromYaml(ClusterRoleBinding)},
		},
		workv1.Manifest{
			RawExtension: runtime.RawExtension{Raw: getJsonFromYaml(ServiceAccount)},
		},
		workv1.Manifest{
			RawExtension: runtime.RawExtension{Object: job},
		},
	}

	// Create manifest Config option to watch cluster version upgrade status
	manifestConfigOpt := workv1.ManifestConfigOption{
		ResourceIdentifier: workv1.ResourceIdentifier{
			Group:     "batch",
			Name:      "installplan-approver",
			Namespace: "installplan-approver",
			Resource:  "jobs",
		},
		FeedbackRules: []workv1.FeedbackRule{
			workv1.FeedbackRule{
				Type: workv1.FeedBackType("JSONPaths"),
				JsonPaths: []workv1.JsonPath{
					workv1.JsonPath{
						Name: OperatorUpgradeSucceededState,
						Path: ".status." + OperatorUpgradeSucceededState,
					},
					workv1.JsonPath{
						Name: OperatorUpgradeActiveState,
						Path: ".status." + OperatorUpgradeActiveState,
					},
					workv1.JsonPath{
						Name: OperatorUpgradeFailedState,
						Path: ".status." + OperatorUpgradeFailedState,
					},
				},
			},
		},
	}

	manifestWork := &workv1.ManifestWork{
		ObjectMeta: metav1.ObjectMeta{
			Name:      clusterName + OperatorUpgradeManifestName,
			Namespace: clusterName,
		},
		Spec: workv1.ManifestWorkSpec{
			DeleteOption: &workv1.DeleteOption{
				PropagationPolicy: workv1.DeletePropagationPolicyTypeForeground,
			},
			Workload: workv1.ManifestsTemplate{
				Manifests: manifests,
			},
			ManifestConfigs: []workv1.ManifestConfigOption{manifestConfigOpt},
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
		TypeMeta: metav1.TypeMeta{
			APIVersion: "config.openshift.io/v1",
			Kind:       "ClusterVersion",
		},
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
		TypeMeta: metav1.TypeMeta{
			APIVersion: "rbac.authorization.k8s.io/v1",
			Kind:       "ClusterRoleBinding",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        "admin-ocm",
			Annotations: map[string]string{"rbac.authorization.kubernetes.io/autoupdate": "true"},
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

	// TODO: Add failed condition to watch failed state
	// Create manifest Config option to watch cluster version upgrade status
	manifestConfigOpt := workv1.ManifestConfigOption{
		ResourceIdentifier: workv1.ResourceIdentifier{
			Group:    "config.openshift.io",
			Name:     "version",
			Resource: "clusterversions",
		},
		FeedbackRules: []workv1.FeedbackRule{
			workv1.FeedbackRule{
				Type: workv1.FeedBackType("JSONPaths"),
				JsonPaths: []workv1.JsonPath{
					workv1.JsonPath{
						Name: "version",
						Path: ".status.history[0].version",
					},
					workv1.JsonPath{
						Name: "state",
						Path: ".status.history[0].state",
					},
					workv1.JsonPath{
						Name: "verified",
						Path: ".status.history[0].verified",
					},
				},
			},
		},
	}

	manifestWork := &workv1.ManifestWork{
		ObjectMeta: metav1.ObjectMeta{
			Name:      clusterName + ClusterUpgradeManifestName,
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
			ManifestConfigs: []workv1.ManifestConfigOption{manifestConfigOpt},
		},
	}
	return manifestWork, nil
}

func getJsonFromYaml(yamlStr string) []byte {
	if json, err := yaml.YAMLToJSON([]byte(yamlStr)); err == nil {
		return json
	}
	return nil
}

func getManifestWork(kubeclient client.Client, name string, ns string, timeOut string) (*workv1.ManifestWork, bool, error) {
	isTimeOut := false
	manifestWork := &workv1.ManifestWork{}
	err := kubeclient.Get(context.TODO(), client.ObjectKey{
		Name:      name,
		Namespace: ns,
	}, manifestWork)
	if err != nil {
		return nil, isTimeOut, err
	}
	if timeOut != "" {
		mwTimeOut, err := time.ParseDuration(timeOut)
		if err != nil {
			return manifestWork, isTimeOut, err
		}
		// Add 5 min timeOut to avoid feedback delay or overlap timeout less than Reconcile every 5min
		//min := time.Duration(mwTimeOut.Minutes() + float64(5))
		timeRemaining := &metav1.Time{
			Time: manifestWork.CreationTimestamp.Add(mwTimeOut),
		}

		timeNow := metav1.Now()
		isTimeOut = !timeNow.Before(timeRemaining)
		//klog.Info("mwTimeOut= ", mwTimeOut, " manifestWork.CreationTimestamp= ", manifestWork.CreationTimestamp,
		//	" timeRemaining= ", timeRemaining, " timeNow= ", timeNow, " isTimeOut= ", isTimeOut)
	}
	return manifestWork, isTimeOut, nil
}

func isManifestWorkResourcesAvailable(kubeclient client.Client, name string, ns string, timeOut string) (bool, bool, error) {
	manifestwork, isTimeOut, err := getManifestWork(kubeclient, name, ns, timeOut)
	if err != nil {
		return false, isTimeOut, err
	}

	return apimeta.IsStatusConditionTrue(manifestwork.Status.Conditions, workv1.WorkAvailable), isTimeOut, nil
}

// Get the clusterVersion Upgrade  status (value); state, version, verified
func getClusterUpgradeManifestStatus(kubeclient client.Client, name string, ns string, timeOut string) (string, string, bool, bool, error) {
	version, state := "", ""
	verified := false
	manifestwork, isTimeOut, err := getManifestWork(kubeclient, name, ns, timeOut)
	if err != nil {
		return version, state, verified, isTimeOut, err
	}

	for _, manifest := range manifestwork.Status.ResourceStatus.Manifests {
		if manifest.ResourceMeta.Kind == "ClusterVersion" {
			for _, v := range manifest.StatusFeedbacks.Values {
				if v.Name == "version" {
					version = string(*v.Value.String)
				} else if v.Name == "state" {
					state = string(*v.Value.String)
				} else if v.Name == "verified" {
					verified = bool(*v.Value.Boolean)
				}
			}
		}
	}
	return version, state, verified, isTimeOut, err
}

// Get the Operator Upgrade Job status (name, value); active, succeeded or failed
func getOperatorUpgradeManifestStatus(kubeclient client.Client, name string, ns string, timeOut string) (string, int, bool, error) {
	manifestwork, isTimeOut, err := getManifestWork(kubeclient, name, ns, timeOut)
	if err != nil {
		return "", 0, isTimeOut, err
	}
	for _, manifest := range manifestwork.Status.ResourceStatus.Manifests {
		if manifest.ResourceMeta.Kind == "Job" {
			if len(manifest.StatusFeedbacks.Values) < 1 {
				return "", 0, isTimeOut, fmt.Errorf("Operator Upgrade job status not found %s", name)
			}
			return manifest.StatusFeedbacks.Values[0].Name, int(*manifest.StatusFeedbacks.Values[0].Value.Integer), isTimeOut, nil
		}
	}
	return "", 0, isTimeOut, fmt.Errorf("Operator Upgrade job not found %s", name)
}

func deleteManifestWork(kubeclient client.Client, name string, ns string) error {
	manifestWork := &workv1.ManifestWork{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: ns,
		},
	}

	return kubeclient.Delete(context.TODO(), manifestWork)
}

func getInstallPlanApproverJob(includeOperator []clusterv1beta1.GenericOperatorReference, excludeOperators []clusterv1beta1.GenericOperatorReference, upgradeTimeout string, oseCliImage string) (*batchv1.Job, error) {
	jobTimeOut, err := time.ParseDuration(upgradeTimeout)
	if err != nil {
		return nil, fmt.Errorf("Invalid Operators upgrade timeout %s", upgradeTimeout)
	}

	sec := int64(jobTimeOut.Seconds())
	// Adding 90sec to the timeout to terminate the job if it didn't reach succeeded state
	deadLine := sec + 90
	includeOperatorStr, excludeOperatorsStr := "", ""
	manualSelect := true

	for _, ref := range includeOperator {
		includeOperatorStr = ref.Name + "," + ref.Namespace
	}

	for _, ref := range excludeOperators {
		excludeOperatorsStr = ref.Name + "," + ref.Namespace
	}

	job := &batchv1.Job{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "batch/v1",
			Kind:       "Job",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "installplan-approver",
			Namespace: "installplan-approver",
		},
		Spec: batchv1.JobSpec{
			ManualSelector:        &manualSelect,
			ActiveDeadlineSeconds: &deadLine,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"job-name": "installplan-approver"},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{"job-name": "installplan-approver"},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:            "installplan-approver",
							Image:           oseCliImage,
							ImagePullPolicy: corev1.PullIfNotPresent,
							Env: []corev1.EnvVar{
								{
									Name:  "EXCLUDELIST",
									Value: excludeOperatorsStr,
								},
								{
									Name:  "INCLUDELIST",
									Value: includeOperatorStr,
								},
								{
									Name:  "WAITTIME",
									Value: fmt.Sprintf("%d", sec),
								},
							},
							Command: []string{"/bin/bash", "-c", JobScript},
						},
					},
					DNSPolicy:          corev1.DNSClusterFirst,
					RestartPolicy:      corev1.RestartPolicyOnFailure,
					ServiceAccountName: "installplan-approver",
				},
			},
		},
	}
	return job, nil
}

const JobScript = `
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

    if [[ $INCLUDELIST != "" ]]; then
      if [[ $INCLUDELIST != *$subscription* ]]; then
        continue
      else
        echo "Subscription $subscription Included"
      fi
    fi

    if [[ $EXCLUDELIST != "" ]]; then
      if [[ $EXCLUDELIST == *$subscription* ]]; then
        echo "Subscription $subscription Excluded"
        continue
      fi
    fi

    echo "Processing subscription '$subscription'"
    n=$(echo $subscription | cut -f1 -d,)
    ns=$(echo $subscription | cut -f2 -d,)

    installplan=$(oc get subscriptions.operators.coreos.com -n ${ns} --field-selector metadata.name=${n} -o jsonpath='{.items[0].status.installPlanRef.name}')
    installplanNs=$(oc get subscriptions.operators.coreos.com -n ${ns} --field-selector metadata.name=${n} -o jsonpath='{.items[0].status.installPlanRef.namespace}')
    echo "Check installplan approved status"
    oc get installplan $installplan -n $installplanNs -o jsonpath="{.spec.approved}"
    if [ $(oc get installplan $installplan -n $installplanNs -o jsonpath="{.spec.approved}") == "false" ]; then
      echo " - Approving Subscription $subscription with install plan $installplan"
      oc patch installplan $installplan -n $installplanNs --type=json -p='[{"op":"replace","path": "/spec/approved", "value": true}]'
    else
      echo " - Install Plan '$installplan' already approved"
    fi
  done
done
`
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
