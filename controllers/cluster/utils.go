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

	ReasonApplied                   = "ManagedClustersUpgradeApplied"
	ReasonOperatorsApplied          = "ManagedClustersOperatorsUpgradeApplied"
	ReasonNotSelected               = "ManagedClustersNotSelected"
	ReasonSelected                  = "ManagedClustersSelected"
	ReasonUpgradeInProgress         = "ManagedClustersUpgradeInProgress"
	ReasonUpgradeOperatorInProgress = "ManagedClustersUpgradeOperatorsInProgress"
	ReasonCanaryUpgradeInProgress   = "ManagedClustersCanaryUpgradeInProgress"
	ReasonUpgradeFailed             = "ManagedClustersUpgradeFailed"
	ReasonUpgradeCanaryFailed       = "ManagedClustersCanaryUpgradeFailed"
	ReasonUpgradeComplete           = "ManagedClustersUpgradeComplete"
	ReasonUpgradeCanaryComplete     = "ManagedClustersCanaryUpgradeComplete"
)

// ConvertLabels converts LabelSelectors to Selectors
func ConvertLabels(labelSelector *metav1.LabelSelector) (labels.Selector, error) {
	if labelSelector != nil {
		selector, err := metav1.LabelSelectorAsSelector(labelSelector)
		if err != nil {
			return labels.Nothing(), err
		}

		return selector, nil
	}

	return labels.Everything(), nil
}

func GetCompleteCondition() metav1.Condition {
	return getCondition(TypeComplete, ReasonUpgradeComplete, "ManagedClsuters upgrade Complete", metav1.ConditionFalse)
}

func GetInProgressCondition() metav1.Condition {
	return getCondition(TypeInProgress, ReasonUpgradeInProgress, "ManagedClsuters upgrade InProgress", metav1.ConditionFalse)
}

func GetAppliedCondition() metav1.Condition {
	return getCondition(TypeApplied, ReasonApplied, "ManagedClsuters upgrade ManifestWork applied", metav1.ConditionFalse)
}

func GetSelectedCondition(numClusters int) metav1.Condition {
	if numClusters > 0 {
		return getCondition(TypeSelected, ReasonSelected,
			fmt.Sprintf("Cluster Selector has match %d ManagedClusters.", numClusters), metav1.ConditionTrue)
	}
	return getCondition(TypeSelected, ReasonNotSelected,
		fmt.Sprintf("Cluster Selector has match %d ManagedClusters.", numClusters), metav1.ConditionFalse)
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

//func GetManagedClusterInfo(kubeclient client.Client, clusterName string) (*clusterinfov1beta1.ManagedClusterInfo, error) {
//	mClusterInfo := &clusterinfov1beta1.ManagedClusterInfo{}
//	err := kubeclient.Get(context.TODO(), client.ObjectKey{Name: clusterName, Namespace: clusterName}, mClusterInfo)
//	if err != nil {
//		return nil, err
//	}
//	return mClusterInfo, nil
//}

//func IsManagedClusterVersionComplete(kubeclient client.Client, clusterName string, version string) (bool, error) {
//	mClusterInfo, err := GetManagedClusterInfo(kubeclient, clusterName)
//	if err != nil {
//		return false, err
//	}
//	for _, versionHistory := range mClusterInfo.Status.DistributionInfo.OCP.VersionHistory {
//		if versionHistory.Version == version {
//			return versionHistory.State == managedClusterv1beta1.CompleteState, nil
//		}
//	}
//	return false, nil
//}
