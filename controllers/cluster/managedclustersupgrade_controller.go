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
	"fmt"

	clusterv1beta1 "github.com/redhat-ztp/managedclusters-lifecycle-operator/apis/cluster/v1beta1"
	//clusterv1 "open-cluster-management.io/api/cluster/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/klog"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
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
		klog.Info(managedClustersUpgrade.Name)
		if len(managedClustersUpgrade.Status.Conditions) == 0 {

			condition := metav1.Condition{}
			clustersStatus, err := r.initClustersList(r.Client, managedClustersUpgrade.Spec.GenericPlacementFields, &condition)
			if err != nil {
				klog.Error("Listing clusters: ", err)
				return ctrl.Result{}, nil
			}

			managedClustersUpgrade.Status.Clusters = clustersStatus
			apimeta.SetStatusCondition(&managedClustersUpgrade.Status.Conditions, condition)
			//klog.Info("ManagedClustersUpgrade", managedClustersUpgrade)
			err = r.Status().Update(ctx, managedClustersUpgrade)
			if err != nil {
				klog.Error("Update managedClustersUpgrade.Status:", err)
				return ctrl.Result{}, nil
			}
		} else {

			//var conditions []metav1.Condition
			for _, condition := range managedClustersUpgrade.Status.Conditions {
				if condition.Type == TypeSelected && condition.Reason == ReasonNotSelected && condition.Status == metav1.ConditionFalse {
					clustersStatus, err := r.initClustersList(r.Client, managedClustersUpgrade.Spec.GenericPlacementFields, &condition)
					if err != nil {
						klog.Error("Listing clusters: ", err)
						return ctrl.Result{}, nil
					}

					managedClustersUpgrade.Status.Clusters = clustersStatus
					klog.Info("condition", condition)
					apimeta.SetStatusCondition(&managedClustersUpgrade.Status.Conditions, condition)
				}
			}
			//klog.Info("ManagedClustersUpgrade", managedClustersUpgrade)
			err = r.Status().Update(ctx, managedClustersUpgrade)
			if err != nil {
				klog.Error("Update managedClustersUpgrade.Status:", err)
				return ctrl.Result{}, nil
			}
		}

	}

	return ctrl.Result{}, nil
}

func (r *ManagedClustersUpgradeReconciler) initClustersList(kubeClient client.Client, placement clusterv1beta1.GenericPlacementFields, condition *metav1.Condition) ([]clusterv1beta1.ClusterStatusSpec, error) {
	Clusters, err := GetManagedClusterInfos(kubeClient, placement)
	if err != nil {
		return nil, err
	}

	var clustersStatus []clusterv1beta1.ClusterStatusSpec

	for k, _ := range Clusters {
		clusterStatus := clusterv1beta1.ClusterStatusSpec{
			Name: k,
			ClusterVersionStatus: clusterv1beta1.ClusterVersionStatus{
				State: clusterv1beta1.NotStartedState,
			},
		}

		clustersStatus = append(clustersStatus, clusterStatus)
	}

	if len(clustersStatus) == 0 {
		condition.Status = metav1.ConditionFalse
		condition.Reason = ReasonNotSelected
	} else {
		condition.Status = metav1.ConditionTrue
		condition.Reason = ReasonSelected
	}

	condition.Message = fmt.Sprintf("Cluster Selector has match %d ManagedClusters.", len(clustersStatus))
	condition.Type = TypeSelected
	condition.LastTransitionTime = metav1.Now()

	return clustersStatus, err
}

// SetupWithManager sets up the controller with the Manager.
func (r *ManagedClustersUpgradeReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&clusterv1beta1.ManagedClustersUpgrade{}).
		Complete(r)
}
