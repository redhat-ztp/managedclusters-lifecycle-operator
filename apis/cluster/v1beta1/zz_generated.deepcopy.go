//go:build !ignore_autogenerated
// +build !ignore_autogenerated

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

// Code generated by controller-gen. DO NOT EDIT.

package v1beta1

import (
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
)

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ClusterStatusSpec) DeepCopyInto(out *ClusterStatusSpec) {
	*out = *in
	out.ClusterUpgradeStatus = in.ClusterUpgradeStatus
	out.OperatorsStatus = in.OperatorsStatus
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ClusterStatusSpec.
func (in *ClusterStatusSpec) DeepCopy() *ClusterStatusSpec {
	if in == nil {
		return nil
	}
	out := new(ClusterStatusSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ClusterUpgradeStatus) DeepCopyInto(out *ClusterUpgradeStatus) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ClusterUpgradeStatus.
func (in *ClusterUpgradeStatus) DeepCopy() *ClusterUpgradeStatus {
	if in == nil {
		return nil
	}
	out := new(ClusterUpgradeStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ClusterVersionSpec) DeepCopyInto(out *ClusterVersionSpec) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ClusterVersionSpec.
func (in *ClusterVersionSpec) DeepCopy() *ClusterVersionSpec {
	if in == nil {
		return nil
	}
	out := new(ClusterVersionSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *GenericClusterReference) DeepCopyInto(out *GenericClusterReference) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new GenericClusterReference.
func (in *GenericClusterReference) DeepCopy() *GenericClusterReference {
	if in == nil {
		return nil
	}
	out := new(GenericClusterReference)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *GenericOperatorReference) DeepCopyInto(out *GenericOperatorReference) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new GenericOperatorReference.
func (in *GenericOperatorReference) DeepCopy() *GenericOperatorReference {
	if in == nil {
		return nil
	}
	out := new(GenericOperatorReference)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *GenericPlacementFields) DeepCopyInto(out *GenericPlacementFields) {
	*out = *in
	if in.Clusters != nil {
		in, out := &in.Clusters, &out.Clusters
		*out = make([]GenericClusterReference, len(*in))
		copy(*out, *in)
	}
	if in.ClusterSelector != nil {
		in, out := &in.ClusterSelector, &out.ClusterSelector
		*out = new(v1.LabelSelector)
		(*in).DeepCopyInto(*out)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new GenericPlacementFields.
func (in *GenericPlacementFields) DeepCopy() *GenericPlacementFields {
	if in == nil {
		return nil
	}
	out := new(GenericPlacementFields)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *LabelAction) DeepCopyInto(out *LabelAction) {
	*out = *in
	if in.AddClusterLabels != nil {
		in, out := &in.AddClusterLabels, &out.AddClusterLabels
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
	if in.DeleteClusterLabels != nil {
		in, out := &in.DeleteClusterLabels, &out.DeleteClusterLabels
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new LabelAction.
func (in *LabelAction) DeepCopy() *LabelAction {
	if in == nil {
		return nil
	}
	out := new(LabelAction)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ManagedClusterLabelActionSpec) DeepCopyInto(out *ManagedClusterLabelActionSpec) {
	*out = *in
	in.BeforeUpgrade.DeepCopyInto(&out.BeforeUpgrade)
	in.AfterClusterUpgrade.DeepCopyInto(&out.AfterClusterUpgrade)
	in.AfterUpgrade.DeepCopyInto(&out.AfterUpgrade)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ManagedClusterLabelActionSpec.
func (in *ManagedClusterLabelActionSpec) DeepCopy() *ManagedClusterLabelActionSpec {
	if in == nil {
		return nil
	}
	out := new(ManagedClusterLabelActionSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ManagedClustersUpgrade) DeepCopyInto(out *ManagedClustersUpgrade) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ManagedClustersUpgrade.
func (in *ManagedClustersUpgrade) DeepCopy() *ManagedClustersUpgrade {
	if in == nil {
		return nil
	}
	out := new(ManagedClustersUpgrade)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *ManagedClustersUpgrade) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ManagedClustersUpgradeList) DeepCopyInto(out *ManagedClustersUpgradeList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]ManagedClustersUpgrade, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ManagedClustersUpgradeList.
func (in *ManagedClustersUpgradeList) DeepCopy() *ManagedClustersUpgradeList {
	if in == nil {
		return nil
	}
	out := new(ManagedClustersUpgradeList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *ManagedClustersUpgradeList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ManagedClustersUpgradeSpec) DeepCopyInto(out *ManagedClustersUpgradeSpec) {
	*out = *in
	in.GenericPlacementFields.DeepCopyInto(&out.GenericPlacementFields)
	if in.UpgradeStrategy != nil {
		in, out := &in.UpgradeStrategy, &out.UpgradeStrategy
		*out = new(UpgradeStrategySpec)
		(*in).DeepCopyInto(*out)
	}
	if in.ClusterVersion != nil {
		in, out := &in.ClusterVersion, &out.ClusterVersion
		*out = new(ClusterVersionSpec)
		**out = **in
	}
	if in.OcpOperators != nil {
		in, out := &in.OcpOperators, &out.OcpOperators
		*out = new(OcpOperatorsSpec)
		(*in).DeepCopyInto(*out)
	}
	if in.ManagedClusterLabelAction != nil {
		in, out := &in.ManagedClusterLabelAction, &out.ManagedClusterLabelAction
		*out = new(ManagedClusterLabelActionSpec)
		(*in).DeepCopyInto(*out)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ManagedClustersUpgradeSpec.
func (in *ManagedClustersUpgradeSpec) DeepCopy() *ManagedClustersUpgradeSpec {
	if in == nil {
		return nil
	}
	out := new(ManagedClustersUpgradeSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ManagedClustersUpgradeStatus) DeepCopyInto(out *ManagedClustersUpgradeStatus) {
	*out = *in
	if in.Conditions != nil {
		in, out := &in.Conditions, &out.Conditions
		*out = make([]v1.Condition, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.Clusters != nil {
		in, out := &in.Clusters, &out.Clusters
		*out = make([]*ClusterStatusSpec, len(*in))
		for i := range *in {
			if (*in)[i] != nil {
				in, out := &(*in)[i], &(*out)[i]
				*out = new(ClusterStatusSpec)
				**out = **in
			}
		}
	}
	if in.CanaryClusters != nil {
		in, out := &in.CanaryClusters, &out.CanaryClusters
		*out = make([]*ClusterStatusSpec, len(*in))
		for i := range *in {
			if (*in)[i] != nil {
				in, out := &(*in)[i], &(*out)[i]
				*out = new(ClusterStatusSpec)
				**out = **in
			}
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ManagedClustersUpgradeStatus.
func (in *ManagedClustersUpgradeStatus) DeepCopy() *ManagedClustersUpgradeStatus {
	if in == nil {
		return nil
	}
	out := new(ManagedClustersUpgradeStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *OcpOperatorsSpec) DeepCopyInto(out *OcpOperatorsSpec) {
	*out = *in
	if in.Include != nil {
		in, out := &in.Include, &out.Include
		*out = make([]GenericOperatorReference, len(*in))
		copy(*out, *in)
	}
	if in.Exclude != nil {
		in, out := &in.Exclude, &out.Exclude
		*out = make([]GenericOperatorReference, len(*in))
		copy(*out, *in)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new OcpOperatorsSpec.
func (in *OcpOperatorsSpec) DeepCopy() *OcpOperatorsSpec {
	if in == nil {
		return nil
	}
	out := new(OcpOperatorsSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *OperatorsStatus) DeepCopyInto(out *OperatorsStatus) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new OperatorsStatus.
func (in *OperatorsStatus) DeepCopy() *OperatorsStatus {
	if in == nil {
		return nil
	}
	out := new(OperatorsStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *UpgradeStrategySpec) DeepCopyInto(out *UpgradeStrategySpec) {
	*out = *in
	in.CanaryClusters.DeepCopyInto(&out.CanaryClusters)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new UpgradeStrategySpec.
func (in *UpgradeStrategySpec) DeepCopy() *UpgradeStrategySpec {
	if in == nil {
		return nil
	}
	out := new(UpgradeStrategySpec)
	in.DeepCopyInto(out)
	return out
}
