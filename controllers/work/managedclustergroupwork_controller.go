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

	workv1beta1 "github.com/redhat-ztp/managedclusters-lifecycle-operator/apis/work/v1beta1"
	utils "github.com/redhat-ztp/managedclusters-lifecycle-operator/controllers/utils"
	workv1 "open-cluster-management.io/api/work/v1"
)

const mcgWorkFinalizer = "managedclustergroupwork.work.open-cluster-management-extension.io/finalizer"

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
	mcgWork := &workv1beta1.ManagedClusterGroupWork{}

	result := ctrl.Result{RequeueAfter: 20 * time.Second}
	err := r.Get(ctx, req.NamespacedName, mcgWork)

	if err != nil {
		if apierrors.IsNotFound(err) {
			klog.Info("ManagedClusterGroupWork resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		klog.Error("Failed to get ManagedClusterGroupWork ", err)
		return result, err
	} else if mcgWork.GetDeletionTimestamp() != nil && ctrlutil.ContainsFinalizer(mcgWork, mcgWorkFinalizer) {
		if err := r.finalizeMcgWork(mcgWork); err != nil {
			klog.Error("Failed to finaliz ManagedClusterGroupWork ", err)
			return result, err
		}
		ctrlutil.RemoveFinalizer(mcgWork, mcgWorkFinalizer)
		err := r.Update(ctx, mcgWork)
		if err != nil {
			klog.Error("Failed update ManagedClusterGroupWork finalizer ", err)
			return result, err
		}
	} else {
		//klog.Info(mcgWork.Name)
		// Condition; TypeSelected
		clustersCount := r.processSelectedCondition(mcgWork)
		apimeta.SetStatusCondition(&mcgWork.Status.Conditions, utils.GetSelectedCondition(clustersCount))
		allCreated, allApplied, allAvailable := r.processAppliedCondition(ctx, mcgWork)

		// Condition; TypeCreated, TypeApplied and TypeAvailable
		if allCreated {
			apimeta.SetStatusCondition(&mcgWork.Status.Conditions, utils.GetCreatedCondition(metav1.ConditionTrue))
		} else {
			apimeta.SetStatusCondition(&mcgWork.Status.Conditions, utils.GetCreatedCondition(metav1.ConditionFalse))
		}

		if allApplied {
			apimeta.SetStatusCondition(&mcgWork.Status.Conditions, utils.GetAppliedCondition(metav1.ConditionTrue))
		} else {
			apimeta.SetStatusCondition(&mcgWork.Status.Conditions, utils.GetAppliedCondition(metav1.ConditionFalse))
		}

		if allAvailable {
			apimeta.SetStatusCondition(&mcgWork.Status.Conditions, utils.GetAvailableCondition(metav1.ConditionTrue))
		} else {
			apimeta.SetStatusCondition(&mcgWork.Status.Conditions, utils.GetAvailableCondition(metav1.ConditionFalse))
		}

		if !ctrlutil.ContainsFinalizer(mcgWork, mcgWorkFinalizer) {
			ctrlutil.AddFinalizer(mcgWork, mcgWorkFinalizer)
			err = r.Update(ctx, mcgWork)
		} else {
			err = r.Status().Update(ctx, mcgWork)
		}

		if err != nil {
			klog.Error("Failed update managedClusterGroupAct ", err)
			return result, err
		}
	}

	return result, nil
}

func (r *ManagedClusterGroupWorkReconciler) processSelectedCondition(mcgWork *workv1beta1.ManagedClusterGroupWork) int {
	existingClusters := sets.NewString()
	if len(mcgWork.Status.Clusters) > 0 {
		for _, cluster := range mcgWork.Status.Clusters {
			existingClusters.Insert(cluster.Name)
		}
	}

	addedClusters, deletedClusters := sets.NewString(), sets.NewString()
	var err error
	if mcgWork.Spec.Placement.Name != "" && mcgWork.Spec.Placement.Namespace != "" {
		placement, err := utils.GetPlacement(r.Client, mcgWork.Spec.Placement.Name, mcgWork.Spec.Placement.Namespace)
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
	} else if mcgWork.Spec.GenericPlacementFields.ClusterSelector != nil || len(mcgWork.Spec.GenericPlacementFields.Clusters) != 0 {
		_, addedClusters, deletedClusters, err = utils.GetManagedClusterList(r.Client, mcgWork.Spec.GenericPlacementFields, existingClusters)
		if err != nil {
			klog.Error("Cannot get clusters ", err)
		}
	}

	clusters := mcgWork.Status.Clusters
	indx := 0
	if deletedClusters.Len() > 0 {
		for _, cls := range clusters {
			if deletedClusters.Has(cls.Name) {
				// Delete Created manifest
				err = utils.DeleteManifestWork(r.Client, mcgWork.Name, cls.Name)
				if err != nil {
					klog.Error("Faild to delete ", err)
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
			clusters = append(clusters, &workv1beta1.ClusterWorkState{
				Name: key,
			})
		}
	}

	mcgWork.Status.Clusters = clusters

	return len(mcgWork.Status.Clusters)
}

func (r *ManagedClusterGroupWorkReconciler) processAppliedCondition(ctx context.Context, mcgWork *workv1beta1.ManagedClusterGroupWork) (bool, bool, bool) {
	mManifestWorks, err := utils.GetManifestWorks(r.Client, mcgWork.Namespace+"-"+mcgWork.Name, "")
	if err != nil {
		klog.Error("Failed to get manifestWorks, ", err, " lenght is ", len(mManifestWorks))
	}

	allApplied, allAvailable, allCreated := true, true, true
	clusters := mcgWork.Status.Clusters
	for _, cluster := range clusters {
		if cluster.ManifestState == "" || cluster.ManifestState == utils.StateNotCreated {
			allApplied, allAvailable = false, false
			manifestWork, err := utils.CreateManifestWork(mcgWork, cluster.Name)
			if err != nil {
				klog.Error("Failed to create manifestWork, ", err)
				cluster.ManifestState = utils.StateNotCreated
				allCreated = false
				continue
			}

			err = r.Client.Create(ctx, manifestWork)
			if err != nil {
				klog.Error("Failed create mClusterAction: ", err)
				cluster.ManifestState = utils.StateNotCreated
				allCreated = false
				continue
			}
			cluster.ManifestState = utils.StateCreated
		} else {
			if manifestwork, ok := mManifestWorks[cluster.Name+"-"+mcgWork.Name]; ok {
				if !utils.EqualManifestWorkSpec(manifestwork.Spec, mcgWork.Spec.ManifestWork) {
					manifestwork.Spec = mcgWork.Spec.ManifestWork
					err = r.Client.Update(ctx, manifestwork)

					if err != nil {
						klog.Error("Faild to update Manifestwork ", err)
					}
					cluster.ManifestState = utils.StateCreated
				} else {
					status := utils.GetManifestWorkStatus(manifestwork)
					if status != workv1.WorkApplied && status != workv1.WorkAvailable {
						allApplied = false
					}
					if status != workv1.WorkAvailable {
						allAvailable = false
					}
					cluster.ManifestState = status
				}
			}
		}
	}

	mcgWork.Status.Clusters = clusters
	return allCreated, allApplied, allAvailable
}

func (r *ManagedClusterGroupWorkReconciler) finalizeMcgWork(mcgWork *workv1beta1.ManagedClusterGroupWork) error {
	return utils.DeleteManifestWorks(r.Client, mcgWork.Namespace+"-"+mcgWork.Name)
}

// SetupWithManager sets up the controller with the Manager.
func (r *ManagedClusterGroupWorkReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&workv1beta1.ManagedClusterGroupWork{}).
		Complete(r)
}
