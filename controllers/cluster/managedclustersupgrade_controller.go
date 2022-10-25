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
	utils "github.com/redhat-ztp/managedclusters-lifecycle-operator/controllers/utils"
	//clusterv1 "open-cluster-management.io/api/cluster/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/sets"
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
	// Set default reconcile for 5min as upgrade takes from 40min to 3h
	result := ctrl.Result{RequeueAfter: 30 * time.Second}
	err := r.Get(ctx, req.NamespacedName, managedClustersUpgrade)

	if err != nil {
		if apierrors.IsNotFound(err) {
			klog.Info("ManagedClustersUpgrade resource not found. Ignoring since object must be deleted")
			return result, nil
		}
		klog.Error(err, "Failed to get ManagedClustersUpgrade")
		return result, err
	} else {
		// Condition TypeSelected
		clusterCount, err := r.processSelectedCondition(managedClustersUpgrade)
		apimeta.SetStatusCondition(&managedClustersUpgrade.Status.Conditions, utils.GetSelectedCondition(clusterCount))
		if err != nil {
			klog.Error("Listing clusters: ", err)
			return result, nil
		}

		// Condition TypeApplied
		if r.processAppliedCondition(ctx, managedClustersUpgrade) {
			apimeta.SetStatusCondition(&managedClustersUpgrade.Status.Conditions, utils.GetAppliedCondition(metav1.ConditionTrue))
		} else {
			apimeta.SetStatusCondition(&managedClustersUpgrade.Status.Conditions, utils.GetAppliedCondition(metav1.ConditionFalse))
		}

		// Condition TypeInProgress
		if condition := apimeta.FindStatusCondition(managedClustersUpgrade.Status.Conditions, utils.TypeInProgress); condition != nil {
			if condition.Status == metav1.ConditionFalse && condition.Reason != utils.ReasonUpgradeComplete {
				//klog.Info("type inprogress ConditionFalse")
				inProgress, _ := r.processInProgressCondition(managedClustersUpgrade)
				if inProgress {
					condition.Status = metav1.ConditionTrue
				}
			} else if condition.Status == metav1.ConditionTrue {
				//klog.Info("type inprogress ConditionTrue")
				inProgress, complete := r.processInProgressCondition(managedClustersUpgrade)

				if !inProgress {
					// Set condition to false when all clusters state is complete, failed or Initialized
					//klog.Info("set inprogress false")
					condition.Status = metav1.ConditionFalse
				}

				if complete {
					klog.Info("inprogress upgrade finish set ReasonUpgradeComplete")
					//condition.Status = metav1.ConditionFalse
					condition.Reason = utils.ReasonUpgradeComplete
				}
			}
		} else {
			// Create Condition InProgress if upgrades are applied
			if apimeta.IsStatusConditionTrue(managedClustersUpgrade.Status.Conditions, utils.TypeApplied) {
				apimeta.SetStatusCondition(&managedClustersUpgrade.Status.Conditions, utils.GetInProgressCondition())
			}
		}

		// Condition TypeComplete
		if condition := apimeta.FindStatusCondition(managedClustersUpgrade.Status.Conditions, utils.TypeComplete); condition != nil {
			if condition.Status == metav1.ConditionFalse {
				//klog.Info("type complete false")
				if r.processCompleteCondition(managedClustersUpgrade, false) {
					condition.Status = metav1.ConditionTrue
				}
			} else if condition.Status == metav1.ConditionTrue {
				klog.Info("type complete is done no more reconsle for this CR " + managedClustersUpgrade.Name)
				result = ctrl.Result{}
			}
		} else {
			// InProgress condition exist and its status is true
			if apimeta.IsStatusConditionTrue(managedClustersUpgrade.Status.Conditions, utils.TypeInProgress) {
				apimeta.SetStatusCondition(&managedClustersUpgrade.Status.Conditions, utils.GetCompleteCondition())
			}
		}

		// Condition TypeCanaryComplete
		if condition := apimeta.FindStatusCondition(managedClustersUpgrade.Status.Conditions, utils.TypeCanaryComplete); condition != nil {
			if condition.Status == metav1.ConditionFalse {
				//klog.Info("type complete false")
				if r.processCompleteCondition(managedClustersUpgrade, true) {
					condition.Status = metav1.ConditionTrue
				}
			}
		} else {
			// Check for Canary clusters exist
			if len(managedClustersUpgrade.Status.CanaryClusters) > 0 {
				// Check for InProgress condition exist with true status
				if apimeta.IsStatusConditionTrue(managedClustersUpgrade.Status.Conditions, utils.TypeInProgress) {
					apimeta.SetStatusCondition(&managedClustersUpgrade.Status.Conditions, utils.GetCanaryCompleteCondition())
				}
			}
		}

		// Condition TypeFailed
		if condition := apimeta.FindStatusCondition(managedClustersUpgrade.Status.Conditions, utils.TypeFailed); condition != nil {
			allFailed, failedCount := r.processFailedCondition(managedClustersUpgrade, false)
			condition.Message = utils.GetFailedConditionMessage(failedCount)

			if allFailed {
				klog.Info("All clusters failed no more reconsle for this CR " + managedClustersUpgrade.Name)
				result = ctrl.Result{}
			}

		} else {
			// Complete condition exist and its status is false
			if apimeta.IsStatusConditionFalse(managedClustersUpgrade.Status.Conditions, utils.TypeComplete) {
				_, failedCount := r.processFailedCondition(managedClustersUpgrade, false)
				// Add failed condition if at least 1 cluster failed
				if failedCount > 0 {
					apimeta.SetStatusCondition(&managedClustersUpgrade.Status.Conditions, utils.GetFailedCondition(failedCount))
				}
			}
		}

		// Condition TypeCanaryFailed
		if condition := apimeta.FindStatusCondition(managedClustersUpgrade.Status.Conditions, utils.TypeCanaryFailed); condition != nil {
			allFailed, failedCount := r.processFailedCondition(managedClustersUpgrade, true)
			condition.Message = utils.GetFailedConditionMessage(failedCount)

			if allFailed {
				klog.Info("All canary clusters failed no more reconsle for this CR " + managedClustersUpgrade.Name)
				result = ctrl.Result{}
			}

		} else {
			// check Canary clusters exist
			if len(managedClustersUpgrade.Status.CanaryClusters) > 0 {
				// Complete condition exist and its status is false
				if apimeta.IsStatusConditionFalse(managedClustersUpgrade.Status.Conditions, utils.TypeCanaryComplete) {
					_, failedCount := r.processFailedCondition(managedClustersUpgrade, true)
					// Add failed condition if at least 1 cluster failed
					if failedCount > 0 {
						apimeta.SetStatusCondition(&managedClustersUpgrade.Status.Conditions, utils.GetCanaryFailedCondition(failedCount))
					}
				}
			}
		}

		// Update changes
		//klog.Info("update status")
		err = r.Status().Update(ctx, managedClustersUpgrade)
		if err != nil {
			klog.Error("Update managedClustersUpgrade.Status:", err)
			return result, nil
		}
	}

	return result, nil
}

func (r *ManagedClustersUpgradeReconciler) processSelectedCondition(managedClustersUpgrade *clusterv1beta1.ManagedClustersUpgrade) (int, error) {
	canaryClustersStatus, err := r.getClustersList(r.Client, managedClustersUpgrade.Spec.UpgradeStrategy.CanaryClusters,
		managedClustersUpgrade.Status.CanaryClusters)
	if err != nil {
		return 0, err
	}

	clustersStatus, err := r.getClustersList(r.Client, managedClustersUpgrade.Spec.GenericPlacementFields,
		managedClustersUpgrade.Status.Clusters)
	if err != nil {
		return 0, err
	}

	managedClustersUpgrade.Status.CanaryClusters = canaryClustersStatus
	managedClustersUpgrade.Status.Clusters = clustersStatus

	return len(clustersStatus) + len(canaryClustersStatus), nil
}

func (r *ManagedClustersUpgradeReconciler) processAppliedCondition(ctx context.Context, managedClustersUpgrade *clusterv1beta1.ManagedClustersUpgrade) bool {
	manifestCreated := true
	batchCount := 0
	var clusters []*clusterv1beta1.ClusterStatus

	// check if canary cluster upgrade exist or complete
	if len(managedClustersUpgrade.Status.CanaryClusters) == 0 || apimeta.IsStatusConditionTrue(managedClustersUpgrade.Status.Conditions, utils.TypeCanaryComplete) {
		clusters = managedClustersUpgrade.Status.Clusters
	} else if len(managedClustersUpgrade.Status.CanaryClusters) > 0 {
		clusters = managedClustersUpgrade.Status.CanaryClusters
	}

	if len(clusters) == 0 {
		return false
	}

	for _, cluster := range clusters {
		// limit num of clusters in upgrade to MaxConcurrency
		//klog.Info("batch count is ", batchCount, "max concurrency is ", managedClustersUpgrade.Spec.UpgradeStrategy.MaxConcurrency)
		if batchCount >= managedClustersUpgrade.Spec.UpgradeStrategy.MaxConcurrency {
			break
		}

		//TODO: Better to ignore only the completed clusters if all clusters in a single batch failed we don't want to continue.
		// Ignore failed/complete clusters
		if cluster.OperatorsStatus.UpgradeApproveState == clusterv1beta1.CompleteState ||
			cluster.OperatorsStatus.UpgradeApproveState == clusterv1beta1.FailedState {
			continue
		}

		if cluster.ClusterUpgradeStatus.State == clusterv1beta1.NotStartedState {
			// Create cluster upgrade first
			manifestWork, err := utils.CreateClusterVersionUpgradeManifestWork(cluster.Name, cluster.ClusterID, managedClustersUpgrade.Spec.ClusterVersion)
			if err != nil {
				klog.Info("ClusterVersion ManifestWork not initaliz: ", cluster.Name, " ", err)
			} else {
				//klog.Info("Create ManifestWork ", manifestWork.Name)
				err = r.Client.Create(ctx, manifestWork)
				if err != nil {
					klog.Error("Failed create ManifestWork: ", manifestWork.Name, " ", err)
					manifestCreated = false
				} else {
					cluster.ClusterUpgradeStatus.State = clusterv1beta1.InitializedState
				}
				// ignore operator upgrade till ocp upgrade done & increase batch count
				batchCount++
				continue
			}
		}

		if cluster.OperatorsStatus.UpgradeApproveState == clusterv1beta1.NotStartedState {
			// Only Apply operator upgrades if the cluster upgrade is complete or not started
			if cluster.ClusterUpgradeStatus.State == clusterv1beta1.CompleteState ||
				cluster.ClusterUpgradeStatus.State == clusterv1beta1.NotStartedState {
				manifestWork, err := utils.CreateOperatorUpgradeManifestWork(cluster.Name, managedClustersUpgrade.Spec.OcpOperators,
					managedClustersUpgrade.Spec.UpgradeStrategy.OperatorsUpgradeTimeout, utils.OseCliDefaultImage)
				if err != nil {
					klog.Info("OCP Operators ManifestWork not initaliz: ", cluster.Name, " ", err)
					manifestCreated = false
					// operators upgrade not valid
					continue
				} else {
					//klog.Info("Create ManifestWork ", manifestWork.Name)
					err = r.Client.Create(ctx, manifestWork)
					if err != nil {
						klog.Error("Failed create ManifestWork: ", manifestWork.Name, " ", err)
					} else {
						cluster.OperatorsStatus.UpgradeApproveState = clusterv1beta1.InitializedState
					}
				}
			}
		}
		// increase batch count
		batchCount++
	}

	return manifestCreated
}

func (r *ManagedClustersUpgradeReconciler) processInProgressCondition(managedClustersUpgrade *clusterv1beta1.ManagedClustersUpgrade) (bool, bool) {
	countComplete := 0
	inProgress, isCanary := false, false
	var clusters []*clusterv1beta1.ClusterStatus

	// check if canary cluster upgrade exist or complete
	if len(managedClustersUpgrade.Status.CanaryClusters) == 0 || apimeta.IsStatusConditionTrue(managedClustersUpgrade.Status.Conditions, utils.TypeCanaryComplete) {
		clusters = managedClustersUpgrade.Status.Clusters
	} else if len(managedClustersUpgrade.Status.CanaryClusters) > 0 {
		clusters = managedClustersUpgrade.Status.CanaryClusters
		isCanary = true
	}

	for _, cluster := range clusters {
		if cluster.ClusterUpgradeStatus.State == clusterv1beta1.InitializedState {
			resAvailable, isTimeOut, err := utils.IsManifestWorkResourcesAvailable(r.Client, cluster.Name+utils.ClusterUpgradeManifestName, cluster.Name, managedClustersUpgrade.Spec.UpgradeStrategy.ClusterUpgradeTimeout)
			if err != nil {
				klog.Error("Failed get Manifestwork ", cluster.Name+utils.ClusterUpgradeManifestName+" ", err)
			}

			if resAvailable {
				inProgress = true

				version, state, verified, message, _, err := utils.GetClusterUpgradeManifestStatus(r.Client, cluster.Name+utils.ClusterUpgradeManifestName, cluster.Name, managedClustersUpgrade.Spec.UpgradeStrategy.ClusterUpgradeTimeout)
				if err != nil {
					klog.Error("ClusterUpgradeManifestStatus ", err)
					continue
				}
				// validate same version as expected from status otherwise still in InitializedState
				if version == managedClustersUpgrade.Spec.ClusterVersion.Version {
					cluster.ClusterUpgradeStatus.State = state
					cluster.ClusterUpgradeStatus.Verified = verified
				}
				cluster.ClusterUpgradeStatus.Message = message
			}

			// Check for timeout & InitializedState in case the timeout is less than the Reconcile time.
			if isTimeOut && cluster.ClusterUpgradeStatus.State == clusterv1beta1.InitializedState {
				klog.Info("InitializedState cluster upgrade timeout ", cluster.Name)
				cluster.ClusterUpgradeStatus.State = clusterv1beta1.FailedState
			}
		} else if cluster.ClusterUpgradeStatus.State == clusterv1beta1.PartialState {
			//TODO: check for failed condition status clusterVersion state does not return failed state
			// check for managedCluster state to set
			inProgress = true
			version, state, verified, message, isTimeOut, err := utils.GetClusterUpgradeManifestStatus(r.Client, cluster.Name+utils.ClusterUpgradeManifestName, cluster.Name, managedClustersUpgrade.Spec.UpgradeStrategy.ClusterUpgradeTimeout)
			if err != nil {
				klog.Error("ClusterUpgradeManifestStatus ", err)
				continue
			}

			//klog.Info("ClusterUpgradeManifestStatus ", version+" ", state+" ", verified, " ", message)
			// validate same version as expected from status
			if version == managedClustersUpgrade.Spec.ClusterVersion.Version {
				cluster.ClusterUpgradeStatus.State = state
				cluster.ClusterUpgradeStatus.Verified = verified
			}
			cluster.ClusterUpgradeStatus.Message = message

			// Add check if the complete state reached before the Reconcile or timeout
			if isTimeOut && cluster.ClusterUpgradeStatus.State != clusterv1beta1.CompleteState {
				klog.Info("PartialState cluster upgrade timeout ", cluster.Name)
				cluster.ClusterUpgradeStatus.State = clusterv1beta1.FailedState
			}
		} else if cluster.ClusterUpgradeStatus.State == clusterv1beta1.CompleteState {
			// check if there is operator upgrade defined otherwise increase count
			if managedClustersUpgrade.Spec.OcpOperators == nil ||
				(!managedClustersUpgrade.Spec.OcpOperators.ApproveAllUpgrades &&
					len(managedClustersUpgrade.Spec.OcpOperators.Include) == 0) {
				countComplete++
			}
		}

		// Check the operator upgrade
		if cluster.OperatorsStatus.UpgradeApproveState == clusterv1beta1.InitializedState {
			resAvailable, isTimeOut, err := utils.IsManifestWorkResourcesAvailable(r.Client,
				cluster.Name+utils.OperatorUpgradeManifestName, cluster.Name,
				managedClustersUpgrade.Spec.UpgradeStrategy.OperatorsUpgradeTimeout)
			if err != nil {
				klog.Error("Get Manifest error ", cluster.Name+utils.OperatorUpgradeManifestName+" ", err)
			}

			if resAvailable {
				cluster.OperatorsStatus.UpgradeApproveState = clusterv1beta1.PartialState
				inProgress = true
			}

			// Check for timeout & InitializedState in case the timeout is less than the Reconcile time.
			if isTimeOut && cluster.OperatorsStatus.UpgradeApproveState == clusterv1beta1.InitializedState {
				klog.Info("cluster operators upgrade timeout ", cluster.Name)
				cluster.OperatorsStatus.UpgradeApproveState = clusterv1beta1.FailedState
			}
		} else if cluster.OperatorsStatus.UpgradeApproveState == clusterv1beta1.PartialState {
			// check for the install plan approve job
			inProgress = true
			status, value, _, err := utils.GetOperatorUpgradeManifestStatus(r.Client,
				cluster.Name+utils.OperatorUpgradeManifestName, cluster.Name,
				managedClustersUpgrade.Spec.UpgradeStrategy.OperatorsUpgradeTimeout)
			if err != nil {
				klog.Error("failed get OperatorUpgradeManifestStatus ", err)
			}

			//klog.Info("OperatorUpgradeManifestStatus ", status+" ", value)
			if status == utils.OperatorUpgradeSucceededState && value == utils.OperatorUpgradeValidStateValue {
				cluster.OperatorsStatus.UpgradeApproveState = clusterv1beta1.CompleteState
			} else if status == utils.OperatorUpgradeFailedState && value == utils.OperatorUpgradeValidStateValue {
				cluster.OperatorsStatus.UpgradeApproveState = clusterv1beta1.FailedState
			} else if status == utils.OperatorUpgradeActiveState && value == utils.OperatorUpgradeValidStateValue {
				cluster.OperatorsStatus.UpgradeApproveState = clusterv1beta1.PartialState
			}

		} else if cluster.OperatorsStatus.UpgradeApproveState == clusterv1beta1.CompleteState {
			countComplete++
		}
	}

	// return complete when the count of complete state clusters equal to all clusters and not equal to 0 and not canary clusters
	// klog.Info("Process inProgress complete is ", (countComplete == len(clusters) && countComplete > 0 && !isCanary))
	return inProgress, (countComplete == len(clusters) && countComplete > 0 && !isCanary)
}

func (r *ManagedClustersUpgradeReconciler) processCompleteCondition(managedClustersUpgrade *clusterv1beta1.ManagedClustersUpgrade, canaryClusters bool) bool {
	completeAll := true
	var clusters []*clusterv1beta1.ClusterStatus

	if canaryClusters {
		clusters = managedClustersUpgrade.Status.CanaryClusters
	} else {
		clusters = managedClustersUpgrade.Status.Clusters
	}

	for _, cluster := range clusters {
		if cluster.ClusterUpgradeStatus.State == clusterv1beta1.CompleteState {
			// Delete cluster operator upgrade manifest work
			manifestName := cluster.Name + utils.ClusterUpgradeManifestName
			//klog.Info("Upgrade complete Delete manfiestwork ", manifestName)
			err := utils.DeleteManifestWork(r.Client, manifestName, cluster.Name)
			if !apierrors.IsNotFound(err) {
				klog.Error("Delete manifestwork ", manifestName+" ", err)
			}
		} else if cluster.ClusterUpgradeStatus.State == clusterv1beta1.NotStartedState {
			if managedClustersUpgrade.Spec.ClusterVersion != nil {
				// check if there is cluster version upgrade defined
				completeAll = false
			}
		} else {
			completeAll = false
		}

		if cluster.OperatorsStatus.UpgradeApproveState == clusterv1beta1.CompleteState {
			// Delete cluster operator upgrade manifest work
			manifestName := cluster.Name + utils.OperatorUpgradeManifestName
			// klog.Info("Upgrade complete Delete manfiestwork ", manifestName)
			err := utils.DeleteManifestWork(r.Client, manifestName, cluster.Name)
			if !apierrors.IsNotFound(err) {
				klog.Error("Delete manifestwork ", manifestName+" ", err)
			}
		} else if cluster.OperatorsStatus.UpgradeApproveState == clusterv1beta1.NotStartedState {
			if managedClustersUpgrade.Spec.OcpOperators != nil && (managedClustersUpgrade.Spec.OcpOperators.ApproveAllUpgrades ||
				len(managedClustersUpgrade.Spec.OcpOperators.Include) > 0) {
				completeAll = false
			}
		} else {
			completeAll = false
		}
	}

	return completeAll
}

func (r *ManagedClustersUpgradeReconciler) processFailedCondition(managedClustersUpgrade *clusterv1beta1.ManagedClustersUpgrade, canaryClusters bool) (bool, int) {
	failedCount := 0
	var clusters []*clusterv1beta1.ClusterStatus

	if canaryClusters {
		clusters = managedClustersUpgrade.Status.CanaryClusters

	} else {
		clusters = managedClustersUpgrade.Status.Clusters
	}

	for _, cluster := range clusters {
		if cluster.ClusterUpgradeStatus.State == clusterv1beta1.FailedState {
			manifestName := cluster.Name + utils.ClusterUpgradeManifestName
			klog.Info("Upgrade failed Delete manfiestwork ", manifestName)
			err := utils.DeleteManifestWork(r.Client, manifestName, cluster.Name)
			if !apierrors.IsNotFound(err) {
				klog.Error("Delete manifestwork ", manifestName+" ", err)
			}
		}

		if cluster.OperatorsStatus.UpgradeApproveState == clusterv1beta1.FailedState {
			// Delete cluster operator upgrade manifest work
			manifestName := cluster.Name + utils.OperatorUpgradeManifestName
			klog.Info("Upgrade failed Delete manfiestwork ", manifestName)
			err := utils.DeleteManifestWork(r.Client, manifestName, cluster.Name)
			if !apierrors.IsNotFound(err) {
				klog.Error("Delete manifestwork ", manifestName+" ", err)
			}
		}

		if cluster.ClusterUpgradeStatus.State == clusterv1beta1.FailedState ||
			cluster.OperatorsStatus.UpgradeApproveState == clusterv1beta1.FailedState {
			failedCount++
		}

	}

	// return true when all clusters in failed state
	return (failedCount == len(clusters) && failedCount != 0), failedCount
}

func (r *ManagedClustersUpgradeReconciler) getClustersList(kubeClient client.Client, placement clusterv1beta1.GenericPlacementFields, existingClustersList []*clusterv1beta1.ClusterStatus) ([]*clusterv1beta1.ClusterStatus, error) {
	existingClusters := sets.NewString()
	for _, cluster := range existingClustersList {
		existingClusters.Insert(cluster.Name)
	}

	mClusters, addedClusters, deletedClusters, err := utils.GetManagedClusterList(kubeClient, placement, existingClusters)
	if err != nil {
		return nil, err
	}

	if addedClusters.Len() == 0 && deletedClusters.Len() == 0 {
		return existingClustersList, nil
	}

	for id, cluster := range existingClustersList {
		if deletedClusters.Has(cluster.Name) {
			// Ignore removing the if the upgrade is running
			if cluster.ClusterUpgradeStatus.State == clusterv1beta1.InitializedState || cluster.ClusterUpgradeStatus.State == clusterv1beta1.PartialState {
				continue
			}
			existingClustersList = append(existingClustersList[:id], existingClustersList[id+1:]...)
		}
	}

	for addedCluster, _ := range addedClusters {
		if mCluster, ok := mClusters[addedCluster]; ok {
			labels := mCluster.GetLabels()
			clusterId := ""
			if len(labels) > 0 {
				clusterId = labels["clusterID"]
			}
			clusterStatus := &clusterv1beta1.ClusterStatus{
				Name:      addedCluster,
				ClusterID: clusterId,
				ClusterUpgradeStatus: clusterv1beta1.ClusterUpgradeStatus{
					State: clusterv1beta1.NotStartedState,
				},
				OperatorsStatus: clusterv1beta1.OperatorsStatus{
					UpgradeApproveState: clusterv1beta1.NotStartedState,
				},
			}
			existingClustersList = append(existingClustersList, clusterStatus)
		}
	}

	return existingClustersList, err
}

// SetupWithManager sets up the controller with the Manager.
func (r *ManagedClustersUpgradeReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&clusterv1beta1.ManagedClustersUpgrade{}).
		Complete(r)
}
