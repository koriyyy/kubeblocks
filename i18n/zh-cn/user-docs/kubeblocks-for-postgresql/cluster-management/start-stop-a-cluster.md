---
title: 停止/启动集群
description: 如何停止/启动集群
keywords: [postgresql, 停止集群, 启动集群]
sidebar_position: 5
sidebar_label: 停止/启动
---

# 停止/启动集群

你可以停止/启动集群以释放计算资源。当集群被停止时，其计算资源将被释放，也就是说 Kubernetes 的 Pod 将被释放，但其存储资源仍将被保留。如果你希望通过快照从原始存储中恢复集群资源，请重新启动该集群。

## 停止集群

### 选项 1.（推荐）使用 kbcli

配置集群名称，并执行以下命令来停止该集群。

```bash
kbcli cluster stop <name>
```

***示例***

```bash
kbcli cluster stop pg-cluster
```

### 选项 2. 创建 OpsRequest

执行以下命令来停止集群。

```bash
kubectl apply -f - <<EOF
apiVersion: apps.kubeblocks.io/v1alpha1
kind: OpsRequest
metadata:
  name: pg-cluster
  generateName: stop-
spec:
  # cluster ref
  clusterRef: pg-cluster
  type: Stop
EOF
```

### 选项 3. 更改 YAML 文件

将副本数设置为 0，删除 Pod。

```yaml
apiVersion: apps.kubeblocks.io/v1alpha1
kind: Cluster
metadata:
    name: pg-cluster
spec:
  clusterDefinitionRef: postgresql
  clusterVersionRef: postgresql-14.7.0
  terminationPolicy: WipeOut
  componentSpecs:
  - name: postgresql
    componentDefRef: postgresql
    monitor: false  
    replicas: 0
    volumeClaimTemplates:
    - name: data
      spec:
        storageClassName: standard
        accessModes:
          - ReadWriteOnce
        resources:
          requests:
            storage: 1Gi
```

## 启动集群
  
### 选项 1. （推荐）使用 kbcli

配置集群名称，并执行以下命令来启动该集群。

```bash
kbcli cluster start <name>
```

***示例***

```bash
kbcli cluster start pg-cluster
```

### 选项 2. 创建 OpsRequest

执行以下命令，启动集群。

```bash
kubectl apply -f - <<EOF
apiVersion: apps.kubeblocks.io/v1alpha1
kind: OpsRequest
metadata:
  name: pg-cluster
  generateName: start-
spec:
  # cluster ref
  clusterRef: pg-cluster
  type: Start
EOF 
```

### 选项 3. 更改 YAML 文件

将副本数改为原始数量，重新启动该集群。

```yaml
apiVersion: apps.kubeblocks.io/v1alpha1
kind: Cluster
metadata:
    name: pg-cluster
spec:
  clusterDefinitionRef: postgresql
  clusterVersionRef: postgresql-14.7.0
  terminationPolicy: WipeOut
  componentSpecs:
  - name: postgresql
    componentDefRef: postgresql
    monitor: false  
    replicas: 1
    volumeClaimTemplates:
    - name: data
      spec:
        storageClassName: standard
        accessModes:
          - ReadWriteOnce
        resources:
          requests:
            storage: 1Gi
```
