# XXX-controllers

### 说明

XXX-controllers 是我刚开始研究 CRD + Controller 时写的一个例子，写的比较简单。

当时关于 CRD + Controller 的使用做了一个PPT：[K8S中CRD的使用](docs/K8S中CRD的使用.md)。

现在将代码和PPT都分享出来，供感兴趣的朋友学习使用。但由于当时研究的不是很深，所以现在看有些部分其实还是有些问题的。

### How to generate code?

```shell
export GOPATH="/opt/go"

mkdir -p /opt/go/src/iop.inspur.com/XXX-controllers

cd /opt/go/src/iop.inspur.com/XXX-controllers

export GOPROXY=https://goproxy.io

export GO111MODULE=on

./kubebuilder init --domain inspur.com

./kubebuilder create api --group test --version v1alpha1 --kind Xxx

go mod vendor
```

剩下的只需要修改`api/v1alpha1/xxx_types.go`中自定义资源的参数，以及`controller/xxx_controller.go`中的`Reconcile`实现调谐逻辑。

### How to build binary?

```shell
export GOPATH="/opt/go"

cd /opt/go/src/iop.inspur.com/XXX-controllers

CGO_ENABLED=0 GOOS=linux GOARCH=amd64 GO111MODULE=on go build -a -o manager main.go
```

### How to run binary?

```shell
cd /opt/go/src/iop.inspur.com/XXX-controllers

./manager --metrics-addr :9097
```

直接以二进制的形式运行起来，需要指定`--metrics-addr`端口，默认端口为`8080`，与节点上的`apiserver`端口冲突。

### How to use?

##### 1. 在k8s集群中创建CRD资源

```yaml
apiVersion: apiextensions.k8s.io/v1beta1
kind: CustomResourceDefinition
metadata:
  name: xxxes.test.inspur.com
spec:
  group: test.inspur.com
  version: v1alpha1
  names:
    kind: Xxx
    plural: xxxes
  scope: Namespaced
```

##### 2. 在k8s集群中运行编译出的二进制manager

```shell
./manager --metrics-addr :9097
```

##### 3. 在k8s集群中创建自定义资源Xxx

```yaml
apiVersion: test.inspur.com/v1alpha1
kind: Xxx
metadata:
  name: xxx-sample
spec:
  gitUrl: https://github.com/kelseyhightower/helloworld.git
  clonePath: /root/go/src/helloworld
  buildCommand: go build
  binaryName: helloworld
  jobName: build-helloworld
```

当`manager`程序监听到创建出新资源后，会启动一个名为`build-helloworld`的`job`来执行以下命令：

```shell
git clone https://github.com/kelseyhightower/helloworld.git /root/go/src/helloworld
cd /root/go/src/helloworld
go build
cp helloworld /opt/
```

从`github`上拉取代码到指定目录中，进入目录构建出`helloworld`二进制，并将该二进制复制到宿主机的`/opt/`目录下，实际效果如下：

```shell
root@inspurtest14-wxhqn85ukg-master-1:/opt/go/src/iop.inspur.com/XXX-controllers# ls -l /opt/ | grep helloworld
root@inspurtest14-wxhqn85ukg-master-1:/opt/go/src/iop.inspur.com/XXX-controllers# kubectl get job -owide | grep helloworld
root@inspurtest14-wxhqn85ukg-master-1:/opt/go/src/iop.inspur.com/XXX-controllers# kubectl get pod -owide | grep helloworld
root@inspurtest14-wxhqn85ukg-master-1:/opt/go/src/iop.inspur.com/XXX-controllers# kubectl apply -f ex.yml 
xxx.test.inspur.com/xxx-sample created
root@inspurtest14-wxhqn85ukg-master-1:/opt/go/src/iop.inspur.com/XXX-controllers# kubectl get job -owide | grep helloworld
build-helloworld   0/1           8s         12s   golang          golang:1.13                                                                                                                      controller-uid=d8b28253-5848-11ea-afc8-fa163ebb64ae
root@inspurtest14-wxhqn85ukg-master-1:/opt/go/src/iop.inspur.com/XXX-controllers# kubectl get pod -owide | grep helloworld
build-helloworld-8cm4l                          0/1     Completed           0          25s   192.168.202.150   master2   <none>           <none>
root@inspurtest14-wxhqn85ukg-master-1:/opt/go/src/iop.inspur.com/XXX-controllers# ls -l /opt/ | grep helloworld
-rwxr-xr-x 1 root root 7485619 Feb 26 11:34 helloworld
root@inspurtest14-wxhqn85ukg-master-1:/opt/go/src/iop.inspur.com/XXX-controllers# 
```

##### 4. 在k8s集群中删除自定义资源Xxx

当`manager`程序监听到要删除自定义资源Xxx后，会启动一个名为`build-helloworld-delete`的`job`将之前构建出的`helloworld`二进制从宿主机目录`/opt/`下删除，实际效果如下：

```shell
root@inspurtest14-wxhqn85ukg-master-1:/opt/go/src/iop.inspur.com/XXX-controllers# ls -l /opt/ | grep helloworld
-rwxr-xr-x 1 root root 7485619 Feb 26 11:34 helloworld
root@inspurtest14-wxhqn85ukg-master-1:/opt/go/src/iop.inspur.com/XXX-controllers# kubectl delete -f ex.yml 
xxx.test.inspur.com "xxx-sample" deleted
root@inspurtest14-wxhqn85ukg-master-1:/opt/go/src/iop.inspur.com/XXX-controllers# kubectl get job -owide | grep helloworld
build-helloworld-delete   0/1           6s         11s   golang          golang:1.13                                                                                                                      controller-uid=899c4752-5849-11ea-afc8-fa163ebb64ae
root@inspurtest14-wxhqn85ukg-master-1:/opt/go/src/iop.inspur.com/XXX-controllers# kubectl get pod -owide | grep helloworld
build-helloworld-delete-kk5zp                   0/1     Completed           0          21s   192.168.202.150   master2   <none>           <none>
root@inspurtest14-wxhqn85ukg-master-1:/opt/go/src/iop.inspur.com/XXX-controllers# ls -l /opt/ | grep helloworld
root@inspurtest14-wxhqn85ukg-master-1:/opt/go/src/iop.inspur.com/XXX-controllers# 
```

##### 5. 在k8s集群中修改自定义资源Xxx

可以修改`Xxx`资源下的`spec`中任意参数，只要有变化程序就会监听到，然后将原先的`job`删除掉，创建一个新的`job`执行新的命令，实际效果如下，命令`cp helloworld`变为`cp helloworld2`：

```shell
root@inspurtest14-wxhqn85ukg-master-1:/opt/go/src/iop.inspur.com/XXX-controllers# kubectl get job -owide | grep helloworld
build-helloworld   1/1           29s        55s   golang          golang:1.13                                                                                                                      controller-uid=34e3c7f7-584a-11ea-afc8-fa163ebb64ae
root@inspurtest14-wxhqn85ukg-master-1:/opt/go/src/iop.inspur.com/XXX-controllers# kubectl get pod -owide | grep helloworld
build-helloworld-j5btd                          0/1     Completed           0          61s   192.168.202.150   master2   <none>           <none>
root@inspurtest14-wxhqn85ukg-master-1:/opt/go/src/iop.inspur.com/XXX-controllers# kubectl get pod build-helloworld-j5btd -oyaml | grep command -A 4
  - command:
    - bash
    - -c
    - git clone https://github.com/kelseyhightower/helloworld.git /root/go/src/helloworld;
      cd /root/go/src/helloworld; go build; cp helloworld /opt/
root@inspurtest14-wxhqn85ukg-master-1:/opt/go/src/iop.inspur.com/XXX-controllers# vi ex.yml 
root@inspurtest14-wxhqn85ukg-master-1:/opt/go/src/iop.inspur.com/XXX-controllers# kubectl apply -f ex.yml 
xxx.test.inspur.com/xxx-sample configured
root@inspurtest14-wxhqn85ukg-master-1:/opt/go/src/iop.inspur.com/XXX-controllers# kubectl get job -owide | grep helloworld
build-helloworld   0/1           25s        25s   golang          golang:1.13                                                                                                                      controller-uid=8b1ffdb7-584a-11ea-afc8-fa163ebb64ae
root@inspurtest14-wxhqn85ukg-master-1:/opt/go/src/iop.inspur.com/XXX-controllers# kubectl get pod -owide | grep helloworld
build-helloworld-j5btd                          0/1     Completed           0          3m4s   192.168.202.150   master2   <none>           <none>
build-helloworld-r7r76                          0/1     Error               0          39s    192.168.202.150   master2   <none>           <none>
root@inspurtest14-wxhqn85ukg-master-1:/opt/go/src/iop.inspur.com/XXX-controllers# kubectl get pod build-helloworld-r7r76 -oyaml | grep command -A 4
  - command:
    - bash
    - -c
    - git clone https://github.com/kelseyhightower/helloworld.git /root/go/src/helloworld;
      cd /root/go/src/helloworld; go build; cp helloworld2 /opt/
```

### How to build image?

Dockerfile:

```Dockerfile
FROM alpine:latest
Add manager /manager
WORKDIR /
ENTRYPOINT ["/manager"]
```

```shell
cd /opt/go/src/iop.inspur.com/XXX-controllers

CGO_ENABLED=0 GOOS=linux GOARCH=amd64 GO111MODULE=on go build -a -o manager main.go

docker build -t test.io/controller .
```

### How to deploy in k8s cluster by pod?

##### 1. config/crd/crd.yaml

`kubectl apply -f config/crd/crd.yaml`

```yaml
apiVersion: apiextensions.k8s.io/v1beta1
kind: CustomResourceDefinition
metadata:
  name: xxxes.test.inspur.com
spec:
  group: test.inspur.com
  version: v1alpha1
  names:
    kind: Xxx
    plural: xxxes
  scope: Namespaced
```

这个`crd`资源要先于`controller`程序创建，否则程序会报错。

##### 2. config/manager/manager.yaml

`kubectl apply -f config/manager/manager.yaml`

```yaml
apiVersion: v1
kind: Namespace
metadata:
  labels:
    control-plane: controller-manager
  name: system
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: controller-manager
  namespace: system
  labels:
    control-plane: controller-manager
spec:
  selector:
    matchLabels:
      control-plane: controller-manager
  replicas: 3
  template:
    metadata:
      labels:
        control-plane: controller-manager
    spec:
      containers:
      - command:
        - /manager
        args:
        - --enable-leader-election
        image: test.io/controller:latest
        name: manager
        resources:
          limits:
            cpu: 100m
            memory: 30Mi
          requests:
            cpu: 100m
            memory: 20Mi
      terminationGracePeriodSeconds: 10
```

创建了`system`命名空间，并在其下面创建`deployment`用于运行`controller`。

**注意:**
在`deployment`中的启动命令增加了参数`--enable-leader-election`，这样做的好处是可以设置多个副本同时运行程序，且保证同一时刻只会有一个起作用，类似于k8s管理组件中的`kube-controller-manager`。



```shell
root@inspurtest14-wxhqn85ukg-master-1:/opt/go/src/iop.inspur.com/XXX-controllers# kubectl get pod -nsystem -owide
NAME                                  READY   STATUS    RESTARTS   AGE     IP               NODE      NOMINATED NODE   READINESS GATES
controller-manager-598bb7d786-ddp56   1/1     Running   0          39s     100.101.208.10   master2   <none>           <none>
controller-manager-598bb7d786-nq5gl   1/1     Running   0          5m43s   100.101.208.9    master2   <none>           <none>
controller-manager-598bb7d786-sn67j   1/1     Running   0          36s     100.101.208.11   master2   <none>           <none>
root@inspurtest14-wxhqn85ukg-master-1:/opt/go/src/iop.inspur.com/XXX-controllers# kubectl logs controller-manager-598bb7d786-ddp56   -nsystem
2020-02-26T07:21:54.050Z	INFO	controller-runtime.metrics	metrics server is starting to listen	{"addr": ":8080"}
2020-02-26T07:21:54.053Z	INFO	setup	starting manager
2020-02-26T07:21:54.056Z	INFO	controller-runtime.manager	starting metrics server	{"path": "/metrics"}
root@inspurtest14-wxhqn85ukg-master-1:/opt/go/src/iop.inspur.com/XXX-controllers# 
root@inspurtest14-wxhqn85ukg-master-1:/opt/go/src/iop.inspur.com/XXX-controllers# kubectl logs controller-manager-598bb7d786-nq5gl -nsystem
2020-02-26T07:16:43.000Z	INFO	controller-runtime.metrics	metrics server is starting to listen	{"addr": ":8080"}
2020-02-26T07:16:43.001Z	INFO	setup	starting manager
2020-02-26T07:16:43.038Z	INFO	controller-runtime.manager	starting metrics server	{"path": "/metrics"}
2020-02-26T07:16:59.121Z	INFO	controller-runtime.controller	Starting EventSource	{"controller": "xxx", "source": "kind source: /, Kind="}
2020-02-26T07:16:59.121Z	DEBUG	controller-runtime.manager.events	Normal	{"object": {"kind":"ConfigMap","namespace":"system","name":"controller-leader-election-helper","uid":"6f409cf5-5865-11ea-85fe-fa163e59946d","apiVersion":"v1","resourceVersion":"32805493"}, "reason": "LeaderElection", "message": "controller-manager-598bb7d786-nq5gl_367bdc74-b387-42fa-b71e-75a8e62cfd28 became leader"}
2020-02-26T07:16:59.222Z	INFO	controller-runtime.controller	Starting Controller	{"controller": "xxx"}
2020-02-26T07:16:59.322Z	INFO	controller-runtime.controller	Starting workers	{"controller": "xxx", "worker count": 1}
2020/02/26 07:16:59 Get Xxx [default] [xxx-sample], Spec: [{https://github.com/kelseyhightower/helloworld.git /root/go/src/helloworld go build helloworld build-helloworld}], Finalizers: [[job.finalizers.test.inspur.com]] 
2020/02/26 07:16:59 Xxx [default] [xxx-sample]'s ObjectMeta.DeletionTimestamp is 0 
2020/02/26 07:16:59 Create Build Job [default] [build-helloworld] success 
2020/02/26 07:16:59 The Build Job [default] [xxx-sample]'s old commands are: [git clone https://github.com/kelseyhightower/helloworld.git /root/go/src/helloworld; cd /root/go/src/helloworld; go build; cp helloworld /opt/]
2020-02-26T07:16:59.839Z	DEBUG	controller-runtime.controller	Successfully Reconciled	{"controller": "xxx", "request": "default/xxx-sample"}
2020/02/26 07:18:43 Get Xxx [default] [xxx-sample], Spec: [{https://github.com/kelseyhightower/helloworld.git /root/go/src/helloworld go build helloworld build-helloworld}], Finalizers: [[job.finalizers.test.inspur.com]] 
2020/02/26 07:18:43 Xxx [default] [xxx-sample]'s ObjectMeta.DeletionTimestamp is not 0 and the finalizerName need to handle 
2020/02/26 07:18:43 Create Delete Job [default] [build-helloworld-delete] success 
2020/02/26 07:18:44 Remove Xxx [default] [build-helloworld-delete] finalizers success 
2020-02-26T07:18:44.170Z	DEBUG	controller-runtime.controller	Successfully Reconciled	{"controller": "xxx", "request": "default/xxx-sample"}
2020/02/26 07:18:44 Unable to get Xxx [default] [xxx-sample], Error: [Xxx.test.inspur.com "xxx-sample" not found] 
2020-02-26T07:18:44.170Z	DEBUG	controller-runtime.controller	Successfully Reconciled	{"controller": "xxx", "request": "default/xxx-sample"}
root@inspurtest14-wxhqn85ukg-master-1:/opt/go/src/iop.inspur.com/XXX-controllers# 
root@inspurtest14-wxhqn85ukg-master-1:/opt/go/src/iop.inspur.com/XXX-controllers# kubectl logs controller-manager-598bb7d786-sn67j -nsystem
2020-02-26T07:21:54.143Z	INFO	controller-runtime.metrics	metrics server is starting to listen	{"addr": ":8080"}
2020-02-26T07:21:54.143Z	INFO	setup	starting manager
2020-02-26T07:21:54.143Z	INFO	controller-runtime.manager	starting metrics server	{"path": "/metrics"}
```

##### 3. config/rbac/

`kubectl apply -f config/rbac/`

```yaml
---
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: leader-election-role
  namespace: system
rules:
- apiGroups:
  - ""
  resources:
  - configmaps
  verbs:
  - get
  - list
  - watch
  - create
  - update
  - patch
  - delete
- apiGroups:
  - ""
  resources:
  - configmaps/status
  verbs:
  - get
  - update
  - patch
- apiGroups:
  - ""
  resources:
  - events
  verbs:
  - create

---
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: leader-election-rolebinding
  namespace: system
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: Role
  name: leader-election-role
subjects:
- kind: ServiceAccount
  name: default
  namespace: system

---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: manager-rolebinding
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: manager-role
subjects:
- kind: ServiceAccount
  name: default
  namespace: system

---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: xxx-editor-role
rules:
- apiGroups:
  - test.inspur.com
  resources:
  - xxxes
  verbs:
  - create
  - delete
  - get
  - list
  - patch
  - update
  - watch
- apiGroups:
  - test.inspur.com
  resources:
  - xxxes/status
  verbs:
  - get
  - patch
  - update

---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: xxx-viewer-role
rules:
- apiGroups:
  - test.inspur.com
  resources:
  - xxxes
  verbs:
  - get
  - list
  - watch
- apiGroups:
  - test.inspur.com
  resources:
  - xxxes/status
  verbs:
  - get

---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: manager-role
rules:
- apiGroups:
  - "test.inspur.com"
  resources:
  - xxxes
  verbs:
  - create
  - delete
  - get
  - list
  - patch
  - update
  - watch
- apiGroups:
  - batch
  resources:
  - jobs
  verbs:
  - create
  - delete
  - get
  - list
  - patch
  - update
  - watch
```

`rbac`当前目录中内容与自动生成时略有不同，首先删除了部分无用的文件,只保留了必要部分；另外`leader-election-rolebinding`和`leader-election-role`原先是没写命名空间的，这里统一改为`system`命名空间；最重要的是`manager-role`，这个文件默认没有生成，需要根据程序中实际需要操控的权限来具体编写，比如在我的程序里需要访问`Xxx`和`Job`，那我就需要将二者的权限添加进去。

##### 3. config/samples/test_v1alpha1_xxx.yaml

到上面为止，已经可以将`controller`以`pod`的形式运行起来了，所有功能都可以正常运行起来了，可以通过下面的这个例子进行验证。

`kubectl apply -f config/samples/test_v1alpha1_xxx.yaml`

```yaml
apiVersion: test.inspur.com/v1alpha1
kind: Xxx
metadata:
  name: xxx-sample
spec:
  gitUrl: https://github.com/kelseyhightower/helloworld.git
  clonePath: /root/go/src/helloworld
  buildCommand: go build
  binaryName: helloworld
  jobName: build-helloworld
```

