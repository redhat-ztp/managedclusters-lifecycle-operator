# Managed Clusters LifeCycle Operator (MCL Operator)

MCL (Managed Clusters LifeCycle) operator manages the OpenShift clusters and its Subscriptions (operators) upgrades. 
In order to perform platform upgrade to group of managed clusters using RHACM, the ClusterVersion CR (version) with the desired upgrade configurations as the example below needs to be applied at each managed cluster.
```
apiVersion: config.openshift.io/v1
kind: ClusterVersion
metadata:
  name: version
spec:
  channel: stable-4.9
  desiredUpdate:
    version: 4.9.33
```
MCL operator uses RHACM to select group of managed clusters based on label or name and apply the cluster version configuration on the selected clusters.
MCL operator does not set the cluster's subscription version. However, it approves the install plan for subscriptions to apply the upgrades.
The ManagedClustersUpgrade CRD defines the match expression for cluster selector and the cluster version configurations. 
The example below select group of clusters with label common is true, set the cluster version to v4.9.22 and approve the install plan for all subscriptions that installed on the managed cluster. 

```
apiVersion: cluster.open-cluster-management-extension.io/v1beta1
kind: ManagedClustersUpgrade
metadata:
  name: example
  namespace: default
spec:
  clusterSelector:
    matchExpressions:
      - key: common
        operator: In
        values:
          - 'true'
  clusterVersion:
    channel: stable-4.9
    version: 4.9.22
  ocpOperators:
    approveAllUpgrades: true
```

Other example below; select group of cluster with type is prod and having a canary cluster name is ocp-prod-backup. It set the cluster version to v4.10.8 and approve only the subscription local-storage-operator upgrade.  
```
apiVersion: cluster.open-cluster-management-extension.io/v1beta1
kind: ManagedClustersUpgrade
metadata:
  name: prod-upgrade
  namespace: default
spec:
  clusterSelector:
    matchExpressions:
      - key: type
        operator: In
        values:
          - 'prod'
  clusterVersion:
    channel: stable-4.10
    version: 4.10.8
  ocpOperators:
    approveAllUpgrades: false
    include:
      - name: local-storage-operator
        namespace: openshift-local-storage
  upgradeStrategy:
    canaryClusters:
      clusterSelector:
        matchExpressions:
          - key: name
            operator: In
            values:
            - ocp-prod-backup
    clusterUpgradeTimeout: 2h
    maxConcurrency: 4
    operatorsUpgradeTimeout: 20m
```

## How it works

MCL operator uses the ManagedClustersUpgrade CR to select the managed clusters, clusterVersion configuration and subscriptions upgrade. 
The MCL operator use the RHACM ManifestWork API to creates a manifestwork for each managed cluster to apply the clusterVersion configuration and continue monitoring the upgrade status till it complete or fail.
The managed cluster upgrade process has 5 states;

  - **NotStart** state indicates the update is not start yet. 
  - **Initialized** state indicates the update has been initialized for the managed cluster. 
  - **Partial** state indicates the update is not fully applied.
  - **Completed** state indicates the update was successfully rolled out at least once.
  - **Failed** state indicates the update did not successfully applied to the managed cluster.

After the managed cluster upgrade complete successfully, MCL operator will creates another ManifestWork for each managed cluster to approve the install plan for the selected subscriptions. 
The example below show the ManagedClustersUpgrade CR status for a selected group of clusters

```
apiVersion: cluster.open-cluster-management-extension.io/v1beta1
kind: ManagedClustersUpgrade
metadata:
  name: example-prod
  namespace: default
spec:
  clusterSelector:
    matchExpressions:
      - key: type
        operator: In
        values:
          - 'prod'
  clusterVersion:
    channel: stable-4.9
    version: 4.9.22
  ocpOperators:
    approveAllUpgrades: true
status
  clusters:
    - clusterID: 55e170c1-7a3c-4ae1-a85d-8bd904ad783f
      clusterUpgradeStatus:
        state: Partial
        verified: true
      name: prod3
      operatorsStatus:
        upgradeApproveState: NotStart  
    - clusterID: 44e170c1-7a3c-4ae1-a85d-8bd904ad783f
      clusterUpgradeStatus:
        state: Completed
        verified: true
      name: prod2
      operatorsStatus:
        upgradeApproveState: Partial
    - clusterID: 34e170c1-7a3c-4ae1-a85d-8bd904ad783f
      clusterUpgradeStatus:
        state: Completed
        verified: true
      name: prod1
      operatorsStatus:
        upgradeApproveState: Completed
  conditions:
    - lastTransitionTime: '2022-05-16T17:14:15Z'
      message: ManagedClsuters upgrade select 3 clusters
      reason: ManagedClustersSelected
      status: 'True'
      type: Selected
    - lastTransitionTime: '2022-05-16T17:14:15Z'
      message: ManagedClsuters upgrade applied
      reason: ManagedClustersUpgradeApplied
      status: 'True'
      type: Applied
    - lastTransitionTime: '2022-05-16T17:14:25Z'
      message: ManagedClsuters upgrade InProgress
      reason: ManagedClustersUpgradeComplete
      status: 'True'
      type: InProgress
    - lastTransitionTime: '2022-05-16T17:14:45Z'
      message: ManagedClsuters upgrade InProgress
      reason: ManagedClustersUpgradeComplete
      status: 'False'
      type: Complete
```
## Run and install

1. Export **KUBECONFIG** environment variable to point to your cluster running RHACM
2. Run **$ make install run**

## Build and deploy

1. Run **$ make docker-build docker-push IMG=*your_repo_image***
2. Run **$ make deploy IMG=*your_repo_image***



