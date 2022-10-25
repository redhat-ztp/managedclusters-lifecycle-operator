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

package work

import (
	"context"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/klog"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	//"sigs.k8s.io/controller-runtime/pkg/log"

	workv1beta1 "github.com/redhat-ztp/managedclusters-lifecycle-operator/apis/work/v1beta1"
)

// ManagedClusterGroupWorkReconciler reconciles a ManagedClusterGroupWork object
type ManagedClusterGroupWorkReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=work.open-cluster-management-extension.io,resources=managedclustergroupworks,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=work.open-cluster-management-extension.io,resources=managedclustergroupworks/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=work.open-cluster-management-extension.io,resources=managedclustergroupworks/finalizers,verbs=update
//+kubebuilder:rbac:groups=work.open-cluster-management.io,resources=manifestwork,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the ManagedClusterGroupWork object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.10.0/pkg/reconcile
func (r *ManagedClusterGroupWorkReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {

	klog.Info("Hi, ManagedClusterGroupWork controller")
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ManagedClusterGroupWorkReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&workv1beta1.ManagedClusterGroupWork{}).
		Complete(r)
}
