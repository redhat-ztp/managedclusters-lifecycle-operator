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

package v1beta1

import (
	clusterv1beta1 "github.com/redhat-ztp/managedclusters-lifecycle-operator/apis/cluster/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	workv1 "open-cluster-management.io/api/work/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// ManagedClusterGroupWorkSpec defines the desired state of ManagedClusterGroupWork
type ManagedClusterGroupWorkSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Important: Run "make" to regenerate code after modifying this file
	clusterv1beta1.GenericPlacementFields `json:",inline"`
	// List of manifestWorks to be applied on the selected clusters
	// +optional
	ManifesWorks []workv1.ManifestWork `json:"manifesWorks,omitempty"`
}

// ClusterWorkState indicate the selected clusters manifest work status
type ClusterWorkState struct {
	// ManagedCluster Name
	Name string `json:"name"`
	// ManagedCluster manifest work resources state
	ManifestWorkState string `json:"manifestWorkState,omitempty"`
	// ManagedCluster manifest work feedback fields
	ManifestWorkFeedback map[string]string `json:"manifestWorkFeedback,omitempty"`
}

// ManagedClusterGroupWorkStatus defines the observed state of ManagedClusterGroupWork
type ManagedClusterGroupWorkStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	//List of the ManagedClusterGroupAct conditions
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// List of the selected managedClusters with act status
	Clusters []*ClusterWorkState `json:"clusters,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// ManagedClusterGroupWork is the Schema for the managedclustergroupworks API
type ManagedClusterGroupWork struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ManagedClusterGroupWorkSpec   `json:"spec,omitempty"`
	Status ManagedClusterGroupWorkStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// ManagedClusterGroupWorkList contains a list of ManagedClusterGroupWork
type ManagedClusterGroupWorkList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ManagedClusterGroupWork `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ManagedClusterGroupWork{}, &ManagedClusterGroupWorkList{})
}
