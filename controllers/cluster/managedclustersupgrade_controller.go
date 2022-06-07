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
	apierrors "k8s.io/apimachinery/pkg/api/errors"
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
		klog.Info(managedClustersUpgrade.Name)

		// Condition TypeSelected
		if condition := apimeta.FindStatusCondition(managedClustersUpgrade.Status.Conditions, TypeSelected); condition != nil {
			//klog.Info("type selected")
			if condition.Reason == ReasonNotSelected && condition.Status == metav1.ConditionFalse {
				//klog.Info("reason not selected")
				clusterCount, err := r.processSelectedCondition(managedClustersUpgrade)
				apimeta.SetStatusCondition(&managedClustersUpgrade.Status.Conditions, GetSelectedCondition(clusterCount))
				if err != nil {
					klog.Error("Listing clusters: ", err)
					return result, nil
				}
			} else if condition.Reason == ReasonSelected && condition.Status == metav1.ConditionTrue {
				//Check for label selector changes after select group of clusters
			}
		} else {
			// Create Condition Selected: init Clusters list
			clusterCount, err := r.processSelectedCondition(managedClustersUpgrade)
			apimeta.SetStatusCondition(&managedClustersUpgrade.Status.Conditions, GetSelectedCondition(clusterCount))
			if err != nil {
				klog.Error("Listing clusters: ", err)
				return result, nil
			}
		}

		// Condition TypeApplied
		if condition := apimeta.FindStatusCondition(managedClustersUpgrade.Status.Conditions, TypeApplied); condition != nil {
			if condition.Status == metav1.ConditionFalse {
				klog.Info("type applied condition false")
				if r.processAppliedCondition(ctx, managedClustersUpgrade) {
					condition.Status = metav1.ConditionTrue
				}
			} else if condition.Status == metav1.ConditionTrue && condition.Reason == ReasonApplied {
				klog.Info("type applied condition true")
				r.processAppliedCondition(ctx, managedClustersUpgrade)
			}
		} else {
			// Create Condition Applied if clusters are selected
			if apimeta.IsStatusConditionTrue(managedClustersUpgrade.Status.Conditions, TypeSelected) {
				apimeta.SetStatusCondition(&managedClustersUpgrade.Status.Conditions, GetAppliedCondition())
			}
		}

		// Condition TypeInProgress
		if condition := apimeta.FindStatusCondition(managedClustersUpgrade.Status.Conditions, TypeInProgress); condition != nil {
			if condition.Status == metav1.ConditionFalse && condition.Reason != ReasonUpgradeComplete {
				klog.Info("type inprogress ConditionFalse")
				inProgress, _ := r.processInProgressCondition(managedClustersUpgrade)
				if inProgress {
					condition.Status = metav1.ConditionTrue
				}
			} else if condition.Status == metav1.ConditionTrue {
				klog.Info("type inprogress ConditionTrue")
				inProgress, complete := r.processInProgressCondition(managedClustersUpgrade)

				if !inProgress {
					// Set condition to false when all clusters state is complete, failed or Initialized
					klog.Info("set inprogress false")
					condition.Status = metav1.ConditionFalse
				}

				if complete {
					klog.Info("inprogress upgrade finish set ReasonUpgradeComplete")
					//condition.Status = metav1.ConditionFalse
					condition.Reason = ReasonUpgradeComplete
				}
			}
		} else {
			// Create Condition InProgress if upgrades are applied
			if apimeta.IsStatusConditionTrue(managedClustersUpgrade.Status.Conditions, TypeApplied) {
				apimeta.SetStatusCondition(&managedClustersUpgrade.Status.Conditions, GetInProgressCondition())
			}
		}

		// Condition TypeComplete
		if condition := apimeta.FindStatusCondition(managedClustersUpgrade.Status.Conditions, TypeComplete); condition != nil {

			if condition.Status == metav1.ConditionFalse {
				klog.Info("type complete false")
				if r.processCompleteCondition(managedClustersUpgrade, false) {
					condition.Status = metav1.ConditionTrue
				}
			} else if condition.Status == metav1.ConditionTrue {
				klog.Info("type complete is done no more reconsle for this CR " + managedClustersUpgrade.Name)
				result = ctrl.Result{}
			}
		} else {
			// InProgress condition exist and its status is true
			if apimeta.IsStatusConditionTrue(managedClustersUpgrade.Status.Conditions, TypeInProgress) {
				apimeta.SetStatusCondition(&managedClustersUpgrade.Status.Conditions, GetCompleteCondition())
			}
		}

		// Condition TypeCanaryComplete
		if condition := apimeta.FindStatusCondition(managedClustersUpgrade.Status.Conditions, TypeCanaryComplete); condition != nil {
			if condition.Status == metav1.ConditionFalse {
				klog.Info("type complete false")
				if r.processCompleteCondition(managedClustersUpgrade, true) {
					condition.Status = metav1.ConditionTrue
				}
			}
		} else {
			// Check for Canary clusters exist
			if len(managedClustersUpgrade.Status.CanaryClusters) > 0 {
				// Check for InProgress condition exist with true status
				if apimeta.IsStatusConditionTrue(managedClustersUpgrade.Status.Conditions, TypeInProgress) {
					apimeta.SetStatusCondition(&managedClustersUpgrade.Status.Conditions, GetCanaryCompleteCondition())
				}
			}
		}

		// Condition TypeFailed
		if condition := apimeta.FindStatusCondition(managedClustersUpgrade.Status.Conditions, TypeFailed); condition != nil {
			allFailed, failedCount, failedNames := r.processFailedCondition(managedClustersUpgrade, false)
			condition.Message = GetFailedConditionMessage(failedCount, failedNames)

			if allFailed {
				klog.Info("All clusters failed no more reconsle for this CR " + managedClustersUpgrade.Name)
				result = ctrl.Result{}
			}

		} else {
			// Complete condition exist and its status is false
			if apimeta.IsStatusConditionFalse(managedClustersUpgrade.Status.Conditions, TypeComplete) {
				_, failedCount, failedNames := r.processFailedCondition(managedClustersUpgrade, false)
				// Add failed condition if at least 1 cluster failed
				if failedCount > 0 {
					apimeta.SetStatusCondition(&managedClustersUpgrade.Status.Conditions, GetFailedCondition(failedCount, failedNames))
				}
			}
		}

		// Condition TypeCanaryFailed
		if condition := apimeta.FindStatusCondition(managedClustersUpgrade.Status.Conditions, TypeCanaryFailed); condition != nil {
			allFailed, failedCount, failedNames := r.processFailedCondition(managedClustersUpgrade, true)
			condition.Message = GetFailedConditionMessage(failedCount, failedNames)

			if allFailed {
				klog.Info("All canary clusters failed no more reconsle for this CR " + managedClustersUpgrade.Name)
				result = ctrl.Result{}
			}

		} else {
			// check Canary clusters exist
			if len(managedClustersUpgrade.Status.CanaryClusters) > 0 {
				// Complete condition exist and its status is false
				if apimeta.IsStatusConditionFalse(managedClustersUpgrade.Status.Conditions, TypeCanaryComplete) {
					_, failedCount, failedNames := r.processFailedCondition(managedClustersUpgrade, true)
					// Add failed condition if at least 1 cluster failed
					if failedCount > 0 {
						apimeta.SetStatusCondition(&managedClustersUpgrade.Status.Conditions, GetCanaryFailedCondition(failedCount, failedNames))
					}
				}
			}
		}

		// Update changes
		klog.Info("update status")
		err = r.Status().Update(ctx, managedClustersUpgrade)
		if err != nil {
			klog.Error("Update managedClustersUpgrade.Status:", err)
			return result, nil
		}
	}

	return result, nil
}

func (r *ManagedClustersUpgradeReconciler) processSelectedCondition(managedClustersUpgrade *clusterv1beta1.ManagedClustersUpgrade) (int, error) {

	canaryClustersStatus, err := r.initClustersList(r.Client, managedClustersUpgrade.Spec.UpgradeStrategy.CanaryClusters)
	if err != nil {
		return 0, err
	}

	clustersStatus, err := r.initClustersList(r.Client, managedClustersUpgrade.Spec.GenericPlacementFields)
	if err != nil {
		return 0, err
	}

	managedClustersUpgrade.Status.CanaryClusters = canaryClustersStatus
	managedClustersUpgrade.Status.Clusters = clustersStatus

	return len(clustersStatus) + len(canaryClustersStatus), nil
}

func (r *ManagedClustersUpgradeReconciler) processAppliedCondition(ctx context.Context, managedClustersUpgrade *clusterv1beta1.ManagedClustersUpgrade) bool {
	manifestCreated := false
	batchCount := 0
	var clusters []*clusterv1beta1.ClusterStatusSpec

	// check if canary cluster upgrade exist or complete
	if len(managedClustersUpgrade.Status.CanaryClusters) == 0 || apimeta.IsStatusConditionTrue(managedClustersUpgrade.Status.Conditions, TypeCanaryComplete) {
		clusters = managedClustersUpgrade.Status.Clusters
	} else if len(managedClustersUpgrade.Status.CanaryClusters) > 0 {
		clusters = managedClustersUpgrade.Status.CanaryClusters
	}

	for _, cluster := range clusters {
		// limit num of clusters in upgrade to MaxConcurrency
		klog.Info("batch count is ", batchCount, "max concurrency is ", managedClustersUpgrade.Spec.UpgradeStrategy.MaxConcurrency)
		if batchCount >= managedClustersUpgrade.Spec.UpgradeStrategy.MaxConcurrency {
			break
		}

		//TODO: Better to ignore only the completed clusters if the clusters in a single batch all failed we don't want to continue.
		// Ignore failed/complete clusters
		if cluster.OperatorsStatus.UpgradeApproveState == clusterv1beta1.CompleteState ||
			cluster.OperatorsStatus.UpgradeApproveState == clusterv1beta1.FailedState {
			continue
		}

		if cluster.ClusterUpgradeStatus.State == clusterv1beta1.NotStartedState {
			// Create cluster upgrade first
			manifestWork, err := CreateClusterVersionUpgradeManifestWork(cluster.Name, cluster.ClusterID, managedClustersUpgrade.Spec.ClusterVersion)
			if err != nil {
				klog.Info("ClusterVersion ManifestWork not initaliz: ", cluster.Name, " ", err)
			} else {
				klog.Info("Create ManifestWork ", manifestWork.Name)
				err = r.Client.Create(ctx, manifestWork)
				if err != nil {
					klog.Error("Error create ManifestWork: ", manifestWork.Name, " ", err)
				} else {
					cluster.ClusterUpgradeStatus.State = clusterv1beta1.InitializedState
					manifestCreated = true
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
				manifestWork, err := CreateOperatorUpgradeManifestWork(cluster.Name, managedClustersUpgrade.Spec.OcpOperators,
					managedClustersUpgrade.Spec.UpgradeStrategy.OperatorsUpgradeTimeout, OseCliDefaultImage)
				if err != nil {
					klog.Info("OCP Operators ManifestWork not initaliz: ", cluster.Name, " ", err)
					// operators upgrade not valid
					continue
				} else {
					klog.Info("Create ManifestWork ", manifestWork.Name)
					err = r.Client.Create(ctx, manifestWork)
					if err != nil {
						klog.Error("Error create ManifestWork: ", manifestWork.Name, " ", err)
					} else {
						cluster.OperatorsStatus.UpgradeApproveState = clusterv1beta1.InitializedState
						manifestCreated = true
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
	var clusters []*clusterv1beta1.ClusterStatusSpec

	// check if canary cluster upgrade exist or complete
	if len(managedClustersUpgrade.Status.CanaryClusters) == 0 || apimeta.IsStatusConditionTrue(managedClustersUpgrade.Status.Conditions, TypeCanaryComplete) {
		clusters = managedClustersUpgrade.Status.Clusters
	} else if len(managedClustersUpgrade.Status.CanaryClusters) > 0 {
		clusters = managedClustersUpgrade.Status.CanaryClusters
		isCanary = true
	}

	for _, cluster := range clusters {
		if cluster.ClusterUpgradeStatus.State == clusterv1beta1.InitializedState {
			resAvailable, isTimeOut, err := isManifestWorkResourcesAvailable(r.Client, cluster.Name+ClusterUpgradeManifestName, cluster.Name, managedClustersUpgrade.Spec.UpgradeStrategy.ClusterUpgradeTimeout)
			if err != nil {
				klog.Error("Get Manifestwork error ", cluster.Name+ClusterUpgradeManifestName+" ", err)
			}

			if resAvailable {
				inProgress = true

				version, state, verified, message, _, err := getClusterUpgradeManifestStatus(r.Client, cluster.Name+ClusterUpgradeManifestName, cluster.Name, managedClustersUpgrade.Spec.UpgradeStrategy.ClusterUpgradeTimeout)
				if err != nil {
					klog.Error("Error ClusterUpgradeManifestStatus ", err)
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
			klog.Info("I'm in ocp upgrade partial state")
			inProgress = true
			version, state, verified, message, isTimeOut, err := getClusterUpgradeManifestStatus(r.Client, cluster.Name+ClusterUpgradeManifestName, cluster.Name, managedClustersUpgrade.Spec.UpgradeStrategy.ClusterUpgradeTimeout)
			if err != nil {
				klog.Error("Error ClusterUpgradeManifestStatus ", err)
				continue
			}

			klog.Info("ClusterUpgradeManifestStatus ", version+" ", state+" ", verified, " ", message)
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
			resAvailable, isTimeOut, err := isManifestWorkResourcesAvailable(r.Client, cluster.Name+OperatorUpgradeManifestName, cluster.Name, managedClustersUpgrade.Spec.UpgradeStrategy.OperatorsUpgradeTimeout)
			if err != nil {
				klog.Error("Get Manifest error ", cluster.Name+OperatorUpgradeManifestName+" ", err)
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
			klog.Info("I'm in operator upgrade partial state")
			inProgress = true
			status, value, _, err := getOperatorUpgradeManifestStatus(r.Client, cluster.Name+OperatorUpgradeManifestName, cluster.Name, managedClustersUpgrade.Spec.UpgradeStrategy.OperatorsUpgradeTimeout)
			if err != nil {
				klog.Error("Get OperatorUpgradeManifestStatus ", err)
			}

			klog.Info("OperatorUpgradeManifestStatus ", status+" ", value)
			if status == OperatorUpgradeSucceededState && value == OperatorUpgradeValidStateValue {
				cluster.OperatorsStatus.UpgradeApproveState = clusterv1beta1.CompleteState
			} else if status == OperatorUpgradeFailedState && value == OperatorUpgradeValidStateValue {
				cluster.OperatorsStatus.UpgradeApproveState = clusterv1beta1.FailedState
			} else if status == OperatorUpgradeActiveState && value == OperatorUpgradeValidStateValue {
				cluster.OperatorsStatus.UpgradeApproveState = clusterv1beta1.PartialState
			}

		} else if cluster.OperatorsStatus.UpgradeApproveState == clusterv1beta1.CompleteState {
			countComplete++
		}
	}

	// return complete when the count of complete state clusters equal to all clusters and not equal to 0 and not canary clusters
	klog.Info("Process inProgress complete is ", (countComplete == len(clusters) && countComplete > 0 && !isCanary))
	return inProgress, (countComplete == len(clusters) && countComplete > 0 && !isCanary)
}

func (r *ManagedClustersUpgradeReconciler) processCompleteCondition(managedClustersUpgrade *clusterv1beta1.ManagedClustersUpgrade, canaryClusters bool) bool {
	completeAll := true
	var clusters []*clusterv1beta1.ClusterStatusSpec

	if canaryClusters {
		clusters = managedClustersUpgrade.Status.CanaryClusters
	} else {
		clusters = managedClustersUpgrade.Status.Clusters
	}

	for _, cluster := range clusters {
		if cluster.ClusterUpgradeStatus.State == clusterv1beta1.CompleteState {
			// Delete cluster operator upgrade manifest work
			manifestName := cluster.Name + ClusterUpgradeManifestName
			klog.Info("upgrade complete Delete manfiestwork ", manifestName)
			err := deleteManifestWork(r.Client, manifestName, cluster.Name)
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
			manifestName := cluster.Name + OperatorUpgradeManifestName
			klog.Info("upgrade complete Delete manfiestwork ", manifestName)
			err := deleteManifestWork(r.Client, manifestName, cluster.Name)
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

func (r *ManagedClustersUpgradeReconciler) processFailedCondition(managedClustersUpgrade *clusterv1beta1.ManagedClustersUpgrade, canaryClusters bool) (bool, int, string) {
	failedNames := ""
	failedCount := 0
	var clusters []*clusterv1beta1.ClusterStatusSpec

	if canaryClusters {
		clusters = managedClustersUpgrade.Status.CanaryClusters

	} else {
		clusters = managedClustersUpgrade.Status.Clusters
	}

	for _, cluster := range clusters {
		if cluster.ClusterUpgradeStatus.State == clusterv1beta1.FailedState {
			manifestName := cluster.Name + ClusterUpgradeManifestName
			klog.Info("upgrade failed Delete manfiestwork ", manifestName)
			err := deleteManifestWork(r.Client, manifestName, cluster.Name)
			if !apierrors.IsNotFound(err) {
				klog.Error("Delete manifestwork ", manifestName+" ", err)
			}
		}

		if cluster.OperatorsStatus.UpgradeApproveState == clusterv1beta1.FailedState {
			// Delete cluster operator upgrade manifest work
			manifestName := cluster.Name + OperatorUpgradeManifestName
			klog.Info("upgrade failed Delete manfiestwork ", manifestName)
			err := deleteManifestWork(r.Client, manifestName, cluster.Name)
			if !apierrors.IsNotFound(err) {
				klog.Error("Delete manifestwork ", manifestName+" ", err)
			}
		}

		if cluster.ClusterUpgradeStatus.State == clusterv1beta1.FailedState ||
			cluster.OperatorsStatus.UpgradeApproveState == clusterv1beta1.FailedState {
			failedCount++
			failedNames = cluster.Name + " " + failedNames
		}

	}

	// return true when all clusters in failed state
	return (failedCount == len(clusters) && failedCount != 0), failedCount, failedNames
}

func (r *ManagedClustersUpgradeReconciler) initClustersList(kubeClient client.Client, placement clusterv1beta1.GenericPlacementFields) ([]*clusterv1beta1.ClusterStatusSpec, error) {
	Clusters, err := GetManagedClusterList(kubeClient, placement)
	if err != nil {
		return nil, err
	}

	var clustersStatus []*clusterv1beta1.ClusterStatusSpec

	for k, cluster := range Clusters {
		//klog.Info("clusterInfo version:", cluster.Status.DistributionInfo.OCP.Version)
		//klog.Info("clusterInfo versionHistory:", cluster.Status.DistributionInfo.OCP.VersionHistory)
		labels := cluster.GetLabels()
		clusterId := ""
		if len(labels) > 0 {
			clusterId = labels["clusterID"]
		}
		clusterStatus := &clusterv1beta1.ClusterStatusSpec{
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
