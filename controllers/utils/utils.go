package utils

import (
	"context"
	"fmt"
	actv1beta1 "github.com/redhat-ztp/managedclusters-lifecycle-operator/apis/act/v1beta1"
	common "github.com/redhat-ztp/managedclusters-lifecycle-operator/apis/common/v1beta1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/sets"
	"strings"

	actionv1beta1 "github.com/stolostron/cluster-lifecycle-api/action/v1beta1"
	viewv1beta1 "github.com/stolostron/cluster-lifecycle-api/view/v1beta1"
	"k8s.io/klog"
	clusterv1 "open-cluster-management.io/api/cluster/v1"
	clusterv1beta1 "open-cluster-management.io/api/cluster/v1beta1"
	workv1 "open-cluster-management.io/api/work/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	TypeSelected       = "Selected"
	TypeApplied        = "Applied"
	TypeInProgress     = "InProgress"
	TypeComplete       = "Complete"
	TypeFailed         = "Failed"
	TypeCanaryComplete = "CanaryComplete"
	TypeCanaryFailed   = "CanaryFailed"
	TypeCreated        = "Created"
	//
	ReasonApplied               = "ManagedClustersResourcesApplied"
	ReasonNotSelected           = "ManagedClustersNotSelected"
	ReasonSelected              = "ManagedClustersSelected"
	ReasonUpgradeInProgress     = "ManagedClustersUpgradeInProgress"
	ReasonUpgradeFailed         = "ManagedClustersUpgradeFailed"
	ReasonUpgradeCanaryFailed   = "ManagedClustersCanaryUpgradeFailed"
	ReasonUpgradeComplete       = "ManagedClustersUpgradeComplete"
	ReasonUpgradeCanaryComplete = "ManagedClustersCanaryUpgradeComplete"
	ReasonActionsDone           = "ActionsDone"
	ReasonCreated               = "ManifestWorkCreated"
	//
	StateApplied           = "Applied"
	StateNotingApplied     = "NotingApplied"
	StateActionDone        = "ActionsDone"
	StateActionFailed      = "ActionsFailed"
	StateViewProcessing    = "ViewProcessing"
	StateViewNotProcessing = "ViewNotProcessing"
	StatusNotFound         = "StatusNotFound"
	//
	MCGAct_Label   = "managedClusterGroupAct"
	ViewName_Label = "viewName"
)

type PlacementDecisionGetter struct {
	Client client.Client
}

func (pdl PlacementDecisionGetter) List(selector labels.Selector, namespace string) ([]*clusterv1beta1.PlacementDecision, error) {
	decisionList := clusterv1beta1.PlacementDecisionList{}
	err := pdl.Client.List(context.Background(), &decisionList, &client.ListOptions{
		Namespace:     namespace,
		LabelSelector: selector})
	if err != nil {
		return nil, err
	}
	var decisions []*clusterv1beta1.PlacementDecision
	for i := range decisionList.Items {
		decisions = append(decisions, &decisionList.Items[i])
	}
	return decisions, nil
}

func GetPlacement(kubeclient client.Client, name string, ns string) (*clusterv1beta1.Placement, error) {
	placement := &clusterv1beta1.Placement{}
	err := kubeclient.Get(context.TODO(), client.ObjectKey{
		Name:      name,
		Namespace: ns,
	}, placement)

	return placement, err
}

func GetClusters(kubeclient client.Client, placement *clusterv1beta1.Placement, existingClusters sets.String) (sets.String, sets.String, error) {
	pdtracker := clusterv1beta1.NewPlacementDecisionClustersTracker(placement, PlacementDecisionGetter{Client: kubeclient}, existingClusters)

	return pdtracker.Get()
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
func GetManagedClusterList(kubeclient client.Client, placement common.GenericPlacementFields, existingClusters sets.String) (map[string]*clusterv1.ManagedCluster, sets.String, sets.String, error) {
	mClusterMap := make(map[string]*clusterv1.ManagedCluster)
	addedClusters, deletedClusters := sets.NewString(), sets.NewString()

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
		return nil, addedClusters, deletedClusters, err
	}

	mClusterlist := &clusterv1.ManagedClusterList{}
	err = kubeclient.List(context.TODO(), mClusterlist, &client.ListOptions{LabelSelector: clSelector})

	if err != nil && !errors.IsNotFound(err) {
		return nil, addedClusters, deletedClusters, err
	}

	for _, mCluster := range mClusterlist.Items {
		// the cluster will not be returned if it is in terminating process
		if mCluster.DeletionTimestamp != nil && !mCluster.DeletionTimestamp.IsZero() {
			continue
		}

		// reduce memory consumption by cleaning up ManagedFields, Annotations, Finalizers for each managedCluster
		mCluster.DeepCopy().SetManagedFields(nil)
		mCluster.DeepCopy().SetAnnotations(nil)
		mCluster.DeepCopy().SetFinalizers(nil)

		mClusterMap[mCluster.Name] = mCluster.DeepCopy()

		if !existingClusters.Has(mCluster.Name) {
			addedClusters.Insert(mCluster.Name)
		}
	}

	for mCluster, _ := range existingClusters {
		if _, ok := mClusterMap[mCluster]; !ok {
			deletedClusters.Insert(mCluster)
		}
	}

	return mClusterMap, addedClusters, deletedClusters, nil
}

func GetSelectedCondition(numClusters int) metav1.Condition {
	message := "ManagedClsuters select %d clusters"
	if numClusters > 0 {
		return getCondition(TypeSelected, ReasonSelected,
			fmt.Sprintf(message, numClusters), metav1.ConditionTrue)
	}
	return getCondition(TypeSelected, ReasonNotSelected,
		fmt.Sprintf(message, numClusters), metav1.ConditionFalse)
}

func GetCreatedCondition(status metav1.ConditionStatus) metav1.Condition {
	return getCondition(TypeCreated, ReasonCreated, "ManagedClusters ManifestWork created", status)
}

func GetAppliedCondition(status metav1.ConditionStatus) metav1.Condition {
	return getCondition(TypeApplied, ReasonApplied, "ManagedClsuters Resources applied", status)
}

func GetAvailableCondition(status metav1.ConditionStatus) metav1.Condition {
	return getCondition(workv1.WorkAvailable, workv1.WorkAvailable, "ManagedClusters ManifestWork available", status)
}

func GetActionsCompleteCondition(status metav1.ConditionStatus) metav1.Condition {
	return getCondition(actionv1beta1.ConditionActionCompleted, ReasonActionsDone, "", status)
}

func GetProcessingCondition(status metav1.ConditionStatus) metav1.Condition {
	return getCondition(viewv1beta1.ConditionViewProcessing, viewv1beta1.ReasonGetResource, "", status)
}

func GetFailedCondition(failedCount int) metav1.Condition {
	return getCondition(TypeFailed, ReasonUpgradeFailed, GetFailedConditionMessage(failedCount),
		metav1.ConditionTrue)
}

func GetCanaryFailedCondition(failedCount int) metav1.Condition {
	return getCondition(TypeCanaryFailed, ReasonUpgradeCanaryFailed, GetFailedConditionMessage(failedCount),
		metav1.ConditionTrue)
}

func GetFailedConditionMessage(failedCount int) string {
	return fmt.Sprintf("ManagedClsuters upgrade failed \ncount: %d", failedCount)
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

func getCondition(conditionType string, reason string, message string, status metav1.ConditionStatus) metav1.Condition {
	return metav1.Condition{
		Type:               conditionType,
		Reason:             reason,
		Message:            message,
		Status:             status,
		LastTransitionTime: metav1.Now(),
	}
}

func CreateView(mcgaName string, namespace string, viewSpec actv1beta1.View) (*viewv1beta1.ManagedClusterView, error) {
	if mcgaName == "" || namespace == "" || viewSpec.Name == "" {
		return nil, fmt.Errorf("Invalid namespace or name")
	}

	view := &viewv1beta1.ManagedClusterView{
		ObjectMeta: metav1.ObjectMeta{
			Name:      mcgaName + "-" + viewSpec.Name,
			Namespace: namespace,
			Labels:    map[string]string{MCGAct_Label: mcgaName, ViewName_Label: viewSpec.Name},
		},
		Spec: viewSpec.ViewSpec,
	}

	return view, nil
}

func GetManagedClusterViewStatus(mClusteView *viewv1beta1.ManagedClusterView) (bool, string, error) {
	for _, condition := range mClusteView.Status.Conditions {
		if condition.Type == viewv1beta1.ConditionViewProcessing {
			return condition.Status == metav1.ConditionTrue, condition.Reason, nil
		}
	}

	return false, "", fmt.Errorf("ConditionViewProcessing not found")
}

func GetMangedClusterViews(kubeclient client.Client, mcgaName string, namespace string) (map[string]*viewv1beta1.ManagedClusterView, error) {
	clSelector, err := ConvertLabels(&metav1.LabelSelector{MatchLabels: map[string]string{MCGAct_Label: mcgaName}})

	if err != nil {
		return nil, err
	}

	mClusterViewlist := &viewv1beta1.ManagedClusterViewList{}
	err = kubeclient.List(context.TODO(), mClusterViewlist, &client.ListOptions{LabelSelector: clSelector, Namespace: namespace})

	if err != nil && !errors.IsNotFound(err) {
		return nil, err
	}

	mClusterViewMap := make(map[string]*viewv1beta1.ManagedClusterView)
	for _, mClusterView := range mClusterViewlist.Items {
		// the mClusterView will not be returned if it is in terminating process
		if mClusterView.DeletionTimestamp != nil && !mClusterView.DeletionTimestamp.IsZero() {
			continue
		}

		// reduce memory consumption by cleaning up ManagedFields, Annotations, Finalizers for each managedCluster
		mClusterView.DeepCopy().SetManagedFields(nil)
		mClusterView.DeepCopy().SetAnnotations(nil)
		mClusterView.DeepCopy().SetFinalizers(nil)

		mClusterViewMap[mClusterView.Name] = mClusterView.DeepCopy()
	}

	return mClusterViewMap, nil
}

func DeleteMangedClusterView(kubeclient client.Client, name string, namespace string) error {
	if name == "" || namespace == "" {
		return fmt.Errorf("Invalid namespace or name")
	}

	return kubeclient.Delete(context.TODO(), &viewv1beta1.ManagedClusterView{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace}})
}

func DeleteMangedClusterViews(kubeclient client.Client, mcgAct *actv1beta1.ManagedClusterGroupAct, all bool) error {
	var ownerSelector labels.Selector
	var err error
	if len(mcgAct.Spec.Views) == 0 || all {
		ownerSelector, err = ConvertLabels(&metav1.LabelSelector{MatchLabels: map[string]string{MCGAct_Label: mcgAct.Name}})
	} else {
		viewsNames := make([]string, len(mcgAct.Spec.Views))
		for id, view := range mcgAct.Spec.Views {
			viewsNames[id] = view.Name
		}
		namereq := metav1.LabelSelectorRequirement{}
		namereq.Key = ViewName_Label
		namereq.Operator = metav1.LabelSelectorOpNotIn
		namereq.Values = viewsNames
		ownerSelector, err = ConvertLabels(&metav1.LabelSelector{MatchExpressions: []metav1.LabelSelectorRequirement{namereq},
			MatchLabels: map[string]string{MCGAct_Label: mcgAct.Name}})
	}

	if err != nil {
		klog.Error("label selector ", err)
		return err
	}

	mClusterViewlist := &viewv1beta1.ManagedClusterViewList{}
	err = kubeclient.List(context.TODO(), mClusterViewlist, &client.ListOptions{LabelSelector: ownerSelector})

	if err != nil && !errors.IsNotFound(err) {
		return err
	}

	for _, mcView := range mClusterViewlist.Items {
		err = DeleteMangedClusterView(kubeclient, mcView.Name, mcView.Namespace)
		if err != nil {
			klog.Error("Faild to delete ", err)
		}
	}

	return nil
}

func CreateAction(mcgaName string, namespace string, actionSpec actv1beta1.Action) (*actionv1beta1.ManagedClusterAction, error) {
	if namespace == "" || actionSpec.Name == "" {
		return nil, fmt.Errorf("Invalid namespace or name")
	}

	action := &actionv1beta1.ManagedClusterAction{
		ObjectMeta: metav1.ObjectMeta{
			Name:      mcgaName + "-" + actionSpec.Name,
			Namespace: namespace,
			Labels:    map[string]string{MCGAct_Label: mcgaName},
		},
		Spec: actionSpec.ActionSpec,
	}

	return action, nil
}

func GetMangedClusterActions(kubeclient client.Client, mcgaName string, namespace string) (map[string]*actionv1beta1.ManagedClusterAction, error) {
	clSelector, err := ConvertLabels(&metav1.LabelSelector{MatchLabels: map[string]string{MCGAct_Label: mcgaName}})

	if err != nil {
		return nil, err
	}
	mClusterActionlist := &actionv1beta1.ManagedClusterActionList{}
	err = kubeclient.List(context.TODO(), mClusterActionlist, &client.ListOptions{LabelSelector: clSelector, Namespace: namespace})

	if err != nil && !errors.IsNotFound(err) {
		return nil, err
	}
	mClusterActionMap := make(map[string]*actionv1beta1.ManagedClusterAction)

	for _, mClusterAction := range mClusterActionlist.Items {
		// the mClusterAction will not be returned if it is in terminating process
		if mClusterAction.DeletionTimestamp != nil && !mClusterAction.DeletionTimestamp.IsZero() {
			continue
		}

		// reduce memory consumption by cleaning up ManagedFields, Annotations, Finalizers for each managedCluster
		mClusterAction.DeepCopy().SetManagedFields(nil)
		mClusterAction.DeepCopy().SetAnnotations(nil)
		mClusterAction.DeepCopy().SetFinalizers(nil)

		mClusterActionMap[mClusterAction.Name] = mClusterAction.DeepCopy()
	}

	return mClusterActionMap, nil
}

func GetManagedClusterActionStatus(mClusteAction *actionv1beta1.ManagedClusterAction) (bool, string, error) {
	if mClusteAction == nil {
		return false, "", fmt.Errorf("ManagedClusterAction is nil")
	}

	for _, condition := range mClusteAction.Status.Conditions {
		if condition.Type == actionv1beta1.ConditionActionCompleted {
			return condition.Status == metav1.ConditionTrue, condition.Reason, nil
		}
	}

	return false, StatusNotFound, fmt.Errorf("ConditionActionComplete not found")
}

func GetActions(appliedActions string, newActions []actv1beta1.Action) (sets.String, sets.String, map[string]actv1beta1.Action, string) {
	appliedActionsSet, newActionsSet := sets.NewString(), sets.NewString()
	newActionsMap := make(map[string]actv1beta1.Action)

	for _, action := range strings.Split(appliedActions, " ") {
		if strings.TrimSpace(action) == "" {
			continue
		}
		appliedActionsSet.Insert(action)
	}

	actionsStr := ""
	for _, action := range newActions {
		newActionsMap[action.Name] = action
		newActionsSet.Insert(action.Name)
		actionsStr = actionsStr + action.Name + " "
	}

	addActions := newActionsSet.Difference(appliedActionsSet)
	RemovedActions := appliedActionsSet.Difference(newActionsSet)

	return addActions, RemovedActions, newActionsMap, actionsStr
}

//func CreatePlacement(name string, ns string, selector managedClusterv1beta1.GenericPlacementFields) (*clusterv1beta1.Placement, error) {
//	var labelSelector *metav1.LabelSelector
//
//	if name == "" {
//		return nil, fmt.Errorf("Invalid placement name")
//	}
//	if ns == "" {
//		return nil, fmt.Errorf("Invalid placement namespace")
//	}
//
//	// Clusters are always labeled with name
//	if len(selector.Clusters) != 0 {
//		namereq := metav1.LabelSelectorRequirement{}
//		namereq.Key = "name"
//		namereq.Operator = metav1.LabelSelectorOpIn
//
//		for _, cl := range selector.Clusters {
//			namereq.Values = append(namereq.Values, cl.Name)
//		}
//
//		labelSelector = &metav1.LabelSelector{
//			MatchExpressions: []metav1.LabelSelectorRequirement{namereq},
//		}
//	} else {
//		labelSelector = selector.ClusterSelector
//	}
//
//	clusterPredicate := clusterv1beta1.ClusterPredicate{
//		RequiredClusterSelector: clusterv1beta1.ClusterSelector{
//			LabelSelector: *labelSelector,
//		},
//	}
//
//	placement := &clusterv1beta1.Placement{
//		ObjectMeta: metav1.ObjectMeta{
//			Name:      name + PlacementPrefix,
//			Namespace: ns,
//		},
//		Spec: clusterv1beta1.PlacementSpec{
//			ClusterSets: selector.ClusterSets,
//			Predicates:  []clusterv1beta1.ClusterPredicate{clusterPredicate},
//		},
//	}
//
//	return placement, nil
//}
