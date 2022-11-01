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
	configv1 "github.com/openshift/api/config/v1"
	common "github.com/redhat-ztp/managedclusters-lifecycle-operator/apis/common/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

const (
	NotStartedState  = "NotStarted"
	InitializedState = "Initialized"
	PartialState     = string(configv1.PartialUpdate)
	CompleteState    = string(configv1.CompletedUpdate)
	FailedState      = "Failed"
)

// Generic Operator Reference
type GenericOperatorReference struct {
	// Operator->Subscription Name
	Name string `json:"name"`
	// Operator->Subscription Namespace
	Namespace string `json:"namespace"`
}

// UpgradeStrategy defines the upgrades Strategy
type UpgradeStrategySpec struct {
	// CanaryClusters defines the list of managedClusters that should be remediated first
	CanaryClusters common.GenericPlacementFields `json:"canaryClusters,omitempty"`
	//kubebuilder:validation:Minimum=1
	//+kubebuilder:default=10
	// Max number of clusters to perform upgrade at the same time
	MaxConcurrency int `json:"maxConcurrency"`
	//+kubebuilder:default="3h"
	// TimeOut for cluster upgrade process in hours default is (3h) or in min ex; 180m
	ClusterUpgradeTimeout string `json:"clusterUpgradeTimeout,omitempty"`
	//+kubebuilder:default="20m"
	// TimeOut for operators upgrade process in min default is (20m) or in hours ex; 1h
	OperatorsUpgradeTimeout string `json:"operatorsUpgradeTimeout,omitempty"`
}

// ClusterVersion define the desired state of ClusterVersion
type ClusterVersionSpec struct {
	// version for cluster upgrade
	Version string `json:"version"`
	// channel name for cluster upgrade
	Channel string `json:"channel"`
	// +optional
	// upstream url for cluster upgrade
	Upstream string `json:"upstream,omitempty"`
	// +optional
	// Image url for cluster upgrade
	Image string `json:"image,omitempty"`
}

// OcpOperators define the desire action for ocp operator upgrade
type OcpOperatorsSpec struct {
	// ApproveAllUpgrade is a boolean flag to approve all the installed operators installPlan for the selected
	// managed clusters after cluster upgrade is done. When set to false only selected operator in the include list
	// will be approved
	//+kubebuilder:default=false
	ApproveAllUpgrades bool `json:"approveAllUpgrades"`
	// List of the selected operators to approve its installPlan after cluster upgrade done.
	Include []GenericOperatorReference `json:"include,omitempty"`
	// List of the selected operators to not approve its installPlan after cluster upgrade done.
	Exclude []GenericOperatorReference `json:"exclude,omitempty"`
}

// ManagedClustersUpgrade defines the desired state of ManagedClustersUpgrade
type ManagedClustersUpgradeSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster

	// Skip using placement for now as clusterID is a required attribute to get
	// Placement common.GenericPlacementReference `json:"placement,omitempty"`

	// +optional
	common.GenericPlacementFields `json:",inline"`

	UpgradeStrategy *UpgradeStrategySpec `json:"upgradeStrategy,omitempty"`
	// +optional
	ClusterVersion *ClusterVersionSpec `json:"clusterVersion,omitempty"`
	// +optional
	OcpOperators *OcpOperatorsSpec `json:"ocpOperators,omitempty"`
}

// ClusterUpgradeStatus is a single attempted update to the cluster.
// The original definition is from https://github.com/openshift/api/blob/master/config/v1/types_cluster_version.go
type ClusterUpgradeStatus struct {
	// state reflects whether the update was fully applied. The NotStart state
	// indicates the update is not start yet. The Initialized state indicates the update
	// has been initialized for the spoke. The Partial state indicates the update is not fully applied.
	// while the Completed state indicates the update was successfully rolled out at least once (all parts of the update successfully applied).
	State string `json:"state,omitempty"`

	// verified indicates whether the provided update was properly verified
	// before it was installed. If this is false the cluster may not be trusted.
	//+kubebuilder:default=false
	Verified bool `json:"verified,omitempty"`

	// message reflect the upgrade state output
	Message string `json:"message,omitempty"`
}

// OperatorStatus indicate that operators installPlan approved
type OperatorsStatus struct {
	// UpgradeApproveState reflects whether the operator install plan approve fully applied. The NotStart state
	// indicates the install plan approve not start yet. The Initialized state indicates the install plan approve
	// has been initialized. The Partial state indicates the install plan approve is not fully applied
	// while the Completed state indicates install plan approve is running
	UpgradeApproveState string `json:"upgradeApproveState,omitempty"`
}

// ClusterStatus indicate the selected clusters upgrade status
type ClusterStatus struct {
	// ManagedCluster generated ID
	ClusterID string `json:"clusterID"`
	// ManagedCluster Name
	Name string `json:"name"`
	// ManagedCluster upgrade status
	ClusterUpgradeStatus ClusterUpgradeStatus `json:"clusterUpgradeStatus,omitempty"`
	// ManagedCluster Operators Upgrade status
	OperatorsStatus OperatorsStatus `json:"operatorsStatus,omitempty"`
}

// ManagedClustersUpgradeStatus defines the observed state of ManagedClustersUpgrade
type ManagedClustersUpgradeStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Conditions contains the different condition statuses for ManagedClustersUpgrade
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// List of the selected managedClusters with its upgrade status
	Clusters []*ClusterStatus `json:"clusters,omitempty"`

	// List of the selected Canary managedClusters with its upgrade status
	CanaryClusters []*ClusterStatus `json:"canaryClusters,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:resource:path=managedclustersupgrades,shortName=mcu

// ManagedClustersUpgrade is the Schema for the managedclustersupgrades API
type ManagedClustersUpgrade struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ManagedClustersUpgradeSpec   `json:"spec,omitempty"`
	Status ManagedClustersUpgradeStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// ManagedClustersUpgradeList contains a list of ManagedClustersUpgrade
type ManagedClustersUpgradeList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ManagedClustersUpgrade `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ManagedClustersUpgrade{}, &ManagedClustersUpgradeList{})
}
