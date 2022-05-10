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
	"time"
	//"fmt"

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
	//"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
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
//+kubebuilder:rbac:groups=work.open-cluster-management.io,resources=manifestwork,verbs=get;list;watch;create;update;patch;delete

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
		klog.Info(managedClustersUpgrade.Name)

		// Condition TypeSelected
		if condition := apimeta.FindStatusCondition(managedClustersUpgrade.Status.Conditions, TypeSelected); condition != nil {
			//klog.Info("type selected")
			if condition.Reason == ReasonNotSelected && condition.Status == metav1.ConditionFalse {
				//klog.Info("reason not selected")
				clustersStatus, err := r.initClustersList(r.Client, managedClustersUpgrade.Spec.GenericPlacementFields)
				if err != nil {
					klog.Error("Listing clusters: ", err)
					return ctrl.Result{}, nil
				}

				managedClustersUpgrade.Status.Clusters = clustersStatus
				apimeta.SetStatusCondition(&managedClustersUpgrade.Status.Conditions, GetSelectedCondition(len(clustersStatus)))
			} else if condition.Reason == ReasonSelected && condition.Status == metav1.ConditionTrue {
				//Check for label selector changes after select group of clusters

			}
		} else {
			// Init Condition TypeSelected: init Clusters list
			clustersStatus, err := r.initClustersList(r.Client, managedClustersUpgrade.Spec.GenericPlacementFields)
			if err != nil {
				klog.Error("Listing clusters: ", err)
				return ctrl.Result{}, nil
			}

			managedClustersUpgrade.Status.Clusters = clustersStatus
			apimeta.SetStatusCondition(&managedClustersUpgrade.Status.Conditions, GetSelectedCondition(len(clustersStatus)))
		}

		// Condition TypeApplied
		if condition := apimeta.FindStatusCondition(managedClustersUpgrade.Status.Conditions, TypeApplied); condition != nil {
			klog.Info("type applied")
			if condition.Status == metav1.ConditionFalse {
				manifestCreated := false
				for id, cluster := range managedClustersUpgrade.Status.Clusters {
					// Create ocp upgrade first
					manifestWork, err := CreateClusterVersionUpgradeManifestWork(cluster.Name, cluster.ClusterID, managedClustersUpgrade.Spec.ClusterVersion)
					if err != nil {
						klog.Info("ClusterVersion ManifestWork not initaliz: "+cluster.Name+" ", err)
					} else {
						klog.Info("Create ManifestWork ", manifestWork.Name)
						err = r.Client.Create(ctx, manifestWork)
						if err != nil {
							klog.Error("Error create ManifestWork: ", manifestWork.Name, " ", err)
						}
						managedClustersUpgrade.Status.Clusters[id].ClusterUpgradeStatus.State = clusterv1beta1.InitializedState
						manifestCreated = true
						// ignore operator upgrade till ocp upgrade done
						continue
					}

					// check Operators upgrade if only required
					manifestWork, err = CreateOperatorUpgradeManifestWork(cluster.Name, managedClustersUpgrade.Spec.OcpOperators)
					if err != nil {
						klog.Info("OCP Operators ManifestWork not initaliz: "+cluster.Name+" ", err)
					} else {
						klog.Info("Create ManifestWork ", manifestWork.Name)
						err = r.Client.Create(ctx, manifestWork)
						if err != nil {
							klog.Error("Error create ManifestWork: ", manifestWork.Name, " ", err)
						} else {
							managedClustersUpgrade.Status.Clusters[id].OperatorsStatus.UpgradeApproveState = clusterv1beta1.InitializedState
							manifestCreated = true
						}
					}
				}

				if manifestCreated {
					condition.Status = metav1.ConditionTrue
				}
			} else if condition.Status == metav1.ConditionTrue && condition.Reason == ReasonApplied {
				// Check for cluster upgrade if done then apply operators upgrade if exist
				if managedClustersUpgrade.Spec.OcpOperators.ApproveAllUpgrades || len(managedClustersUpgrade.Spec.OcpOperators.Include) > 0 {
					for _, cluster := range managedClustersUpgrade.Status.Clusters {
						// check for ocp upgrade complete and operators upgrade not start
						if cluster.ClusterUpgradeStatus.State == clusterv1beta1.CompleteState &&
							cluster.OperatorsStatus.UpgradeApproveState == clusterv1beta1.NotStartedState {

						}
					}
				}
			}
		} else {
			// Init Condition TypeApplied if clusters are selected
			if apimeta.IsStatusConditionTrue(managedClustersUpgrade.Status.Conditions, TypeSelected) {
				apimeta.SetStatusCondition(&managedClustersUpgrade.Status.Conditions, GetAppliedCondition())
			}
		}

		// Condition TypeInProgress
		if condition := apimeta.FindStatusCondition(managedClustersUpgrade.Status.Conditions, TypeInProgress); condition != nil {
			if condition.Status == metav1.ConditionFalse && condition.Reason != ReasonUpgradeComplete {
				klog.Info("type inprogress ConditionFalse")
				// Check if the ManifestWork get Applied without errors
				inProgress := false
				for id, cluster := range managedClustersUpgrade.Status.Clusters {
					if cluster.ClusterUpgradeStatus.State == clusterv1beta1.InitializedState {
						resAvailable, err := isManifestWorkResourcesAvailable(r.Client, cluster.Name+ClusterUpgradeManifestName, cluster.Name)
						if err != nil {
							klog.Error("Get Manifestwork error ", cluster.Name+ClusterUpgradeManifestName+" ", err)
						}

						if resAvailable {
							managedClustersUpgrade.Status.Clusters[id].ClusterUpgradeStatus.State = clusterv1beta1.PartialState
							inProgress = true
						}

					}
					if cluster.OperatorsStatus.UpgradeApproveState == clusterv1beta1.InitializedState {
						resAvailable, err := isManifestWorkResourcesAvailable(r.Client, cluster.Name+OperatorUpgradeManifestName, cluster.Name)
						if err != nil {
							klog.Error("Get Manifest error ", cluster.Name+OperatorUpgradeManifestName+" ", err)
						}

						if resAvailable {
							managedClustersUpgrade.Status.Clusters[id].OperatorsStatus.UpgradeApproveState = clusterv1beta1.PartialState
							inProgress = true
						}
					}
				}
				if inProgress {
					condition.Status = metav1.ConditionTrue
				}
			} else if condition.Status == metav1.ConditionTrue {
				klog.Info("type inprogress ConditionTrue")
				countComplete := 0
				for id, cluster := range managedClustersUpgrade.Status.Clusters {
					// Check the cluster upgrade
					if cluster.ClusterUpgradeStatus.State == clusterv1beta1.InitializedState {
						resAvailable, err := isManifestWorkResourcesAvailable(r.Client, cluster.Name+ClusterUpgradeManifestName, cluster.Name)
						if err != nil {
							klog.Error("Get Manifestwork error ", cluster.Name+ClusterUpgradeManifestName+" ", err)
						}

						if resAvailable {
							managedClustersUpgrade.Status.Clusters[id].ClusterUpgradeStatus.State = clusterv1beta1.PartialState
						}
					} else if cluster.ClusterUpgradeStatus.State == clusterv1beta1.PartialState {
						// check for managedCluster state to set
						klog.Info("I'm in ocp upgrade partial state")
					} else if cluster.ClusterUpgradeStatus.State == clusterv1beta1.CompleteState {
						// check if there is operator upgrade defined otherwise increase count
						if !managedClustersUpgrade.Spec.OcpOperators.ApproveAllUpgrades &&
							len(managedClustersUpgrade.Spec.OcpOperators.Include) == 0 {
							countComplete++
						}
					}

					// Check the operator upgrade
					if cluster.OperatorsStatus.UpgradeApproveState == clusterv1beta1.InitializedState {
						resAvailable, err := isManifestWorkResourcesAvailable(r.Client, cluster.Name+OperatorUpgradeManifestName, cluster.Name)
						if err != nil {
							klog.Error("Get Manifest error ", cluster.Name+OperatorUpgradeManifestName+" ", err)
						}

						if resAvailable {
							managedClustersUpgrade.Status.Clusters[id].OperatorsStatus.UpgradeApproveState = clusterv1beta1.PartialState
						}
					} else if cluster.OperatorsStatus.UpgradeApproveState == clusterv1beta1.PartialState {
						// check for the install plan approve job
						klog.Info("I'm in operator upgrade partial state")
						status, value, err := getOperatorUpgradeManifestStatus(r.Client, cluster.Name+OperatorUpgradeManifestName, cluster.Name)
						if err != nil {
							klog.Error("Get OperatorUpgradeManifestStatus ", err)
						}
						klog.Info("OperatorUpgradeManifestStatus ", status+" ", value)
						if status == OperatorUpgradeSucceededState && value == OperatorUpgradeValidStateValue {
							managedClustersUpgrade.Status.Clusters[id].OperatorsStatus.UpgradeApproveState = clusterv1beta1.CompleteState
						} else if status == OperatorUpgradeFailedState && value == OperatorUpgradeValidStateValue {
							managedClustersUpgrade.Status.Clusters[id].OperatorsStatus.UpgradeApproveState = clusterv1beta1.FailedState
						} else if status == OperatorUpgradeActiveState && value == OperatorUpgradeValidStateValue {
							managedClustersUpgrade.Status.Clusters[id].OperatorsStatus.UpgradeApproveState = clusterv1beta1.PartialState
						}
					} else if cluster.OperatorsStatus.UpgradeApproveState == clusterv1beta1.CompleteState {
						countComplete++
					}
				}
				if countComplete == len(managedClustersUpgrade.Status.Clusters) {
					condition.Status = metav1.ConditionFalse
					condition.Reason = ReasonUpgradeComplete
				}
			}
		} else {
			// Init Condition TypeInProgress if upgrades are applied
			if apimeta.IsStatusConditionTrue(managedClustersUpgrade.Status.Conditions, TypeApplied) {
				apimeta.SetStatusCondition(&managedClustersUpgrade.Status.Conditions, GetInProgressCondition())
			}
		}

		// Condition TypeComplete
		if condition := apimeta.FindStatusCondition(managedClustersUpgrade.Status.Conditions, TypeComplete); condition != nil {

			if condition.Status == metav1.ConditionFalse {
				klog.Info("type complete false")
				allClusterUpgradeComplete, allOperatorUpgradeComplete := true, true
				for _, cluster := range managedClustersUpgrade.Status.Clusters {
					if cluster.ClusterUpgradeStatus.State == clusterv1beta1.CompleteState {
						// Delete cluster upgrade manifest work
					} else if cluster.ClusterUpgradeStatus.State == clusterv1beta1.NotStartedState {
						// check if there is cluster version upgrade defined
					} else {
						allClusterUpgradeComplete = false
					}

					if cluster.OperatorsStatus.UpgradeApproveState == clusterv1beta1.CompleteState {
						// Delete cluster operator upgrade manifest work

					} else if cluster.OperatorsStatus.UpgradeApproveState == clusterv1beta1.NotStartedState {
						// check if there is cluster operator upgrade defined
					} else {
						allOperatorUpgradeComplete = false
					}
				}
				if allClusterUpgradeComplete && allOperatorUpgradeComplete {
					condition.Status = metav1.ConditionTrue
				}
			} else if condition.Status == metav1.ConditionTrue {
				klog.Info("type complete true")
				// Update changes
				err = r.Status().Update(ctx, managedClustersUpgrade)
				if err != nil {
					klog.Error("Update managedClustersUpgrade.Status:", err)
					return ctrl.Result{}, nil
				}
				return ctrl.Result{}, nil
			}

		} else {
			// InProgress condition exist and its status is true
			if apimeta.IsStatusConditionTrue(managedClustersUpgrade.Status.Conditions, TypeInProgress) {
				apimeta.SetStatusCondition(&managedClustersUpgrade.Status.Conditions, GetCompleteCondition())
			}
		}

		// Update changes
		err = r.Status().Update(ctx, managedClustersUpgrade)
		if err != nil {
			klog.Error("Update managedClustersUpgrade.Status:", err)
			return ctrl.Result{}, nil
		}
	}

	return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
}

func (r *ManagedClustersUpgradeReconciler) initClustersList(kubeClient client.Client, placement clusterv1beta1.GenericPlacementFields) ([]clusterv1beta1.ClusterStatusSpec, error) {
	Clusters, err := GetManagedClusterList(kubeClient, placement)
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
			ClusterUpgradeStatus: clusterv1beta1.ClusterUpgradeStatus{
				State: clusterv1beta1.NotStartedState,
			},
			OperatorsStatus: clusterv1beta1.OperatorsStatus{
				UpgradeApproveState: clusterv1beta1.NotStartedState,
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
