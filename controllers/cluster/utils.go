package cluster

import (
	"context"
	"fmt"
	managedClusterv1beta1 "github.com/redhat-ztp/managedclusters-lifecycle-operator/apis/cluster/v1beta1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/klog"
	clusterv1 "open-cluster-management.io/api/cluster/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	TypeComplete   = "Complete"
	TypeFailed     = "Failed"
	TypeInProgress = "InProgress"
	TypeSelected   = "Selected"
	TypeApplied    = "Applied"

	TypeCanaryComplete = "CanaryComplete"
	TypeCanaryFailed   = "CanaryFailed"

	ReasonApplied               = "ManagedClustersUpgradeApplied"
	ReasonNotSelected           = "ManagedClustersNotSelected"
	ReasonSelected              = "ManagedClustersSelected"
	ReasonUpgradeInProgress     = "ManagedClustersUpgradeInProgress"
	ReasonUpgradeFailed         = "ManagedClustersUpgradeFailed"
	ReasonUpgradeCanaryFailed   = "ManagedClustersCanaryUpgradeFailed"
	ReasonUpgradeComplete       = "ManagedClustersUpgradeComplete"
	ReasonUpgradeCanaryComplete = "ManagedClustersCanaryUpgradeComplete"
)

func GetFailedCondition(failedCount int, failedNames string) metav1.Condition {
	return getCondition(TypeFailed, ReasonUpgradeFailed, GetFailedConditionMessage(failedCount, failedNames),
		metav1.ConditionTrue)
}

func GetCanaryFailedCondition(failedCount int, failedNames string) metav1.Condition {
	return getCondition(TypeCanaryFailed, ReasonUpgradeCanaryFailed, GetFailedConditionMessage(failedCount, failedNames),
		metav1.ConditionTrue)
}

func GetFailedConditionMessage(failedCount int, failedNames string) string {
	return fmt.Sprintf("ManagedClsuters upgrade failed \ncount: %d \nclusters: %s", failedCount, failedNames)
}

func GetCompleteCondition() metav1.Condition {
	return getCondition(TypeComplete, ReasonUpgradeComplete, "ManagedClsuters upgrade Complete", metav1.ConditionFalse)
}

func GetCanaryCompleteCondition() metav1.Condition {
	return getCondition(TypeCanaryComplete, ReasonUpgradeCanaryComplete, "ManagedClsuters canary upgrade Complete", metav1.ConditionFalse)
}

func GetInProgressCondition() metav1.Condition {
	return getCondition(TypeInProgress, ReasonUpgradeInProgress, "ManagedClsuters upgrade InProgress", metav1.ConditionFalse)
}

func GetAppliedCondition() metav1.Condition {
	return getCondition(TypeApplied, ReasonApplied, "ManagedClsuters upgrade applied", metav1.ConditionFalse)
}

func GetSelectedCondition(numClusters int) metav1.Condition {
	message := "ManagedClsuters upgrade select %d clusters"
	if numClusters > 0 {
		return getCondition(TypeSelected, ReasonSelected,
			fmt.Sprintf(message, numClusters), metav1.ConditionTrue)
	}
	return getCondition(TypeSelected, ReasonNotSelected,
		fmt.Sprintf(message, numClusters), metav1.ConditionFalse)
}

func getCondition(conditionType string, reason string, message string, status metav1.ConditionStatus) metav1.Condition {
	return metav1.Condition{
		Type:               conditionType,
		Reason:             reason,
		Message:            message,
		Status:             status,
		LastTransitionTime: metav1.Now(),
	}
}

// ConvertLabels converts LabelSelectors to Selectors
func ConvertLabels(labelSelector *metav1.LabelSelector) (labels.Selector, error) {
	if labelSelector != nil {
		selector, err := metav1.LabelSelectorAsSelector(labelSelector)
		if err != nil {
			return labels.Nothing(), err
		}

		return selector, nil
	}

	return labels.Nothing(), nil
}

// Get ManagedCluster list based on the given label selector
func GetManagedClusterList(kubeclient client.Client, placement managedClusterv1beta1.GenericPlacementFields) (map[string]*clusterv1.ManagedCluster, error) {
	mClusterMap := make(map[string]*clusterv1.ManagedCluster)

	var labelSelector *metav1.LabelSelector

	// Clusters are always labeled with name
	if len(placement.Clusters) != 0 {
		namereq := metav1.LabelSelectorRequirement{}
		namereq.Key = "name"
		namereq.Operator = metav1.LabelSelectorOpIn

		for _, cl := range placement.Clusters {
			namereq.Values = append(namereq.Values, cl.Name)
		}

		labelSelector = &metav1.LabelSelector{
			MatchExpressions: []metav1.LabelSelectorRequirement{namereq},
		}
	} else {
		labelSelector = placement.ClusterSelector
	}

	clSelector, err := ConvertLabels(labelSelector)

	if err != nil {
		return nil, err
	}

	klog.V(1).Info("Using Cluster LabelSelector ", clSelector)

	mClusterlist := &clusterv1.ManagedClusterList{}

	err = kubeclient.List(context.TODO(), mClusterlist, &client.ListOptions{LabelSelector: clSelector})

	if err != nil && !errors.IsNotFound(err) {
		//klog.Error("Listing clusters: ", err)
		return nil, err
	}

	for _, mClusterInfo := range mClusterlist.Items {
		// the cluster will not be returned if it is in terminating process
		if mClusterInfo.DeletionTimestamp != nil && !mClusterInfo.DeletionTimestamp.IsZero() {
			continue
		}

		// reduce memory consumption by cleaning up ManagedFields, Annotations, Finalizers for each managedClusterInfo
		mClusterInfo.DeepCopy().SetManagedFields(nil)
		mClusterInfo.DeepCopy().SetAnnotations(nil)
		mClusterInfo.DeepCopy().SetFinalizers(nil)

		mClusterMap[mClusterInfo.Name] = mClusterInfo.DeepCopy()
	}

	return mClusterMap, nil
}
