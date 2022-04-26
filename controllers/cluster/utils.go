package cluster

import (
	"context"
	clusterinfov1beta1 "github.com/open-cluster-management/multicloud-operators-foundation/pkg/apis/internal.open-cluster-management.io/v1beta1"
	managedClusterv1beta1 "github.com/redhat-ztp/managedclusters-lifecycle-operator/apis/cluster/v1beta1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	TypeComplete   = "Complete"
	TypeFailed     = "Failed"
	TypeInProgress = "InProgress"
	TypeSelected   = "Selected"
	TypeApplying   = "Applying"

	ReasonNotSelected           = "ManagedClustersNotSelected"
	ReasonSelected              = "ManagedClustersSelected"
	ReasonUpgradeProgress       = "ManagedClustersUpgradeInProgress"
	ReasonCanaryUpgradeProgress = "ManagedClustersCanaryUpgradeInProgress"
	ReasonUpgradeFailed         = "ManagedClustersUpgradeFailed"
	ReasonUpgradeCanaryFailed   = "ManagedClustersCanaryUpgradeFailed"
	ReasonUpgradeComplete       = "ManagedClustersUpgradeComplete"
	ReasonUpgradeCanaryComplete = "ManagedClustersCanaryUpgradeComplete"
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

// Get ManagedClusterInfo based on the given label selector
func GetManagedClusterInfos(kubeclient client.Client, placement managedClusterv1beta1.GenericPlacementFields) (map[string]*clusterinfov1beta1.ManagedClusterInfo, error) {
	mClusterInfoMap := make(map[string]*clusterinfov1beta1.ManagedClusterInfo)

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

	mClusterInfolist := &clusterinfov1beta1.ManagedClusterInfoList{}

	err = kubeclient.List(context.TODO(), mClusterInfolist, &client.ListOptions{LabelSelector: clSelector})

	if err != nil && !errors.IsNotFound(err) {
		//klog.Error("Listing clusters: ", err)
		return nil, err
	}

	for _, mClusterInfo := range mClusterInfolist.Items {
		// the cluster will not be returned if it is in terminating process
		if mClusterInfo.DeletionTimestamp != nil && !mClusterInfo.DeletionTimestamp.IsZero() {
			continue
		}

		// reduce memory consumption by cleaning up ManagedFields, Annotations, Finalizers for each managedClusterInfo
		mClusterInfo.DeepCopy().SetManagedFields(nil)
		mClusterInfo.DeepCopy().SetAnnotations(nil)
		mClusterInfo.DeepCopy().SetFinalizers(nil)

		mClusterInfoMap[mClusterInfo.Name] = mClusterInfo.DeepCopy()
	}

	return mClusterInfoMap, nil
}
