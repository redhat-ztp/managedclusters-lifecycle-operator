package cluster

import (
	"fmt"
	clusterv1beta1 "github.com/redhat-ztp/managedclusters-lifecycle-operator/apis/cluster/v1beta1"
	"github.com/stretchr/testify/assert"
	"testing"
)

func Test_CreateOperatorUpgradeManifestWork(t *testing.T) {
	clusterName := "testName"
	operatorConfig := &clusterv1beta1.OcpOperatorsSpec{
		ApproveAllUpgrades: false,
	}
	// Check for invalid cluster name
	_, err := CreateOperatorUpgradeManifestWork("", operatorConfig, "30m", OseCliDefaultImage)
	assert.NotNil(t, err)
	assert.Equal(t, fmt.Errorf("Invalid clusterName"), err)
	// Check for invalid operators config
	_, err = CreateOperatorUpgradeManifestWork(clusterName, operatorConfig, "30m", OseCliDefaultImage)
	assert.NotNil(t, err)
	assert.Equal(t, fmt.Errorf("Invalid operatorConfig at least an operator should be selected to upgrade"), err)

	// set include operator
	operatorConfig.Include = []clusterv1beta1.GenericOperatorReference{
		clusterv1beta1.GenericOperatorReference{
			Name:      "test",
			Namespace: "test-ns",
		},
	}

	// Check for invalid timeout
	_, err = CreateOperatorUpgradeManifestWork(clusterName, operatorConfig, "30mins", OseCliDefaultImage)
	assert.NotNil(t, err)
	assert.Equal(t, fmt.Errorf("Invalid Operators upgrade timeout %s", "30mins"), err)

	// Check for expected mw name, namespace and num of manifests 5 (ns, clusterRole, clusterBinding, serviceAccount, job)
	mw_operator, err := CreateOperatorUpgradeManifestWork(clusterName, operatorConfig, "30m", OseCliDefaultImage)
	assert.Nil(t, err)
	assert.Equal(t, mw_operator.Name, clusterName+OperatorUpgradeManifestName)
	assert.Equal(t, mw_operator.Namespace, clusterName)
	assert.Equal(t, len(mw_operator.Spec.Workload.Manifests), 5)
}

func Test_getInstallPlanApproverJob(t *testing.T) {
	includeOperName, excludeOperName, ns := "test-include", "test-exclude", "test-ns"

	includeOper := []clusterv1beta1.GenericOperatorReference{
		clusterv1beta1.GenericOperatorReference{
			Name:      includeOperName,
			Namespace: ns,
		},
	}

	excludeOper := []clusterv1beta1.GenericOperatorReference{
		clusterv1beta1.GenericOperatorReference{
			Name:      excludeOperName,
			Namespace: ns,
		},
	}

	// check for job env variable
	job, err := getInstallPlanApproverJob(includeOper, excludeOper, "1m", OseCliDefaultImage)
	assert.Nil(t, err)
	for _, env := range job.Spec.Template.Spec.Containers[0].Env {
		if env.Name == "EXCLUDELIST" {
			assert.Equal(t, env.Value, excludeOperName+","+ns)
		}
		if env.Name == "INCLUDELIST" {
			assert.Equal(t, env.Value, includeOperName+","+ns)
		}
		// check for wait time converted to sec
		if env.Name == "WAITTIME" {
			assert.Equal(t, env.Value, "60")
		}
	}
	fmt.Println(job)
}

func Test_CreateClusterVersionUpgradeManifestWork(t *testing.T) {
	clusterName := "testName"
	versionConfig := &clusterv1beta1.ClusterVersionSpec{
		Version: "",
	}

	// check for invalid cluster name
	_, err := CreateClusterVersionUpgradeManifestWork("", "", versionConfig)
	assert.NotNil(t, err)
	assert.Equal(t, fmt.Errorf("Invalid clusterName"), err)

	// check for invalid cluster id
	_, err = CreateClusterVersionUpgradeManifestWork(clusterName, "", versionConfig)
	assert.NotNil(t, err)
	assert.Equal(t, fmt.Errorf("Invalid clusterId"), err)

	// check for invalid cluster version
	_, err = CreateClusterVersionUpgradeManifestWork(clusterName, "234", versionConfig)
	assert.NotNil(t, err)
	assert.Equal(t, fmt.Errorf("Invalid version"), err)

	// Check for expected mw name, namespace and num of manifests 2 (clusterVersion & roleBinding)
	versionConfig.Version = "v4.99"
	mw_upgrade, err := CreateClusterVersionUpgradeManifestWork(clusterName, "234", versionConfig)
	assert.Nil(t, err)
	assert.Equal(t, mw_upgrade.Name, clusterName+ClusterUpgradeManifestName)
	assert.Equal(t, mw_upgrade.Namespace, clusterName)
	assert.Equal(t, len(mw_upgrade.Spec.Workload.Manifests), 2)
}
