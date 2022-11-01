/*
Copyright 2022.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package act

import (
	"context"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/klog"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	ctrlutil "sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	actv1beta1 "github.com/redhat-ztp/managedclusters-lifecycle-operator/apis/act/v1beta1"
	utils "github.com/redhat-ztp/managedclusters-lifecycle-operator/controllers/utils"
)

const mcgActFinalizer = "managedclustergroupacts.act.open-cluster-management-extension.io/finalizer"

// ManagedClusterGroupActReconciler reconciles a ManagedClusterGroupAct object
type ManagedClusterGroupActReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=act.open-cluster-management-extension.io,resources=managedclustergroupacts,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=act.open-cluster-management-extension.io,resources=managedclustergroupacts/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=act.open-cluster-management-extension.io,resources=managedclustergroupacts/finalizers,verbs=update
//+kubebuilder:rbac:groups=view.open-cluster-management.io,resources=managedclusterview,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=action.open-cluster-management.io,resources=managedclusteraction,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the ManagedClusterGroupAct object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.10.0/pkg/reconcile
func (r *ManagedClusterGroupActReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {

	mcgAct := &actv1beta1.ManagedClusterGroupAct{}
	// Set default reconcile for 20sec as managedClusterActoin stays for 60sec
	result := ctrl.Result{RequeueAfter: 20 * time.Second}
	err := r.Get(ctx, req.NamespacedName, mcgAct)

	if err != nil {
		if apierrors.IsNotFound(err) {
			klog.Info("ManagedClusterGroupAct resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		klog.Error("Failed to get ManagedClusterGroupAct ", err)
		return result, err
	} else if mcgAct.GetDeletionTimestamp() != nil && ctrlutil.ContainsFinalizer(mcgAct, mcgActFinalizer) {
		if err := r.finalizeMCGAct(mcgAct); err != nil {
			klog.Error("Failed to finaliz managedClusterGroupAct ", err)
			return result, err
		}
		ctrlutil.RemoveFinalizer(mcgAct, mcgActFinalizer)
		err := r.Update(ctx, mcgAct)
		if err != nil {
			klog.Error("Failed update managedClusterGroupAct finalizer ", err)
			return result, err
		}
	} else {
		klog.Info(mcgAct.Name)
		// Condition; TypeSelected
		clustersCount := r.processSelectedCondition(mcgAct)
		apimeta.SetStatusCondition(&mcgAct.Status.Conditions, utils.GetSelectedCondition(clustersCount))

		// Conditions; Applied, Processing, ActionComplete
		allApplied, allProcessing, allDone := r.processAppliedCondition(ctx, mcgAct)
		if clustersCount > 0 && allApplied {
			apimeta.SetStatusCondition(&mcgAct.Status.Conditions, utils.GetAppliedCondition(metav1.ConditionTrue))
		} else {
			apimeta.SetStatusCondition(&mcgAct.Status.Conditions, utils.GetAppliedCondition(metav1.ConditionFalse))
		}
		if clustersCount > 0 && allProcessing {
			apimeta.SetStatusCondition(&mcgAct.Status.Conditions, utils.GetProcessingCondition(metav1.ConditionTrue))
		} else {
			apimeta.SetStatusCondition(&mcgAct.Status.Conditions, utils.GetProcessingCondition(metav1.ConditionFalse))
		}
		if clustersCount > 0 && allDone {
			apimeta.SetStatusCondition(&mcgAct.Status.Conditions, utils.GetActionsCompleteCondition(metav1.ConditionTrue))
		} else {
			apimeta.SetStatusCondition(&mcgAct.Status.Conditions, utils.GetActionsCompleteCondition(metav1.ConditionFalse))
		}

		if !ctrlutil.ContainsFinalizer(mcgAct, mcgActFinalizer) {
			ctrlutil.AddFinalizer(mcgAct, mcgActFinalizer)
			err = r.Update(ctx, mcgAct)
		} else {
			err = r.Status().Update(ctx, mcgAct)
		}

		if err != nil {
			klog.Error("Failed update managedClusterGroupAct ", err)
			return result, err
		}
	}

	return result, nil
}

func (r *ManagedClusterGroupActReconciler) processSelectedCondition(mcgAct *actv1beta1.ManagedClusterGroupAct) int {
	existingClusters := sets.NewString()
	if len(mcgAct.Status.Clusters) > 0 {
		for _, cluster := range mcgAct.Status.Clusters {
			existingClusters.Insert(cluster.Name)
		}
	}

	addedClusters, deletedClusters := sets.NewString(), sets.NewString()
	var err error
	if mcgAct.Spec.Placement.Name != "" && mcgAct.Spec.Placement.Namespace != "" {
		placement, err := utils.GetPlacement(r.Client, mcgAct.Spec.Placement.Name, mcgAct.Spec.Placement.Namespace)
		if err != nil {
			if apierrors.IsNotFound(err) {
				klog.Info("placement resource not found.")
			}
			klog.Error(err, "Failed to get placement.")
		}
		addedClusters, deletedClusters, err = utils.GetClusters(r.Client, placement, existingClusters)
		if err != nil {
			klog.Error("Cannot get clusters ", err)
		}
		//klog.Info("add clusters: ", addedClusters, "deleted clusters: ", deletedClusters)
	} else if mcgAct.Spec.GenericPlacementFields.ClusterSelector != nil || len(mcgAct.Spec.GenericPlacementFields.Clusters) != 0 {
		_, addedClusters, deletedClusters, err = utils.GetManagedClusterList(r.Client, mcgAct.Spec.GenericPlacementFields, existingClusters)
		if err != nil {
			klog.Error("Cannot get clusters ", err)
		}
	}

	clusters := mcgAct.Status.Clusters
	indx := 0
	if deletedClusters.Len() > 0 {
		for _, cls := range clusters {
			if deletedClusters.Has(cls.Name) {
				//Delete created Views
				for _, view := range mcgAct.Spec.Views {
					err = utils.DeleteMangedClusterView(r.Client, mcgAct.Name+"-"+view.Name, cls.Name)
					if err != nil {
						klog.Error("Faild to delete ", err)
					}
				}
			} else {
				clusters[indx] = cls
				indx++
			}
		}
		clusters = clusters[:indx]
	}

	if addedClusters.Len() > 0 {
		for key, _ := range addedClusters {
			clusters = append(clusters, &actv1beta1.ClusterActState{
				Name: key,
			})
		}
	}

	mcgAct.Status.Clusters = clusters

	return len(mcgAct.Status.Clusters)
}

func (r *ManagedClusterGroupActReconciler) processAppliedCondition(ctx context.Context, mcgAct *actv1beta1.ManagedClusterGroupAct) (bool, bool, bool) {
	if len(mcgAct.Status.Clusters) == 0 {
		mcgAct.Status.Clusters = []*actv1beta1.ClusterActState{}
		return false, false, false
	}

	clusters := mcgAct.Status.Clusters

	addActions, _, actionsMap, actionsStr := utils.GetActions(mcgAct.Status.AppliedActions, mcgAct.Spec.Actions)

	// Delete managedCluserViews removed from mcgAct
	utils.DeleteMangedClusterViews(r.Client, mcgAct, false)
	allApplied, allProcessing, AllDone := true, true, true

	for _, cluster := range clusters {
		actionStatus := ""
		mClusterActions, err := utils.GetMangedClusterActions(r.Client, mcgAct.Name, cluster.Name)
		//klog.Info("len mClusterActions; ", len(mClusterActions), "mcga name; ", mcgAct.Name, "cluster name: ", cluster.Name)
		if err != nil {
			klog.Error("Failed to get mClusterActions: ", err)
		}

		for actionName, _ := range actionsMap {
			if cluster.ActionsStatus == "" || addActions.Has(actionName) {
				mClusterAction, err := utils.CreateAction(mcgAct.Name, cluster.Name, actionsMap[actionName])
				if err != nil {
					klog.Error("Failed to create action", err)
					actionStatus = actionStatus + actionName + "-" + utils.StateNotingApplied + " "
					allApplied = false
					continue
				}
				//klog.Info("Create mClusterAction ", mClusterAction.Name, " cluster; ", cluster.Name)
				err = r.Client.Create(ctx, mClusterAction)
				if err != nil {
					klog.Error("Failed create mClusterAction: ", err)
					actionStatus = actionStatus + actionName + "-" + utils.StateNotingApplied + " "
					allApplied = false
					continue
				}
			} else if mClusterAction, ok := mClusterActions[mcgAct.Name+"-"+actionName]; ok {
				status, reason, err := utils.GetManagedClusterActionStatus(mClusterAction)
				if err != nil {
					klog.Warning("Failed to get action Status: ", err)
					actionStatus = actionStatus + actionName + "-" + reason + " "
				}
				if !status {
					actionStatus = actionStatus + actionName + "-" + reason + " "
				}
			}
		}

		if len(mcgAct.Spec.Actions) == 0 {
			// There are no actions
			cluster.ActionsStatus = utils.StateNotingApplied
			AllDone = false
		} else if cluster.ActionsStatus == "" && actionStatus == "" && len(mClusterActions) > 0 {
			cluster.ActionsStatus = utils.StateApplied
		} else if actionStatus != "" && len(mClusterActions) > 0 {
			cluster.ActionsStatus = utils.StateActionFailed + "; " + actionStatus
			AllDone = false
		} else if actionStatus == "" && len(mClusterActions) > 0 {
			cluster.ActionsStatus = utils.StateActionDone
		}

		mClusterViews, err := utils.GetMangedClusterViews(r.Client, mcgAct.Name, cluster.Name)
		if err != nil {
			klog.Error("Faild to get clusterViews ", err)
			continue
		}

		var applied, processing, notProcessing bool
		notProcessingReasons := ""
		for _, view := range mcgAct.Spec.Views {
			viewName := mcgAct.Name + "-" + view.Name
			if mClusterViews[viewName] == nil {
				managedClusterView, err := utils.CreateView(mcgAct.Name, cluster.Name, view)
				if err != nil {
					klog.Error("Failed to create view", err)
					continue
				}
				//klog.Info("Create managedClusterView ", managedClusterView.Name)
				err = r.Client.Create(ctx, managedClusterView)
				if err != nil {
					klog.Error("Failed create managedClusterView: ", err)
					continue
				}
				applied = true
			} else if mClusterViews[viewName] != nil {
				mClusterView := mClusterViews[viewName]
				status, reason, err := utils.GetManagedClusterViewStatus(mClusterView)
				if err != nil {
					klog.Warning("Failed to get view Status: ", err)
					continue
				}

				if mClusterView.Spec != view.ViewSpec {
					//klog.Info("ViewSpec changed")
					mClusterView.Spec = view.ViewSpec
					err = r.Client.Update(ctx, mClusterView)
					if err != nil {
						klog.Error("Faild to update ManagedClusterView ", err)
					}
					continue
				}

				if status {
					processing = true
				} else {
					notProcessing = true
					notProcessingReasons = notProcessingReasons + view.Name + "-" + reason + " "
				}
			}
		}
		if notProcessing {
			cluster.ViewsStatus = utils.StateViewNotProcessing + " " + notProcessingReasons
			allProcessing = false
		} else if processing {
			cluster.ViewsStatus = utils.StateViewProcessing
		} else if applied {
			cluster.ViewsStatus = utils.StateApplied
		} else {
			cluster.ViewsStatus = utils.StateNotingApplied
			allApplied = false
			allProcessing = false
		}
	}

	mcgAct.Status.AppliedActions = actionsStr
	mcgAct.Status.Clusters = clusters

	return allApplied, allProcessing, AllDone
}

func (r *ManagedClusterGroupActReconciler) finalizeMCGAct(mcgAct *actv1beta1.ManagedClusterGroupAct) error {
	err := utils.DeleteMangedClusterViews(r.Client, mcgAct, true)
	if err != nil {
		klog.Error("Failed to delete mClusterViews ", err)
		return err
	}

	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ManagedClusterGroupActReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&actv1beta1.ManagedClusterGroupAct{}).
		Complete(r)
}
