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

package cluster

import (
	"context"
	//"fmt"

	//clusterinfov1beta1 "github.com/open-cluster-management/multicloud-operators-foundation/pkg/apis/internal.open-cluster-management.io/v1beta1"
	clusterv1beta1 "github.com/redhat-ztp/managedclusters-lifecycle-operator/apis/cluster/v1beta1"
	//clusterv1 "open-cluster-management.io/api/cluster/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/klog"
	//workv1 "open-cluster-management.io/api/work/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	//"sigs.k8s.io/controller-runtime/pkg/log"
)

// ManagedClustersUpgradeReconciler reconciles a ManagedClustersUpgrade object
type ManagedClustersUpgradeReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=cluster.open-cluster-management-extension.io,resources=managedclustersupgrades,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=cluster.open-cluster-management-extension.io,resources=managedclustersupgrades/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=cluster.open-cluster-management-extension.io,resources=managedclustersupgrades/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the ManagedClustersUpgrade object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.10.0/pkg/reconcile
func (r *ManagedClustersUpgradeReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	managedClustersUpgrade := &clusterv1beta1.ManagedClustersUpgrade{}
	err := r.Get(ctx, req.NamespacedName, managedClustersUpgrade)

	if err != nil {
		if errors.IsNotFound(err) {
			klog.Info("ManagedClustersUpgrade resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		klog.Error(err, "Failed to get ManagedClustersUpgrade")
		return ctrl.Result{}, err
	} else {
		//klog.Info(managedClustersUpgrade.Name)

		if condition := apimeta.FindStatusCondition(managedClustersUpgrade.Status.Conditions, TypeComplete); condition != nil {
			//klog.Info("type complete")
		}
		if condition := apimeta.FindStatusCondition(managedClustersUpgrade.Status.Conditions, TypeInProgress); condition != nil {
			klog.Info("type inprogress")
			if condition.Status == metav1.ConditionFalse {
				// Check if the ManifestWork get Applied without errors
				for _, cluster := range managedClustersUpgrade.Status.Clusters {
					if cluster.ClusterVersionStatus.State == clusterv1beta1.InitializedState {

					}
					if cluster.ClusterVersionStatus.State == clusterv1beta1.OperatorsUpgradeState {

					}
				}
				condition.Status = metav1.ConditionTrue
			}
			if condition.Status == metav1.ConditionTrue {
				// Check the ManagedClusterInfo state OR Create/check operatorManifest state
				for _, cluster := range managedClustersUpgrade.Status.Clusters {
					if cluster.ClusterVersionStatus.State == clusterv1beta1.InitializedState {

					}
					if cluster.ClusterVersionStatus.State == clusterv1beta1.PartialState {

					}
					if cluster.ClusterVersionStatus.State == clusterv1beta1.CompleteState {

					}
					if cluster.ClusterVersionStatus.State == clusterv1beta1.OperatorsUpgradeState {

					}
				}
			}
		}
		if condition := apimeta.FindStatusCondition(managedClustersUpgrade.Status.Conditions, TypeApplied); condition != nil {
			klog.Info("type applied")
			if condition.Status == metav1.ConditionFalse {
				//manifestWorks := []*workv1.ManifestWork{}
				manifestCreated := false
				for _, cluster := range managedClustersUpgrade.Status.Clusters {
					manifestWork, err := CreateClusterVersionUpgradeManifestWork(cluster.ClusterID, cluster.Name, managedClustersUpgrade.Spec.ClusterVersion)
					clusterState := clusterv1beta1.InitializedState
					if err != nil {
						klog.Info("ClusterVersion ManifestWork not initaliz: "+cluster.Name+" ", err)

						manifestWork, err = CreateOperatorUpgradeManifestWork(cluster.Name, managedClustersUpgrade.Spec.OcpOperators)
						if err != nil {
							klog.Info("OCP Operators ManifestWork not initaliz: "+cluster.Name+" ", err)
						}
						clusterState = clusterv1beta1.OperatorsUpgradeState
					}

					if manifestWork == nil {
						continue
					}

					if err := controllerutil.SetOwnerReference(managedClustersUpgrade, manifestWork, r.Scheme); err != nil {
						klog.Error("Manifest set Owner Ref: " + cluster.Name + " ", err)
						//return ctrl.Result{}, nil
					}
					//manifestWorks = append(manifestWorks, manifestWork)
					err = r.Client.Create(ctx, manifestWork)
					if err != nil {
						klog.Error("ManifestWork create: " + cluster.Name + " ", err)
						continue
					}
					//klog.Info("ManifestWork Created ", manifestWork)
					cluster.ClusterVersionStatus.State = clusterState
					manifestCreated = true
				}

				if manifestCreated {
					//for _, mw := range manifestWorks {
					//	err = r.Client.Create(ctx, mw)
					//	if err != nil {
					//		klog.Error("ManifestWork create: ", cluster.Name, err)
					//	}
					//	klog.Info("ManifestWork Created ", mw)
					//}
					condition.Status = metav1.ConditionTrue
					apimeta.SetStatusCondition(&managedClustersUpgrade.Status.Conditions, GetInProgressCondition())
				}
			}
		}
		if condition := apimeta.FindStatusCondition(managedClustersUpgrade.Status.Conditions, TypeSelected); condition != nil {
			klog.Info("type selected")
			if condition.Reason == ReasonNotSelected && condition.Status == metav1.ConditionFalse {
				//klog.Info("reason not selected")
				clustersStatus, err := r.initClustersList(r.Client, managedClustersUpgrade.Spec.GenericPlacementFields)
				if err != nil {
					klog.Error("Listing clusters: ", err)
					return ctrl.Result{}, nil
				}

				managedClustersUpgrade.Status.Clusters = clustersStatus
				apimeta.SetStatusCondition(&managedClustersUpgrade.Status.Conditions, GetSelectedCondition(len(clustersStatus)))
			} else if apimeta.FindStatusCondition(managedClustersUpgrade.Status.Conditions, TypeApplied) == nil && condition.Reason == ReasonSelected &&
				condition.Status == metav1.ConditionTrue {
				//klog.Info("add applied condtion from selected condition")
				apimeta.SetStatusCondition(&managedClustersUpgrade.Status.Conditions, GetAppliedCondition())
			}
		}
		// Create Clusters status list
		if apimeta.FindStatusCondition(managedClustersUpgrade.Status.Conditions, TypeSelected) == nil {
			clustersStatus, err := r.initClustersList(r.Client, managedClustersUpgrade.Spec.GenericPlacementFields)
			if err != nil {
				klog.Error("Listing clusters: ", err)
				return ctrl.Result{}, nil
			}

			managedClustersUpgrade.Status.Clusters = clustersStatus
			apimeta.SetStatusCondition(&managedClustersUpgrade.Status.Conditions, GetSelectedCondition(len(clustersStatus)))
		}
		// Update changes
		err = r.Status().Update(ctx, managedClustersUpgrade)
		if err != nil {
			klog.Error("Update managedClustersUpgrade.Status:", err)
			return ctrl.Result{}, nil
		}
	}

	return ctrl.Result{}, nil
}

func (r *ManagedClustersUpgradeReconciler) initClustersList(kubeClient client.Client, placement clusterv1beta1.GenericPlacementFields) ([]clusterv1beta1.ClusterStatusSpec, error) {
	Clusters, err := GetManagedClusterInfos(kubeClient, placement)
	if err != nil {
		return nil, err
	}

	var clustersStatus []clusterv1beta1.ClusterStatusSpec

	for k, cluster := range Clusters {
		//klog.Info("clusterInfo version:", cluster.Status.DistributionInfo.OCP.Version)
		//klog.Info("clusterInfo versionHistory:", cluster.Status.DistributionInfo.OCP.VersionHistory)
		labels := cluster.GetLabels()
		clusterId := ""
		if len(labels) > 0 {
			clusterId = labels["clusterID"]
		}
		clusterStatus := clusterv1beta1.ClusterStatusSpec{
			Name:      k,
			ClusterID: clusterId,
			ClusterVersionStatus: clusterv1beta1.ClusterVersionStatus{
				State: clusterv1beta1.NotStartedState,
			},
		}

		clustersStatus = append(clustersStatus, clusterStatus)
	}

	return clustersStatus, err
}

// SetupWithManager sets up the controller with the Manager.
func (r *ManagedClustersUpgradeReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&clusterv1beta1.ManagedClustersUpgrade{}).
		Complete(r)
}
