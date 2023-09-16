## Jenkins / ArgoCD / Argo Rollouts for Kuberentes

### 1：`Jenkins`

```shell
# 提到基于 kubernetes 的 CI/CD，可以使用的工具那是太多了，比如：Jenkins，Gitlab CI，Tekton，Drone等工具，这里我们将熟悉使用 Jenkins 如何来做基于 Kubernetes 的 CI/CD 工具
```

#### 1.1：`Jenkins` 安装与测试

```shell
# 既然要基于 Kubernetes 来做 CI/CD，我们这里最好还是将 Jenkins 部署到 Kubernetes 中，安装的方法有很多，我们这里可以使用手动的方式，这样可以让我们了解更多的细节
```

```yaml
# persistentvolume.yaml
apiVersion: v1
kind: PersistentVolume
metadata:
  name: jenkins-pv
  labels:
    app: jenkins
spec:
  accessModes: ["ReadWriteOnce"]
  capacity:
    storage: 5Gi
  storageClassName: local-storage
  local:
    path: /data/jenkins
  persistentVolumeReclaimPolicy: Retain
  nodeAffinity:
    required:
      nodeSelectorTerms:
      - matchExpressions:
        - key: kubernetes.io/hostname
          operator: In
          values:
          - k-m-1
---
# namespace.yaml
apiVersion: v1
kind: Namespace
metadata:
  name: kube-ops
---
# persistentvolumeclaim.yaml
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: jenkins-pvc
  namespace: kube-ops
spec:
  storageClassName: local-storage
  accessModes: ["ReadWriteOnce"]
  resources:
    requests:
      storage: 5Gi
---
# serviceaccount.yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  name: jenkins
  namespace: kube-ops
---
# clusterrole.yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: jenkins
rules:
- apiGroups: ["extensions", "apps"]
  resources: ["deployments", "ingresses"]
  verbs: ["create", "delete", "get", "list", "watch", "patch", "update"]
- apiGroups: [""]
  resources: ["services"]
  verbs: ["create", "delete", "get", "list", "watch", "patch", "update"]
- apiGroups: [""]
  resources: ["pods"]
  verbs: ["create", "delete", "get", "list", "watch", "patch", "update"]
- apiGroups: [""]
  resources: ["pods/exec"]
  verbs: ["create", "delete", "get", "list", "watch", "patch", "update"]
- apiGroups: [""]
  resources: ["pods/log", "events"]
  verbs: ["get", "list", "watch"]
- apiGroups: [""]
  resources: ["secrets"]
  verbs: ["get"]
---
# clusterrolebinding.yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: jenkins
  namespace: kube-ops
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: jenkins
subjects:
- kind: ServiceAccount
  name: jenkins
  namespace: kube-ops
---
# deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: jenkins
  namespace: kube-ops
spec:
  replicas: 1
  selector:
    matchLabels:
      app: jenkins
  template:
    metadata:
      labels:
        app: jenkins
    spec:
      serviceAccount: jenkins
      initContainers:
      - name: fix-permissions
        image: busybox:latest
        command: ["sh", "-c", "chown -R 1000:1000 /var/jenkins_home"]
        securityContext:
          privileged: true
        volumeMounts:
        - name: jenkinshome
          mountPath: /var/jenkins_home
      containers:
      - name: jenkins
        image: dockerproxy.com/jenkins/jenkins:lts-jdk11
        imagePullPolicy: IfNotPresent
        env:
        - name: JAVA_OPTS
          value: -Dhudson.model.DownloadService.noSignatureCheck=true -Dhudson.security.csrf.GlobalCrumbIssuerConfiguration.DISABLE_CSRF_PROTECTION=true
        ports:
        - name: web
          protocol: TCP
          containerPort: 8080
        - name: agent
          protocol: TCP
          containerPort: 50000
        readinessProbe:
          httpGet:
            path: /login
            port: 8080
          initialDelaySeconds: 60
          timeoutSeconds: 5
          failureThreshold: 12
        volumeMounts:
        - name: jenkinshome
          mountPath: /var/jenkins_home
      volumes:
      - name: jenkinshome
        persistentVolumeClaim:
          claimName: jenkins-pvc
---
# service.yaml
apiVersion: v1
kind: Service
metadata:
  name: jenkins
  namespace: kube-ops
  labels:
    app: jenkins
spec:
  type: ClusterIP
  selector:
    app: jenkins
  ports:
  - name: web
    port: 8080
    targetPort: web
  - name: agent
    port: 50000
    targetPort: agent
---
# ingress.yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: jenkins
  namespace: kube-ops
  labels:
    app: jenkins
spec:
  ingressClassName: nginx
  rules:
  - host: jenkins.devops-engineer.com
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: jenkins
            port:
              name: web
```

```shell
# 部署如上资源
[root@k-m-1 jenkins]# kubectl get pod,svc,ingress -n kube-ops 
NAME                         READY   STATUS    RESTARTS   AGE
pod/jenkins-b5dc5c57-q5xmf   1/1     Running   0          15m

NAME              TYPE        CLUSTER-IP    EXTERNAL-IP   PORT(S)              AGE
service/jenkins   ClusterIP   10.96.3.147   <none>        8080/TCP,50000/TCP   24m

NAME                                CLASS   HOSTS                         ADDRESS     PORTS   AGE
ingress.networking.k8s.io/jenkins   nginx   jenkins.devops-engineer.com   10.0.0.11   80      24m

# 这里使用的是 jenkins/jenkins:lts-jdk11 镜像，这是 Jenkins 官方的 Docker 镜像，然后也有一些环境变量，当然我们可以根据自己的需求去定制一个自己的镜像，比如，我们可以将一些插件打包到镜像中，这个可以参考：https://github.com/jenkinsci/docker，我们这里使用的是官方默认的镜像，另外还需要注意的是数据的持久化，将容器的 /var/jenkins_home 目录持久化即可，我们这里使用的是 local-pv 的方式

# 由于我们这里使用的镜像内部的用户 UID=1000，所以我们这里挂载出来后会出现权限问题，为了解决这个问题，我们使用了一个 initContainer 来修改挂载的数据目录

# 另外由于 Jenkins 会对 update-center.json 做签名校验安全检查，这里我们需要将其提前关闭，否则更改插件源会失效，我们配置了环境变量来禁止它，其次还有一些限制JVM之类的，也可以查看官方的文档去限制

# 另外我们这里还使用到了一个拥有相关权限的 ServiceAccount，我们这里给 Jenkins 赋予了一些必要的权限，但是如果对 k8s 的 rbac 不太了解的话，测试和开发环境可以直接给一个 cluster-admin 权限，但是这个风险是非常大的，最后就是通过 Ingress 暴露服务就可以了

# 然后我们按照 Ingress 的配置，去访问域名，然后这里需要注意它会让我们去一个路径去拿一个密钥，那么这个密钥如果我们的 Jenkins 做了持久化，我们需要去这这里拿这个密钥
[root@k-m-1 jenkins]# cat /data/jenkins/secrets/initialAdminPassword 
26ff6283993c4c81b29b1eef0b0d0bec

# 然后输入密钥进入下一步，选择插件来安装
```

![jenkins](https://picture.devops-engineer.com.cn/file/d01228414c6aeb6a0ca26.jpg)

![jenkins](https://picture.devops-engineer.com.cn/file/e63fd8276cbec37cf4563.jpg)

![jenkins](https://picture.devops-engineer.com.cn/file/c2b62b94074e415b8adf9.jpg)

![jenkins](https://picture.devops-engineer.com.cn/file/4fff60de33dca5a061b17.jpg)

![jenkins](https://picture.devops-engineer.com.cn/file/2691787d5227c6cd38555.jpg)

![jenkins](https://picture.devops-engineer.com.cn/file/9816e6512885adbc0075d.jpg)

![jenkins](https://picture.devops-engineer.com.cn/file/432c44f51fe5a0bd28e01.jpg)

```shell
# 这样安装完成之后，我们可以去配置插件的加速了，那么这个加速我们可以直接去修改 Jenkins 持久化出来的数据
[root@k-m-1 ~]# cd /data/jenkins/updates/
[root@k-m-1 updates]# cp default.json default.json-bak
# 修改插件的下载地址为国内的地址
[root@k-m-1 updates]# sed -i s#https://updates.jenkins.io/download#https://mirrors.tuna.tsinghua.edu.cn/jenkins#g default.json
# 修改jenkins启动时检测的URL网址，改为国内baidu的地址
[root@k-m-1 updates]# sed -i s#http://www.google.com#https://www.baidu.com#g default.json default.json

# 然后删除 Jenkins 的 Pod 使其重新加载一下配置
[root@k-m-1 updates]# kuwobectl delete pod -n kube-ops jenkins-b5dc5c57-q5xmf 
pod "jenkins-b5dc5c57-q5xmf" deleted
[root@k-m-1 updates]# kubectl get pod -n kube-ops 
NAME                     READY   STATUS    RESTARTS   AGE
jenkins-b5dc5c57-xfs5r   1/1     Running   0          76s

# 然后再次进入 Web 去安装插件
```

![jenkins](https://picture.devops-engineer.com.cn/file/43660dca2ab4d61d68887.jpg)

![jenkins](https://picture.devops-engineer.com.cn/file/7430c714e4f24839fad54.jpg)

![jenkins](https://picture.devops-engineer.com.cn/file/08883f7b0517df5f5171e.jpg)

![jenkins](https://picture.devops-engineer.com.cn/file/c1d4c98fe3a2c6c151b28.jpg)

![jenkins](https://picture.devops-engineer.com.cn/file/1c5a162dd21aae91e8baf.jpg)

![jenkins](https://picture.devops-engineer.com.cn/file/e5c8f3d1c99533316d746.jpg)

```shell
# 这样所需的一些插件就安装好了，然后我们可以去创建我们的第一条测试的流水线
```

![jenkins](https://picture.devops-engineer.com.cn/file/b06c3235bc1d717cb750f.jpg)

![jenkins](https://picture.devops-engineer.com.cn/file/283632cfcc08337dd1dc0.jpg)

![jenkins](https://picture.devops-engineer.com.cn/file/05b7187836fa22c535428.jpg)

![jenkins](https://picture.devops-engineer.com.cn/file/407db25c921852e04dcbe.jpg)

![jenkins](https://picture.devops-engineer.com.cn/file/d2d43be94faab6427467e.jpg)

![jenkins](https://picture.devops-engineer.com.cn/file/6f0b278218715bf4cd99d.jpg)

```shell
# 到这里 Jenkins 的安装与验证就完成了，下面我们将开始了解 Jenkins 的一些使用方法
```

#### 1.2：`Jenkins` 架构

```shell
# 我们都知道持续构建是我们日常工作中必不可少的步骤，目前很多大公司采用的其实都是 Jenkins 或者类 Jenkins 的产品来做符合要求的 CI/CD 流程，然而传统的 Jenkins Slave 一主多从会存在一些痛点：
1：主 Master 发生单点故障时，整个流程则不可用
2：每个 Slave 的配置环境不一样，来完成不同语言的编译打包等操作，但是这些差异化的配置导致管理起来非常不方便，维护起来也比较费劲
3：资源分配不均衡，有的 Slave 要运行 Job 出现排队等待，而有的 Slave 处于空闲状态
4：存在资源浪费，每台 Slave 可能是物理机或者虚拟机，当 Slave 处于空闲状态时，也不会完全释放资源

# 正是因为上面的这些痛点，我们渴望一种更高效更可靠的方式来完成这个 CI/CD 流程，而 Docker 虚拟化容器技术能很好的解决这个痛点，又特别是在 Kubernetes 集群环境下能够更好的解决上面的问题，
```

![jenkins](https://picture.devops-engineer.com.cn/file/21bd41b250259bb146255.png)

```shell
# 从上图可以看出 Jenkins Master 和 Jenkins Slave 以 Pod 形式运行在 Kubernetes 集群的 Node 上，Master 运行在其中的一个节点，并且将配置数据存储到一个 Volume 上，Slave 运行在各个节点上，并且它不是一直处于运行状态，它会按照需求动态的创建和删除

# 这种方式的工作流程大致为：当 Jenkins Master 接收到 Build 请求时，会根据配置的 Label 动态创建一个运行在 Pod 中的 Jenkins Slave 并注册到 Master 上，当运行完 Job 后，这个 Slave 会被注销并且这个 Pod 也会被自动删除，恢复到最初状态

# 那么，使用这种方式给我们带来了带来了哪儿些好处呢？
1：服务高可用：当 Jenkins Master 出现故障时，Kubernetes 会自动创建一个新的 Jenkins Master 容器，并且将 Volume 分配给新创建的容器，保证数据不丢失，从而达到集群服务高可用
2：动态伸缩：合理的使用资源，每次运行 Job 时，会自动创建一个 Jenkins Slave，Job 完成后，Slave 自动注销并删除容器，资源自动释放，而且 Kubernetes 会根据每个资源的使用情况，动态分配 Slave 到空闲的节点上进行创建，降低出现因某节点资源利用率高，还排队等待该节点的情况
3：扩展性强：当 kubernetes 集群上的资源严重不足而导致 Job 排队等待时，可以很容易的添加一个 kubernetes node 到集群中，从而实现扩展，从这几个优点看来，上面的一些问题能够很好的得到解决了

# 
```

#### 1.3：`Agent` 节点

```shell
# 虽然我们上面提到的是动态节点的好处，但是还是有一部分人喜欢采用静态节点的方式，选择静态节点或者动态节点 Jenkins Agent 节点都是可以的，那么接下来，我们开始在集群中为 Jenkins 提供动态和静态Agent节点
```

##### 1.3.1：静态节点

```shell
# 首先在 Jenkins 页面新建一个节点
```

![jenkins-static-agent](https://picture.devops-engineer.com.cn/file/d7d960b02403db90637a7.jpg)

![jenkins-static-agent](https://picture.devops-engineer.com.cn/file/77aaaa22a422b5027fcd2.jpg)

![jenkins-static-agent](https://picture.devops-engineer.com.cn/file/c1d7ca8cac01157f25df7.jpg)

![jenkins-static-agent](https://picture.devops-engineer.com.cn/file/90c4d4579f039deba6299.jpg)

![jenkins-static-agent](https://picture.devops-engineer.com.cn/file/1c5929632daf452c43bbf.jpg)

![jenkins-static-agent](https://picture.devops-engineer.com.cn/file/e47ba472587adf5b6756a.jpg)

```shell
# 到这里我们创建好了一个静态的 Agent，但是也仅仅是创建，还不能用，因为我们还没有创建具体的 Agent 的资源，所以我们看到这里其实就是可以基于各个平台去启动一个 Agent 的程序，然后根据参数去启动它，那么下面是一个 Jenkins 的静态 Agent 的资源文件
```

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: jenkins-agent
  namespace: kube-ops
  labels:
    app: jenkins-agent
spec:
  selector:
    matchLabels:
      app: jenkins-agent
  template:
    metadata:
      labels:
        app: jenkins-agent
    spec:
      containers:
      - name: agent
        image: dockerproxy.com/jenkins/inbound-agent:latest
        imagePullPolicy: IfNotPresent
        securityContext:
          privileged: true
        env:
        - name: JENKINS_URL
          # 如果是使用了 Ingress 的控制器，那么可以将 Ingress 控制器的 service 地址写在 CoreDNS的映射上，这样也可以使用 Jenkins 的域名 + Service 的端口访问到 Jenkins
          value: http://jenkins.kube-ops.svc.cluster.local:8080
        - name: JENKINS_SECRET
          value: 1fa3090f1f020dfef99d4fd4cd905d93900a39a25b4e698da8d0f2cb956a6a38
        - name: JENKINS_AGENT_NAME
          value: static_agent
        - name: JENKINS_AGENT_WORKDIR
          value: /home/jenkins/workspace
```

```shell
# 然后我们只需要部署这个 Agent 让后去 Jenkins Master 查看效果就可以了
[root@k-m-1 jenkins]# kubectl apply -f jenkins-agent.yaml 
deployment.apps/jenkins-agent created

[root@k-m-1 jenkins]# kubectl logs -f -n kube-ops jenkins-agent-7956448fd4-gqs24 
Sep 20, 2023 1:35:58 PM hudson.remoting.jnlp.Main createEngine
INFO: Setting up agent: static_agent
Sep 20, 2023 1:35:58 PM hudson.remoting.Engine startEngine
INFO: Using Remoting version: 3148.v532a_7e715ee3
Sep 20, 2023 1:35:58 PM org.jenkinsci.remoting.engine.WorkDirManager initializeWorkDir
INFO: Using /home/jenkins/workspace/remoting as a remoting work directory
Sep 20, 2023 1:35:58 PM org.jenkinsci.remoting.engine.WorkDirManager setupLogging
INFO: Both error and output logs will be printed to /home/jenkins/workspace/remoting
Sep 20, 2023 1:35:58 PM hudson.remoting.jnlp.Main$CuiListener status
INFO: Locating server among [http://jenkins.kube-ops.svc.cluster.local:8080/]
Sep 20, 2023 1:35:58 PM org.jenkinsci.remoting.engine.JnlpAgentEndpointResolver resolve
INFO: Remoting server accepts the following protocols: [JNLP4-connect, Ping]
Sep 20, 2023 1:35:58 PM hudson.remoting.jnlp.Main$CuiListener status
INFO: Agent discovery successful
  Agent address: jenkins.kube-ops.svc.cluster.local
  Agent port:    50000
  Identity:      a2:3b:6c:12:31:4e:59:30:63:e1:a1:77:41:4c:97:e2
Sep 20, 2023 1:35:58 PM hudson.remoting.jnlp.Main$CuiListener status
INFO: Handshaking
Sep 20, 2023 1:35:58 PM hudson.remoting.jnlp.Main$CuiListener status
INFO: Connecting to jenkins.kube-ops.svc.cluster.local:50000
Sep 20, 2023 1:35:58 PM hudson.remoting.jnlp.Main$CuiListener status
INFO: Trying protocol: JNLP4-connect
Sep 20, 2023 1:35:58 PM org.jenkinsci.remoting.protocol.impl.BIONetworkLayer$Reader run
INFO: Waiting for ProtocolStack to start.
Sep 20, 2023 1:36:04 PM hudson.remoting.jnlp.Main$CuiListener status
INFO: Remote identity confirmed: a2:3b:6c:12:31:4e:59:30:63:e1:a1:77:41:4c:97:e2
Sep 20, 2023 1:36:04 PM hudson.remoting.jnlp.Main$CuiListener status
INFO: Connected  # 看到这条信息证明连接上了
```

![jenkins-static-agent](https://picture.devops-engineer.com.cn/file/1795805b83fe76b53e7fb.jpg)

```shell
# 这样，静态的 Slave 就已经添加好了，修改我们的流水线，使用 node 属性中的 label 标签指定 agent
```

![jenkins-static-agent](https://picture.devops-engineer.com.cn/file/47fb5ae33744befe30387.jpg)

![jenkins-static-agent](https://picture.devops-engineer.com.cn/file/1199d85247c274c469189.jpg)

```shell
# 从这里我们就可以看到，这个静态节点也是跑通的了，Job 已经在静态节点执行成功了，但是至于流水线怎么设计，这里我们不讲，留到后面来讲
```

##### 1.3.2：动态节点

```shell
# 动态节点需要用的的一个插件就是 Kubernetes，这个我们在前面装过了，
```

![jenkins-dynamic-agent](https://picture.devops-engineer.com.cn/file/5f1da834aa0adaa86873c.jpg)

![jenkins-dynamic-agent](https://picture.devops-engineer.com.cn/file/fd43d5ce265939208198a.jpg)

![jenkins-dynamic-agent](https://picture.devops-engineer.com.cn/file/d7cf70b3b19c068ab260b.jpg)

![jenkins-dynamic-agent](https://picture.devops-engineer.com.cn/file/552fc32d7d283fcfc2424.jpg)

![jenkins-dynamic-agent](https://picture.devops-engineer.com.cn/file/767742bf8ca9b402b257a.png)

![jenkins-dynamic-agent](https://picture.devops-engineer.com.cn/file/d0d4815d73d1b60fa0e5c.jpg)

![jenkins-dynamic-agent](https://picture.devops-engineer.com.cn/file/1470165bb5e1ace67c8d3.jpg)

![jenkins-dynamic-agent](https://picture.devops-engineer.com.cn/file/2ca4cd0d01cb790db5f95.jpg)

![jenkins-dynamic-agent](https://picture.devops-engineer.com.cn/file/3142d55061d3d806fa7fe.jpg)

```groovy
pipeline {
    agent {
        kubernetes {
            // 定义动态 agent 的名称，若未定义，则使用 Job Name + 构建次数 + 随机值
            label "dynamic"
            // 指定上面创建的 Cloud 的 NAME
            cloud "kubernetes"
            yaml '''
apiVersion: v1
kind: Pod
spec:
  containers:
  - name: jnlp
    image: jenkins/inbound-agent:latest
'''
        }
    }
    stages {
        stage('Main') {
            steps {
                script {
                    echo "Hello Jenkins!"
                }
            }
        }
    }
}
```

![jenkins-dynamic-agent](https://picture.devops-engineer.com.cn/file/943a636604be151cd2f83.jpg)

```shell
# 这里唯一需要注意的一点就是，container 的名字必须叫做 jnlp，否则 agent 会被意外关闭（这个设计我是不懂的），然后我们来看看构建过程中，K8S 发生了什么事情
[root@k-m-1 ~]# kubectl get pod -n kube-ops -w
NAME                             READY   STATUS    RESTARTS      AGE
jenkins-agent-7956448fd4-gqs24   1/1     Running   0             25h
jenkins-b5dc5c57-xfs5r           1/1     Running   1 (30h ago)   30h
dynamic-agent-7-htqqr-7g390-hjb1z   0/1     Pending   0             0s
dynamic-agent-7-htqqr-7g390-hjb1z   0/1     Pending   0             0s
dynamic-agent-7-htqqr-7g390-hjb1z   0/1     ContainerCreating   0             0s
dynamic-agent-7-htqqr-7g390-hjb1z   0/1     ContainerCreating   0             0s
dynamic-agent-7-htqqr-7g390-hjb1z   1/1     Running             0             5s
dynamic-agent-7-htqqr-7g390-hjb1z   1/1     Terminating         0             21s
dynamic-agent-7-htqqr-7g390-hjb1z   1/1     Terminating         0             21s
dynamic-agent-7-htqqr-7g390-hjb1z   0/1     Terminating         0             22s
dynamic-agent-7-htqqr-7g390-hjb1z   0/1     Terminating         0             22s
dynamic-agent-7-htqqr-7g390-hjb1z   0/1     Terminating         0             22s

# 可以看到，从构建到删除，这个过程还是比较符合我们的预期的，那么这里需要提到的是这个名称，如果你想自定义这个动态节点的名称需要在 Pipeline 添加一个 label，别问我为啥不搞 Gitlab，因为太重了
```

### 2：`Gitea`

```shell
# Gitea 是一个类 Gitlab 的 Web 端 Git 管理平台，它比起 Gitlab 比较轻量化，并且基本的功能也都有，所以我选择 Gitea
```

#### 1.1：`Gitea`的安装与测试

```shell
# 因为是基于 Kubernetes 部署的，所以我们还是以 Helm 为主，但是这里我们并不用官方的 Helm，官方的 Helm 需要太多的配置，而我们使用的是 Bitnami 的 Charts
[root@k-m-1 ~]# helm repo add bitnami https://charts.bitnami.com/bitnami
[root@k-m-1 ~]# helm search repo bitnami
# 然后我们来自定义一下 values.yaml
```

```yaml
global:
  storageClass: "nfs-csi"

image:
  registry: docker.io
  repository: bitnami/gitea
  tag: 1.20.4-debian-11-r0

replicaCount: 1

adminUsername: gitlayzer

adminPassword: gitlayzer

adminEmail: gitlayzer@gmail.com

appName: Gitea

persistence:
  enabled: true
  storageClass: "nfs-csi"
  accessModes:
  - ReadWriteOnce
  size: 5Gi

ingress:
  enabled: true
  ingressClassName: nginx
  hostname: "git.devops-engineer.com.cn"
  path: /
  
resources:
  limits:
    cpu: 500m
    memory: 1Gi
  requests:
    cpu: 500m
    memory: 1Gi

service:
  type: ClusterIP
  ports:
    http: 80
    ssh: 22

postgresql:
  enabled: true
  global:
    postgresql:
      postgresqlDatabase: gitea
      postgresqlUsername: gitea
      postgresqlPassword: gitea
      servicePort: 5432
  persistence:
    size: 5Gi
```

```shell
# 这里是我自定义的，如果有需要可以自行更改，或者更深入的定制，下面就是安装了
[root@k-m-1 ~]# helm upgrade --install gitea ./gitea -f gitea/pre-values.yaml -n kube-ops 
Release "gitea" does not exist. Installing it now.
NAME: gitea
LAST DEPLOYED: Sat Sep 23 19:29:15 2023
NAMESPACE: kube-ops
STATUS: deployed
REVISION: 1
TEST SUITE: None
NOTES:
CHART NAME: gitea
CHART VERSION: 0.4.2
APP VERSION: 1.20.4

** Please be patient while the chart is being deployed **

1. Get the Gitea URL:

  You should be able to access your new Gitea installation through

  http://git.devops-engineer.com.cn/

2. Get your Gitea login credentials by running:

  echo Username: gitlayzer
  echo Password: $(kubectl get secret --namespace kube-ops gitea -o jsonpath="{.data.admin-password}" | base64 -d)
  
# 检查部署
[root@k-m-1 ~]# kubectl get pod -n kube-ops -w
NAME                             READY   STATUS    RESTARTS   AGE
gitea-5f778c94c9-c8zcz           1/1     Running   0          42s
gitea-postgresql-0               1/1     Running   0          42s
jenkins-595bbfc786-5twtc         1/1     Running   0          118m
jenkins-agent-7956448fd4-zmkt9   1/1     Running   0          21h
```

![gitea](https://picture.devops-engineer.com.cn/file/c64ac367f7c41fc66aadd.jpg)

![gitea](https://picture.devops-engineer.com.cn/file/d609b1e07de6dc03a13c4.jpg)

```shell
# OK，这样就部署好了 Gitea 了，然后我们就可以去创建仓库，然后使用了，但是需要注意的是，如果你想使用 SSH 去拉取代码，恰好又想用域名，那个这个时候你就需要了解一下 Ingress 怎么去暴露 TCP 协议了，这个在 Ingress 的使用里面我讲过，在前面的文章中

URL：https://blog.devops-engineer.com.cn/article/ingress_use.html

# 如果使用 WebHook，那么我们需要定制一下配置文件，就是部署好的 Gitea，需要修改配置文件，开启某些功能

/opt/bitnami/gitea/custom/conf
```

```ini
APP_NAME = Gitea
RUN_USER = gitea
RUN_MODE = prod
WORK_PATH = /opt/bitnami/gitea

[repository]
ROOT = /opt/bitnami/gitea/data/git/repositories

[repository.local]
LOCAL_COPY_PATH = /opt/bitnami/gitea/tmp/local-repo

[repository.upload]
TEMP_PATH = /opt/bitnami/gitea/tmp/uploads

[database]
DB_TYPE = postgres
HOST = gitea-postgresql:5432
NAME = bitnami_gitea
USER = bn_gitea
PASSWD = wYwrb2dWt0
SSL_MODE = disable
SCHEMA = 
PATH = 
LOG_SQL = false

[server]
DOMAIN = localhost
HTTP_PORT = 3000
PROTOCOL = http
ROOT_URL = http://git.devops-engineer.com.cn/
APP_DATA_PATH = /opt/bitnami/gitea/data
DISABLE_SSH = false
START_SSH_SERVER = true
SSH_PORT = 22
SSH_LISTEN_PORT = 2222
SSH_DOMAIN = localhost
BUILTIN_SSH_SERVER_USER = gitea
LFS_START_SERVER = false
OFFLINE_MODE = false

[mailer]
ENABLED = false

[session]
PROVIDER_CONFIG = /opt/bitnami/gitea/data/sessions
PROVIDER = file

[picture]
AVATAR_UPLOAD_PATH = /opt/bitnami/gitea/data/avatars
REPOSITORY_AVATAR_UPLOAD_PATH = /opt/bitnami/gitea/data/repo-avatars

[attachment]
PATH = /opt/bitnami/gitea/data/attachments

[log]
ROOT_PATH = /opt/bitnami/gitea/tmp/log
MODE = console
LEVEL = info

[security]
PASSWORD_HASH_ALGO = pbkdf2
REVERSE_PROXY_LIMIT = 1
REVERSE_PROXY_TRUSTED_PROXIES = *
INSTALL_LOCK = true
INTERNAL_TOKEN = eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJuYmYiOjE2OTU0Njg1NzZ9.7cKhYwnEEHCdNnG0-wGd-ZsXFDCO-xEBSaATYE_uHSs

[service]
REGISTER_EMAIL_CONFIRM = false
ENABLE_NOTIFY_MAIL = false
DISABLE_REGISTRATION = false
ALLOW_ONLY_EXTERNAL_REGISTRATION = false
ENABLE_CAPTCHA = false
REQUIRE_SIGNIN_VIEW = false
DEFAULT_KEEP_EMAIL_PRIVATE = false
DEFAULT_ALLOW_CREATE_ORGANIZATION = false
DEFAULT_ENABLE_TIMETRACKING = false
NO_REPLY_ADDRESS = 

[openid]
ENABLE_OPENID_SIGNIN = false
ENABLE_OPENID_SIGNUP = false

[cron.update_checker]
ENABLED = false

[repository.pull-request]
DEFAULT_MERGE_STYLE = merge

[repository.signing]
DEFAULT_TRUST_MODEL = committer

[oauth2]
JWT_SECRET = oMaZIDBGkXuotVAngWiIYUe_8FmVU-3qP9jJ7JTTjgQ

[webhook]
ALLOWED_HOST_LIST = 0.0.0.0/8

[actions]
ENABLED = true
```

```shell
# Oh! Yeash! 因为有持久化的关系，我们可以直接去修改持久化的数据，然后删除一下 Gitea 然后让它重新加载一下新配置，记住 Rollout Restart 是不行的奥
```

#### 1.2：`Gitea + Jenkins` CI / CD

```shell
# 主要操作如下
```

![gitea](https://picture.devops-engineer.com.cn/file/5b1bfa5868ebe8a14de25.jpg)

![gitea](https://picture.devops-engineer.com.cn/file/277b45a246fb15776c41e.jpg)

![gitea](https://picture.devops-engineer.com.cn/file/5b29614d197b377cb5095.jpg)

![jenkins](https://picture.devops-engineer.com.cn/file/6ff4e7781ee3d070aa4fe.jpg)

![jenkins](https://picture.devops-engineer.com.cn/file/1fe830af700260b34f8a5.jpg)

![gitea](https://picture.devops-engineer.com.cn/file/b0ceb1655eed58d66d597.jpg)

![gitea](https://picture.devops-engineer.com.cn/file/0903f44b79ec6941bb1fc.jpg)

![jenkins](https://picture.devops-engineer.com.cn/file/cde160905e5d7830f8e52.jpg)

```shell
# 其实无论需不需要 Jenkins，单独的 Gitlab，Gitea 它们都是可以实现 CI/CD 的，因为 Gitlab 本身自带一个 CI/CD 功能，而 Gitea 效仿 Github 有一个 Action 的功能。也可以实现 CI/CD，但是我们主要还是讲结合 Jenkins + Gitea 实现的 CI/CD
```

### 3：`Jenkins Pipeline`

```shell
# 要实现在 Jenkins 中的构建工作，可以有多种方式，我们这里可以采用比较常见的方式，也就是 Pipeline，简单来说，就是一套运行在 Jenkins 上的工作流框架，将原来独立运行于单个或者多个节点的任务连接起来，实现单个任务难以完成的复杂流程编排和可视化的工作

# 那么 Jenkins Pipeline 有几个核心的概念：
1：Node：节点，一个 Node 节点就是一个 Jenkins 节点，Master 或者 Agent，是执行 Step 的具体运行环境，比如我们前面运行的 Jenkins Slave 就是一个 Node 节点
2：Stage：阶段，一个 Pipeline 可以划分为若干个 Stage，单个 Stage 代表一组操作，如 Build，Test，Deploy，Stage是一个逻辑分组的概念，可以跨越多个 Node
3：Step：步骤，Step 是最基本的操作单元，可以是打印一句话，也可以是构建一个 Docker 镜像，由各类 Jenkins 插件提供，比如命令 sh "make"，就相当于我们平时 shell 终端执行一样

# 那么我们如何构建 Jenkins Pipeline 呢？
1：Pipeline 脚本是由 Groovy 语言实现的，但是我们没必要单独去学习 Groovy，当然如果会的话更好
2：Pipeline 支持两种语法，Declarative（声明式）和 Scripted Pipeline（脚本式）语法
3：Pipeline 也有两种创建方法，可以直接在 Jenkins 的 Web UI 中输入脚本，也可以通过创建一个 JenkinsFile 脚本文件放入到 Git 中托管
4：一般我们都推荐在 Jenkins 中直接从 Git 仓库直接载入 JenkinsFile Pipeline 这种方法

# 那么我们可以快速创建一个比较简单的 Pipeline，直接在 Jenkins 的 Web UI 上插入 Groovy 代码就行
1：新建任务：在 Web UI 中点击 `新建任务` -> `输入任务名称` -> `Fist-Pipeline` -> `选择流水线` -> `点击确定`
2：配置：在最下面的 Pipeline 区域输入如下 Script 脚本，然后保存
```

```groovy
node {
    stage ("Clone") {
        echo "This is Clone Code"
    }
    stage ("Test") {
        echo "This is Test Code"
    }
    stage ("Build") {
        echo "This is Build Image"
    }
    stage ("Deploy") {
        echo "This is Deploy Application"
    }
}
```

```shell
3：点击立即构建，然后它会调度到我们的 static_agent 上去执行这个 Job，因为我们在 Master 上设置的创建任务数量为 0，所以它就不能去执行任务了，这个时候就交给了我们的静态 Agent 去执行这个 Job 了，如果说大家对 Pipeline 语法不是特别了解，那么在我们输入 Pipeline Script 的地方下面有一个流水线语法，大家可以点进去按照自己的需求去配置流水线，然后它会帮助我们去生成这个语法，这样使用起来会更简单
```

#### 3.1：`Jenkins Slave` 中创建任务

```shell
# 上面我们创建了一个简单的 Pipeline 任务，这个任务是跑在静态的 Pod 上的，那么它如何跑在动态的 Agent 上呢？这个其实前面我们已经做过了，可以用 Pod 的模板创建 Agent 节点，它可以通过用户界面进行配置，也可以使用 podTemplate 步骤在 Pipeline 中进行配置，无论那儿种方式，它都可以提供以下字段访问
```

```groovy
podTemplate(cloud: "kubernetes") {
    node(POD_LABEL) {
        stage ("Clone") {
            echo "This is Clone Code"
        }
        stage ("Test") {
            echo "This is Test Code"
        }
        stage ("Build") {
            echo "This is Build Image"
        }
        stage ("Deploy") {
            echo "This is Deploy Application"
        }
    }
}
```

```shell
# 然后点击执行，之后你就会在终端看到这样的操作
[root@k-m-1 ~]# kubectl get pod -n kube-ops -w
NAME                                READY   STATUS              RESTARTS   AGE
fist-pipeline-2-n2l6d-xs93h-dtczg   0/1     ContainerCreating   0          0s
gitea-5f778c94c9-ghnbl              1/1     Running             0          25h
gitea-postgresql-0                  1/1     Running             0          43h
jenkins-8657594859-bv572            1/1     Running             0          42h
jenkins-agent-7956448fd4-zmkt9      1/1     Running             0          2d16h
fist-pipeline-2-n2l6d-xs93h-dtczg   0/1     ContainerCreating   0          0s
fist-pipeline-2-n2l6d-xs93h-dtczg   1/1     Running             0          32s
fist-pipeline-2-n2l6d-xs93h-dtczg   1/1     Terminating         0          46s
fist-pipeline-2-n2l6d-xs93h-dtczg   1/1     Terminating         0          47s
fist-pipeline-2-n2l6d-xs93h-dtczg   0/1     Terminating         0          48s
fist-pipeline-2-n2l6d-xs93h-dtczg   0/1     Terminating         0          48s
fist-pipeline-2-n2l6d-xs93h-dtczg   0/1     Terminating         0          48s

# 从开始创建到结束删除 Pod 的整个过程，那么这个就是动态的 Agent 执行任务的过程了
```

#### 3.2：`Kubernetes` 应用部署

```shell
# 我们的整体步骤如下：
1：编写代码
2：测试
3：编写 DockerFile
4：构建打包 Docker 镜像
5：推送镜像到仓库
6：编写 Kubernetes Yaml 文件
7：更改 Yaml 文件中的 Docker 镜像 Tag
8：使用 kubectl 工具部署应用

# 我们一般部署 K8S 应用的流程就是如上，然后我们需要将这些流程引入到 Jenkins 中来，然后让 Jenkins 自动帮我们完成这些动作，从测试到更新YAML文件属于CI，后面的部署就属于CD的流程了，按照我们的示例，我们要来编写一个 Pipeline 的脚本了，那么如何编写呢？
```

```groovy
podTemplate(cloud: "kubernetes") {
    node(POD_LABEL) {
        stage ("Clone") {
            echo "This is Clone Code"
        }
        stage ("Test") {
            echo "This is Test Code"
        }
        stage ("Build") {
            echo "This is Build Image"
        }
        stage ("Push") {
            echo "This is Push Image"
        }
        stage ("YAML") {
            echo "This is Change YAML"
        }
        stage ("Deploy") {
            echo "This is Deploy Application"
        }
    }
}
```

```shell
# 然后我们创建一个流水线作业，直接使用上面的脚本来构建，同样可以得到正确的结果

# 然后呢，这里我写了一个简单的代码，主要是一个 Web 服务，提供了两个 GET 方法
```

```go
package main

import (
	"github.com/gin-gonic/gin"
	"log"
	"net/http"
	"os"
)

var (
	ServiceMode = os.Getenv("GIN_MODE")
)

func init() {
	if ServiceMode == "" {
		ServiceMode = gin.DebugMode
	}
}

func Handler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"msg": "Hello This is cdk-web-service",
	})
}

func Health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"code": http.StatusOK,
		"msg":  "true",
	})
}

func main() {
	gin.SetMode(ServiceMode)

	r := gin.Default()

	r.GET("/", Handler)
	r.GET("/health", Health)

	if err := r.Run(":8080"); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
```

```dockerfile
FROM golang:1.21-alpine as builder  
WORKDIR /app  
ENV GOPROXY=https://goproxy.cn  
COPY ./go.mod /app
COPY ./go.sum /app
COPY ./main.go /app
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o cdk-web-service  

FROM busybox as runner
COPY --from=builder /app/cdk-web-service /app
ENTRYPOINT ["/app"]
```

![ci/cd](https://picture.devops-engineer.com.cn/file/dc39b1787035f5d4509fb.jpg)

![ci/cd](https://picture.devops-engineer.com.cn/file/6889deffbacd5c092492d.jpg)

![ci/cd](https://picture.devops-engineer.com.cn/file/95c608e13de71d5b35e8a.jpg)

![ci/cd](https://picture.devops-engineer.com.cn/file/a9e386e2841304a9a703a.jpg)

![ci/cd](https://picture.devops-engineer.com.cn/file/225b60c5ae97dd2067a97.jpg)

```shell
# 到这里，自动触发就配置好了，然后我们现在需要做的就是如何去配置这个 Jenkinsfile 了，不过我们有一点可以注意，我们可以将 Jenkinsfile 丢在项目的仓库中，然后直接拿来用，这样我们的 Jenkinsfile 既可以做版本控制，也可以脱离 Jenkins 存储了
```

![ci/cd](https://picture.devops-engineer.com.cn/file/2f44ffe24ffa0179e4144.jpg)

![ci/cd](https://picture.devops-engineer.com.cn/file/5625b0397ef197b3de0e7.jpg)

```shell
# 那么这样做之后，我们就可以直接将 Jenkinsfile 丢到代码仓库里面去了
```

![ci/cd](https://picture.devops-engineer.com.cn/file/b6f5bea9e9ff4190c6568.jpg)

```shell
# 然后我们依旧可以去触发一下流水线，测试一下这样配置是否可以正常运行流水线，然后我们的一个具体的流水线就是如下这样的
```

```groovy
podTemplate(cloud: "kubernetes", containers: [
    containerTemplate(name: 'golang', image: 'golang:1.21.1-alpine3.18', command: 'cat', ttyEnabled: true),
    containerTemplate(name: 'docker', image: 'docker:latest', command: 'cat', ttyEnabled: true),
    containerTemplate(name: 'kubectl', image: 'bitnami/kubectl', command: 'cat', ttyEnabled: true)
], serviceAccount: 'jenkins', volumes: [
    hostPathVolume(mountPath: '/home/jenkins/.kube', hostPath: '/root/.kube')
], envVars: [
    envVar(key: 'DOCKER_HOST', value: 'tcp://docker-dind:2375')
]) {
    node(POD_LABEL) {
        def Repo = checkout scm
        def GitCommit = Repo.GIT_COMMIT
        def GitBranch = Repo.GIT_BRANCH
        
        stage('单元测试') {
            echo "测试阶段"
        }
        
        stage('代码编译打包') {
            container('golang') {
                echo "代码编译打包阶段"
            }
        }
        
        stage('构建镜像') {
            container('docker') {
                echo "构建镜像阶段"
            }
        }
        
        stage('执行部署') {
            container('kubectl') {
                echo "查看 Pod 列表"
                sh "kubectl get pod"
            }
        }
    }
}
```

```shell
# 那么这个就是一个大概的 Jenkinsfile了，不过需要注意的是，因为现在版本的 K8S 基本上都不再使用 Docker 作为 Runtime 了，所以我们需要在 K8S 中单独启动一个 Docker Daemon 作为打包使用，所所以我们要部署一下它
```

```yaml
apiVersion: v1
kind: PersistentVolume
metadata:
  name: dind-pv
spec:
  capacity:
    storage: 5Gi
  volumeMode: Filesystem
  accessModes:
  - ReadWriteOnce
  persistentVolumeReclaimPolicy: Delete
  storageClassName: local-storage
  local:
    path: /data/docker
  nodeAffinity:
    required:
      nodeSelectorTerms:
      - matchExpressions:
        - key: kubernetes.io/hostname
          operator: In
          values:
          - k-m-1
---
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: dind-pvc
  namespace: kube-ops
  labels:
    app: dind
spec:
  accessModes:
  - ReadWriteOnce
  storageClassName: local-storage
  resources:
    requests:
      storage: 5Gi
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: dind
  namespace: kube-ops
  labels:
    app: dind
spec:
  selector:
    matchLabels:
      app: dind
  template:
    metadata:
      labels:
        app: dind
    spec:
      containers:
      - name: dind
        image: docker:dind
        args:
        - --registry-mirror=https://qa57rb9q.mirror.aliyuncs.com  # 指定一个镜像加速地址
        env:
        - name: DOCKER_DRIVER
          value: vfs
        - name: DOCKER_HOST
          value: tcp://0.0.0.0:2375
        - name: DOCKER_TLS_CERTDIR  # 禁用 TLS （最好是不要禁用）
          value: ""
        volumeMounts:
        - name: dind-data  # 持久化 Docker 目录
          mountPath: /var/lib/docker
        ports:
        - name: daemon-port
          containerPort: 2375
        securityContext:
          privileged: true  # 设置特权模式
      volumes:
      - name: dind-data
        persistentVolumeClaim:
          claimName: dind-pvc
---
apiVersion: v1
kind: Service
metadata:
  name: docker-dind
  namespace: kube-ops
  labels:
    app: dind
spec:
  ports:
  - name: daemon-port
    port: 2375
    targetPort: 2375
  selector:
    app: dind
```

```shell
# 然后在 volumes 中通过 hostPathVolume 将集群的 kubeconfig 文件挂载到容器中去，这样我们就可以在容器中访问 kubernetes 了，但是由于我们的构建是在 Slave Pod 中进行的，然而这个 Pod 又不固定在某个节点上，那么我们就必须确保每个节点都必须有 kubeconfig，所以我们选择另一种方式

[root@k-m-1 docker]# kubectl get pod,svc -n kube-ops -l app=dind
NAME                        READY   STATUS    RESTARTS   AGE
pod/dind-8647577b5b-564m6   1/1     Running   0          16s

NAME                  TYPE        CLUSTER-IP    EXTERNAL-IP   PORT(S)    AGE
service/docker-dind   ClusterIP   10.96.3.233   <none>        2375/TCP   102s

# 我们通过将 Kubeconfig 上传到 Jenkins 上，通过 Jenkins 管理它，然后在 Jenkinsfile 中直接读取这个文件，然后拷贝到~/.kube/config 文件中，这样同样也可以正常使用 kubectl 访问集群
```

![ci/cd](https://picture.devops-engineer.com.cn/file/7df904931e08a07e6c701.jpg)

![ci/cd](https://picture.devops-engineer.com.cn/file/3299601753c97eda1b6d3.jpg)

```shell
# 然后更改一下 Jenkinsfile
```

```groovy
podTemplate(cloud: "kubernetes", containers: [
    containerTemplate(name: 'golang', image: 'golang:1.21.1-alpine3.18', command: 'cat', ttyEnabled: true),
    containerTemplate(name: 'docker', image: 'docker:latest', command: 'cat', ttyEnabled: true),
    containerTemplate(name: 'kubectl', image: 'cnych/kubectl', command: 'cat', ttyEnabled: true)
], serviceAccount: 'jenkins', envVars: [
    envVar(key: 'DOCKER_HOST', value: 'tcp://docker-dind:2375')
]) {
    node(POD_LABEL) {
        def Repo = checkout scm
        def GitCommit = Repo.GIT_COMMIT
        def GitBranch = Repo.GIT_BRANCH
        
        stage('单元测试') {
            echo "测试阶段"
        }
        
        stage('代码编译打包') {
            container('golang') {
                echo "代码编译打包阶段"
            }
        }
        
        stage('构建镜像') {
            container('docker') {
                echo "构建镜像阶段"
            }
        }
        
        stage('执行部署') {
            withCredentials([file(credentialsId: 'kubeconfig', variable: 'KUBECONFIG')]) {
                container('kubectl') {
                    script {
                      sh "mkdir -p ~/.kube && cp ${KUBECONFIG} ~/.kube/config"
                      sh "kubectl get pod -n kube-ops"
                    }
                }
            }
        }
    }
}
```

```shell
# 然后我们推送一下 Jenkinsfile，这个时候观察 Pipeline 的执行
```

![ci/cd](https://picture.devops-engineer.com.cn/file/0426e21217f2279502603.jpg)

```shell
# 那么我们的 Pipeline 框架其实就算是搭好了，然后我们就要基于这个框架去填充我们的 CI/CD 的逻辑了，其实单元测试这里我是直接跳过了的，因为这里其实只需要走一些单元测试或者静态代码分析的脚本就可以了

# 我们直接开始第二阶段，也就是编译打包，其实在前面，如果看过 Dockerfile 的话，应该看得出，它其实就完成了构建的操作了，可以完全省略这一步的，不过为了展示，我还是写了一下
```

```groovy
stage('代码编译打包') {
    try {
        container('golang') {
            echo "==========编译打包阶段=========="
            sh """
              export GOPROXY=https://goproxy.cn
              GOOS=linux GORACH=amd64 go build -v -o cdk-web-service
            """
        }
    } catch (exc) {
        println "==========构建失败 --- ${currentBuild.fullDisplayName}=========="
        throw(exc)
    }
}
```

```shell
# 第三个阶段，构建 Docker 镜像，要构建 Docker 镜像，就需要提供 Docker 镜像的名称和tag，要推送到镜像仓库，也就需要提供登录仓库的用户名和密码，所以这里是需要用到 withCredentials 方法，在里面可以提供一个 credentialsId 为 dockerhub 的认证信息
```

```groovy
stage('构建 Docker 镜像') {
    withCredentials([[$class: 'UsernamePasswordMultiBinding',
                      credentialsId: 'docker-auth', 
                      usernameVariable: 'DOCKER_USER', 
                      passwordVariable: 'DOCKER_PASSWORD']]) {
        container('docker') {
            echo "打包 --- 构建镜像阶段"
            sh """
              docker login ${registryUrl} -u ${DOCKER_USER} -p ${DOCKER_PASSWORD}
              docker build -t ${image} .
              docker push ${image}
            """
        }
    }
}
```

```shell
# 然后我们要去添加一个凭证，这个凭证就是 dockerhub 的认证账号密码，凭证的 ID 就是 docker-auth，然后我们还需要定义一下 registryUrl 和 image 的信息
```

```groovy
podTemplate(cloud: "kubernetes", containers: [
    containerTemplate(name: 'golang', image: 'golang:1.21.1-alpine3.18', command: 'cat', ttyEnabled: true),
    containerTemplate(name: 'docker', image: 'docker:latest', command: 'cat', ttyEnabled: true),
    containerTemplate(name: 'kubectl', image: 'cnych/kubectl', command: 'cat', ttyEnabled: true)
], serviceAccount: 'jenkins', envVars: [
    envVar(key: 'DOCKER_HOST', value: 'tcp://docker-dind:2375')
]) {
    node(POD_LABEL) {
        def Repo = checkout scm
        def GitCommit = Repo.GIT_COMMIT
        def GitBranch = Repo.GIT_BRANCH
        def imageTag = sh(script: "git rev-parse --short HEAD", returnStdout: true).trim()
        def registryUrl = "docker.io"
        def imageEndpoint = "layzer/cdk-web-service"
        def image = "${registryUrl}/${imageEndpoint}:${imageTag}"
        
        stage('构建 Docker 镜像') {
            withCredentials([[$class: 'UsernamePasswordMultiBinding',
                              credentialsId: 'docker-auth', 
                              usernameVariable: 'DOCKER_USER', 
                              passwordVariable: 'DOCKER_PASSWORD']]) {
                container('docker') {
                    echo "打包 --- 构建镜像阶段"
                    sh """
                      docker login ${registryUrl} -u ${DOCKER_USER} -p ${DOCKER_PASSWORD}
                      docker build -t ${image} .
                      docker push ${image}
                    """
                }
            }
        }
        
        stage('执行部署') {
            withCredentials([file(credentialsId: 'kubeconfig', variable: 'KUBECONFIG')]) {
                container('kubectl') {
                    script {
                      sh "mkdir -p ~/.kube && cp ${KUBECONFIG} ~/.kube/config"
                      sh "kubectl version"
                      sh "kubectl get pod -n kube-ops"
                    }
                }
            }
        }
    }
}
```

```shell
# 然后触发执行之后我们再来看看最终的结果
```

![ci/cd](https://picture.devops-engineer.com.cn/file/9a0f22e0eed6f3f875a58.jpg)

![ci/cd](https://picture.devops-engineer.com.cn/file/9bd5d7c0b7ffac98c8ff8.jpg)

```shell
# 看到镜像了吧，那么这个其实就证明了，我们的 Push 是成功的，最后就是编写 部署的 YAML 然后将 Push 上去的镜像替换到 YAML 内，就可以实现部署应用了
```

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: cdk-web
  labels:
    app: cdk-web
spec:
  selector:
    matchLabels:
      app: cdk-web
  template:
    metadata:
      labels:
        app: cdk-web
    spec:
      containers:
      - name: cdk-web
        image: ${image}
        env:
        - name: GIN_MODE
          value: release
        ports:
        - name: http
          protocol: TCP
          containerPort: 8080
        livenessProbe:
          failureThreshold: 5
          httpGet:
            path: /health
            port: http
            scheme: HTTP
          initialDelaySeconds: 600
          periodSeconds: 10
          successThreshold: 1
          timeoutSeconds: 5
        readinessProbe:
          failureThreshold: 5
          httpGet:
            path: /health
            port: http
            scheme: HTTP
          initialDelaySeconds: 30
          periodSeconds: 5
          successThreshold: 1
          timeoutSeconds: 1
        resources:
          limits:
            cpu: 100m
            memory: 100Mi
          requests:
            cpu: 100m
            memory: 100Mi
---
apiVersion: v1
kind: Service
metadata:
  name: cdk-web
  labels:
    app: cdk-web
spec:
  type: ClusterIP
  selector:
    app: cdk-web
  ports:
  - name: http
    port: 8080
    targetPort: http
---
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: cdk-web
  labels:
    app: cdk-web
spec:
  ingressClassName: nginx
  rules:
  - host: cdk-web.devops-engineer.com.cn
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: cdk-web
            port:
              name: http 
```

```groovy
podTemplate(cloud: "kubernetes", containers: [
    containerTemplate(name: 'golang', image: 'golang:1.21.1-alpine3.18', command: 'cat', ttyEnabled: true),
    containerTemplate(name: 'docker', image: 'docker:latest', command: 'cat', ttyEnabled: true),
    containerTemplate(name: 'kubectl', image: 'cnych/kubectl', command: 'cat', ttyEnabled: true)
], serviceAccount: 'jenkins', envVars: [
    envVar(key: 'DOCKER_HOST', value: 'tcp://docker-dind:2375')
]) {
    node(POD_LABEL) {
      def Repo = checkout scm
      def GitCommit = Repo.GIT_COMMIT
      def GitBranch = Repo.GIT_BRANCH
      def imageTag = sh(script: "git rev-parse --short HEAD", returnStdout: true).trim()
      def registryUrl = "docker.io"
      def imageEndpoint = "layzer/cdk-web-service"
      def image = "${registryUrl}/${imageEndpoint}:${imageTag}"
      
      stage('构建 Docker 镜像') {
        withCredentials([[$class: 'UsernamePasswordMultiBinding',
                          credentialsId: 'docker-auth', 
                          usernameVariable: 'DOCKER_USER', 
                          passwordVariable: 'DOCKER_PASSWORD']]) {
          container('docker') {
            echo "打包 --- 构建镜像阶段"
            sh """
              docker login ${registryUrl} -u ${DOCKER_USER} -p ${DOCKER_PASSWORD}
              docker build -t ${image} .
              docker push ${image}
            """
          }
        }
      }
        
      stage('执行部署') {
        withCredentials([file(credentialsId: 'kubeconfig', variable: 'KUBECONFIG')]) {
          container('kubectl') {
            script {
              sh "mkdir -p ~/.kube && cp ${KUBECONFIG} ~/.kube/config"
              echo "============替换镜像阶段============"
              sh "sed -i -E 's|(^[ \\t]*image:).*|\\1 ${image}|' deploy/deployment.yaml"
              echo "============部署应用阶段============"
              sh "kubectl apply -f deploy/ -n cdk-service"
              echo "============检查部署阶段============"
              sh "kubectl get pod -n cdk-service -l app=cdk-web"
            }
          }
        }
      }
    }
}
```

```shell
# 如上流水线需要注意，我们第一次执行前一定要创建命名空间哦
[root@k-m-1 cdk-web-service]# kubectl create namespace cdk-service
namespace/cdk-service created

[root@k-m-1 cdk-web-service]# kubectl get pod,svc,ingress -n cdk-service 
NAME                           READY   STATUS    RESTARTS   AGE
pod/cdk-web-5c54d6f4b4-9vztt   1/1     Running   0          76s

NAME              TYPE        CLUSTER-IP    EXTERNAL-IP   PORT(S)    AGE
service/cdk-web   ClusterIP   10.96.1.165   <none>        8080/TCP   76s

NAME                                CLASS   HOSTS                            ADDRESS     PORTS   AGE
ingress.networking.k8s.io/cdk-web   nginx   cdk-web.devops-engineer.com.cn   10.0.0.11   80      76s

# OK！流水线执行完了，那么这个时候我们就是来验证一下是否能够使用了，我们解析并访问一下这个服务来看看

[root@k-m-1 cdk-web-service]# curl cdk-web.devops-engineer.com.cn:32048
{"msg":"Hello This is cdk-web-service"}

# OK！这样我们的整套流水线就跑腿通了，其余的比如扩容之类的操作，大家可以自己去思考思考如何融入到流水线，当然你是 helm 的话，那更简单，无非是我们可以在 kubectl 的镜像内融入一个 helm 然后将应用的 yaml 开发成 helm charts 就可以了
```

### 4：`ArgoCD`

```shell
# Argo CD 是一个专门为 Kubernetes 而生的，遵循声明式 GitOps 理念的持续部署工具，Argo CD 可以在 Git 存储库更改时自动同步和部署应用程序

# Argo CD 遵循 GitOps 模式，使用 Git 仓库作为定义所需应用程序状态的真实来源，Argo CD 支持多种 Kubernetes 清单：
1：kustomize
2：helm charts
3：ksonnet applications
4：jsonnet files
5：Plain directory of YAML/JSON manifests
6：Any custom config management tool configured as a config management plugin

# Argo CD 可在指定的目标环境中自动部署所需的应用程序状态，应用程序部署可以在 Git 提交时跟踪对分支，标签的更新，或固定到清单的指定版本
```

![Argo CD Architecture](https://argo-cd.readthedocs.io/en/stable/assets/argocd_architecture.png)

```shell
# 官网：https://argo-cd.readthedocs.io/en/stable/

# 下面是 Argo CD 的几个主要组件
1：API服务：API服务是一个 gRPC/REST服务，它暴露了 Web UI，CLI和CI/CD系统的使用接口，主要有以下几个功能
	1.1：应用程序管理和状态管理
	1.2：执行应用程序操作（同步，回滚，用户定义操作）
	1.3：存储仓库和集群凭证管理（存储为 K8S Secret 对象）
	1.4：认证和授权给外部身份提供者
	1.5：RBAC
	1.6：Git WebHook 事件的侦听器/转发器
2：仓库服务：存储仓库服务是一个内部服务，负责维护保存应用程序清单 Git 仓库的本地缓存，当提供以下输入时，它负责生成并返回 Kubernetes 清单
	2.1：存储 URL
	2.2：revision 版本（commit，tag，barnch）
	2.3：应用路径
	2.4：模板配置：参数，ksonnet 环境，helm values.yaml 等
3：应用控制器：应用控制器是一个 Kubernetes 控制器，它持续 watch 正在运行的应用程序并将当前的实时状态与所期望的状态（repo 中指定）进行比较，它检测应用程序的 OutOfSync 状态，并采取一些措施来同步状态，它负责调用任何用户定义的生命周期事件的钩子（PreSync，Sync，PostSync）

# 功能
1：自动部署应用程序到指定的目标环境
2：支持多种配置管理/模板工具（kustomize，helm，ksonnet，Jsonnet，plain-YAML）
3：能够管理和部署到多个集群
4：SSO 集成（OIDC，OAuth2，LDAP，SAML2.0，Github，Gitlab，Microsoft，Linkedln）
5：用于授权 多租户 和 RBAC策略
6：回滚/随时回滚到 Git 存储库中提交的任何应用配置
7：应用资源的健康状况分析
8：自动配置检测和可视化
9：自动或手动将应用同步到所需状态
10：提供应用程序活动实时可视图的 Web UI
11：用于自动化和 CI 集成的 CLI
12：WebHook 集成 （Github，Bitbucket，Gitlab）
13：用于自动化的 AccessTokens
14：PreSync，Sync，PostSync Hooks，以支持复杂的应用程序部署（例如 蓝/绿 和 金丝雀发布）
15：应用程序事件 和 API调用的审计
16：Prometheus 监控指标
17：用于覆盖 Git 中的 ksonnet/helm 参数

# 核心概念
1：Application：应用，一组由资源清单定义的 Kubernetes 资源，这是一个 CRD 资源对象
2：Application source type：用于构建应用的工具
3：Target state：目标状态，指应用实时的状态，比如部署了哪儿些 Pod 等真实状态
4：Sync Status：同步状态表示实时状态是否与目标状态一致，部署的应用程序是否与 Git 所描述的一样
5：Sync：同步指将应用程序迁移到其他目标状态的过程，比如通过对 kubernetes 集群应用变更
6：Sync operation status：同步操作状态指的是同步是否成功
7：Refresh：刷新是指将 Git 中的最新代码与实时状态进行比较，弄清楚有什么不同
8：Health：应用程序的健康状况，它是否正常运行，能否为请求提供服务
9：Tool：工具指从文件目录创建清单的工具，例如 Kustomize，或者 ksonnet 等
10：Configuration management tool：配置管理工具
11：Configuration management plugin：配置管理插件

# 安装 Argo CD

# 根据官网给到的部署方式
kubectl create namespace argocd
kubectl apply -n argocd -f https://raw.githubusercontent.com/argoproj/argo-cd/stable/manifests/install.yaml

# 当然，这是普通的使用方法，如果是生产，肯定要部署 HA 版本的
kubectl create namespace argocd
kubectl apply -n argocd -f https://raw.githubusercontent.com/argoproj/argo-cd/master/manifests/ha/install.yaml

# 部署完成之后我们来看看资源
[root@k-m-1 ~]# kubectl get pod,svc -n argocd 
NAME                                                   READY   STATUS    RESTARTS   AGE
pod/argocd-application-controller-0                    1/1     Running   0          8m15s
pod/argocd-applicationset-controller-c89d9dcc4-lspr2   1/1     Running   0          8m15s
pod/argocd-dex-server-f788954bd-gp4v5                  1/1     Running   0          6m36s
pod/argocd-notifications-controller-78f668cb45-t4tjf   1/1     Running   0          8m16s
pod/argocd-redis-79c5d7746d-nf5rq                      1/1     Running   0          31s
pod/argocd-repo-server-686d8dbdcb-vvjjq                1/1     Running   0          8m15s
pod/argocd-server-69b865f5b4-hgwmv                     1/1     Running   0          8m15s

NAME                                              TYPE        CLUSTER-IP    EXTERNAL-IP   PORT(S)                      AGE
service/argocd-applicationset-controller          ClusterIP   10.96.2.9     <none>        7000/TCP,8080/TCP            8m16s
service/argocd-dex-server                         ClusterIP   10.96.1.137   <none>        5556/TCP,5557/TCP,5558/TCP   8m16s
service/argocd-metrics                            ClusterIP   10.96.1.171   <none>        8082/TCP                     8m16s
service/argocd-notifications-controller-metrics   ClusterIP   10.96.3.121   <none>        9001/TCP                     8m16s
service/argocd-redis                              ClusterIP   10.96.0.17    <none>        6379/TCP                     8m16s
service/argocd-repo-server                        ClusterIP   10.96.3.210   <none>        8081/TCP,8084/TCP            8m16s
service/argocd-server                             ClusterIP   10.96.1.85    <none>        80/TCP,443/TCP               8m16s
service/argocd-server-metrics                     ClusterIP   10.96.2.186   <none>        8083/TCP                     8m16s

# OK 这样就部署好了，如果说遇到了镜像的问题，可以选择使用代理：https://dockerproxy.com

# 同时我们看到有一个 argocd-server 服务提供了 80 和 443，我们可以通过 Ingress 来暴露并访问它

# 然后 ArgoCD 还有一个 CLI 的工具，我们可以安装一下

curl -sSL -o /usr/local/bin/argocd https://ghproxy.com/https://github.com/argoproj/argo-cd/releases/download/v2.8.4/argocd-linux-amd64
chmod +x /usr/local/bin/argocd

# Argo CD 会运行一个 gRPC 服务（由 CLI 使用）和 HTTP/HTTPS 服务（UI使用），这两种协议都由 argocd-server 服务在 80 和 443 端口进行暴露，不过因为 Argo CD 本身支持了一个 gRPC 的方式，所以 如果要用一个 Ingress 规则去暴露它，则需要配置一个注解（以Ingress-nginx为例）ssl-passthrough，但是这个透传需要在 ingress 控制器上开启这个功能，这里我选择两个规则
```

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: argocd-server-http-ingress
  namespace: argocd
  annotations:
    nginx.ingress.kubernetes.io/backend-protocol: "HTTP"
spec:
  ingressClassName: "nginx"
  tls:
  - hosts:
    - argocd.devops-engineer.com.cn
    secretName: argocd-secret
  rules:
  - host: argocd.devops-engineer.com.cn
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: argocd-server
            port:
              name: http
---
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: argocd-server-grpc-ingress
  namespace: argocd
  annotations:
    nginx.ingress.kubernetes.io/backend-protocol: "GRPC"
spec:
  ingressClassName: "nginx"
  tls:
  - hosts:
    - grpc.argocd.devops-engineer.com.cn
    secretName: argocd-secret
  rules:
  - host: grpc.argocd.devops-engineer.com.cn
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: argocd-server
            port:
              name: https
```

```shell
# 不过如果是自签名的话，我们还需要在 argocd-server 的 deployment 中添加一个 args

      containers:
      - args:
        - /usr/local/bin/argocd-server
        - --insecure  # 添加这一行

# 本地的话，需要写一下 hosts 解析，然后就可以访问 argocd 了
账号：admin
# 密码需要这样查看
[root@k-m-1 argocd]# kubectl get secrets -n argocd argocd-initial-admin-secret -o jsonpath="{.data.password}" | base64 -d
2Ko7Qm4cx5gQPih5

# 修改密码可以使用 CLI 等陆 ArgoCD 之后修改
[root@k-m-1 cdk-web-service]# argocd account update-password --current-password <old_password> --new-password <new_password>
Password updated
Context 'grpc.argocd.devops-engineer.com.cn' updated
```

![argo-cd](https://picture.devops-engineer.com.cn/file/ec8df8239dc62bb438c8f.jpg)

![argo-cd](https://picture.devops-engineer.com.cn/file/64e1db05e58c5437ee1d4.jpg)



```shell
# 然后我们就可以使用命令来添加集群了（我在本机添加了hosts解析了）
[root@k-m-1 argocd]# argocd login grpc.argocd.devops-engineer.com.cn
WARNING: server certificate had error: tls: failed to verify certificate: x509: certificate is valid for ingress.local, not grpc.argocd.devops-engineer.com.cn. Proceed insecurely (y/n)? y
Username: admin
Password: 
'admin:login' logged in successfully
Context 'grpc.argocd.devops-engineer.com.cn' updated

# OK 这样就登录完成了，那么我们验证一下
[root@k-m-1 argocd]# argocd context
CURRENT  NAME                                SERVER
*        grpc.argocd.devops-engineer.com.cn  grpc.argocd.devops-engineer.com.cn

[root@k-m-1 argocd]# argocd account list
NAME   ENABLED  CAPABILITIES
admin  true     login

# 然后可以添加集群
[root@k-m-1 argocd]# argocd cluster list
SERVER                          NAME        VERSION  STATUS   MESSAGE                                                  PROJECT
https://kubernetes.default.svc  in-cluster           Unknown  Cluster has no applications and is not being monitored.  

# 默认的集群貌似有问题，不过不重要，我们接着添加就可以了
[root@k-m-1 argocd]# kubectl config get-contexts -o name
kubernetes-admin@kubernetes

[root@k-m-1 argocd]# argocd cluster add kubernetes-admin@kubernetes
WARNING: This will create a service account `argocd-manager` on the cluster referenced by context `kubernetes-admin@kubernetes` with full cluster level privileges. Do you want to continue [y/N]? y
INFO[0001] ServiceAccount "argocd-manager" created in namespace "kube-system" 
INFO[0001] ClusterRole "argocd-manager-role" created    
INFO[0001] ClusterRoleBinding "argocd-manager-role-binding" created 
INFO[0006] Created bearer token secret for ServiceAccount "argocd-manager" 
Cluster 'https://10.0.0.11:6443' added

# 然后我们就可以通过 CLI 或者 UI 去创建应用等操作了
```

![argo-cd](https://picture.devops-engineer.com.cn/file/ca5d181ec033b85078e2c.jpg)

![argo-cd](https://picture.devops-engineer.com.cn/file/e76dca772cba3de375333.jpg)

![argo-cd](https://picture.devops-engineer.com.cn/file/bf4568ce4c8f92106cfa2.jpg)

![argo-cd](https://picture.devops-engineer.com.cn/file/5d2c8b6c48198b3ab5bb5.jpg)

![argo-cd](https://picture.devops-engineer.com.cn/file/d7e327431b20098e77519.jpg)

![argo-cd](https://picture.devops-engineer.com.cn/file/8b8c903cca27b3c48331d.jpg)

```shell
# 当我们在代码库进行了变更，比如我变更一个副本
```

![argo-cd](https://picture.devops-engineer.com.cn/file/2ed61f34270b7972a9c54.jpg)

![argo-cd](https://picture.devops-engineer.com.cn/file/212527d0eb2e055b9ad8e.jpg)

![argo-cd](https://picture.devops-engineer.com.cn/file/49832f0fb0756de8f1978.jpg)

![argo-cd](https://picture.devops-engineer.com.cn/file/759e8f74c1fef0a897169.jpg)

```shell
# 当我们再次同步之后，然后看看我们的变更是否生效了
[root@k-m-1 ~]# kubectl get pod 
NAME                           READY   STATUS    RESTARTS   AGE
guestbook-ui-c84b89b4b-92kk4   1/1     Running   0          45m
guestbook-ui-c84b89b4b-p49bl   1/1     Running   0          2m12s

# 看到副本变更了，也就是我们代码库中的期望值了，其实这个操作在 CLI 也支持，大家可以去发掘一下使用方法，那么通过 Web UI 创建应用我们讲过了，那么还有一种就是通过 K8S 的 CRD 来创建应用

[root@k-m-1 ~]# kubectl get crd | grep argoproj
applications.argoproj.io                              2023-10-02T06:02:02Z
applicationsets.argoproj.io                           2023-10-02T06:02:02Z
appprojects.argoproj.io                               2023-10-02T06:02:03Z

# 可以看到，这里有三个 CRD 资源对象，我们可以通过创建 CRD 同步创建应用，其实是和 Web UI 一样的，我们 Web UI 创建之后，CRD 也会同步创建一个对象
[root@k-m-1 ~]# kubectl get applications -n argocd 
NAME        SYNC STATUS   HEALTH STATUS
guestbook   Synced        Healthy

# 可以看到这里的确是有一个和 Web UI 一样的应用，我们可以通过导出它的 YAML 来看看都是怎样定义的
[root@k-m-1 ~]# kubectl get applications -n argocd guestbook -o yaml
```

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: guestbook
  namespace: argocd
spec:
  destination:
    namespace: default
    server: https://10.0.0.11:6443
  project: default
  source:
    path: guestbook
    repoURL: http://gitea.kube-ops.svc.cluster.local/gitlayzer/argocd-example-apps.git
    targetRevision: HEAD
```

```shell
# 我这里做了一些简化，关于 status 的东西其实我们不太需要关注，主要关注这个应用的主体部分就行了，我们可以通过这样的 YAML 来创建应用，这也就是我们的 CRD 的方式了，下面我们将做的是将 前面 Jenkins 的 Pipeline 改造成符合 GitOps 的风格的流水线
```

### 5：`Jenkins + Argo CD`

```shell
# 因为前面讲过，ArgoCD 主要就是负责 CD 方面的操作，所以我们主要就是将上面的流水线中的 Deploy 这一步放到 ArgoCD 中去做，OK，那么我们其实在生产中都知道一个操作，就是我们的 CI/CD 的部署文件其实一般是不会和代码存放在一起的，我们会把它单独分离出来，我前公司使用的 Coding 就是这样操作的，所以这里我也选择重新创建一个仓库用于专门保存应用的部署清单
```

![gitea](https://picture.devops-engineer.com.cn/file/71f50a02133dfe8e1cfd1.jpg)

```shell
# 这个仓库其实就是把 cdk-web-service 中的 deploy 下的部署文件分离了出来，仅此而已，然后我们去创建一个应用，这个应用就针对这个代码仓库，我们这次使用 CRD 创建
```

```yaml
# cdk-web-service-app.yaml
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: cdk-web-service
  namespace: argocd
spec:
  destination:
    namespace: default
    server: 'https://10.0.0.11:6443'
  source:
    path: /
    repoURL: >-
      http://gitea.kube-ops.svc.cluster.local/gitlayzer/cdk-web-service-deploy.git
    targetRevision: HEAD
  project: default
  syncPolicy:
    automated:
      # 开启后 Git Repo 中删除资源会自动在环境中删除对应的资源
      prune: true
      # 自动痊愈，强制以 Git Repo 状态为准，手动在环境中修改不会生效
      selfHeal: true
```

```shell
# 我们可以创建如上资源
[root@k-m-1 ~]# kubectl apply -f cdk-web-service-app.yaml 
application.argoproj.io/cdk-web-service created

# 检查
[root@k-m-1 ~]# kubectl get applications -n argocd 
NAME              SYNC STATUS   HEALTH STATUS
cdk-web-service   Synced        Healthy
guestbook         Synced        Healthy

# 不过这里需要说明一点，如果 Ingress 是在集群内不可访问的，那么需要修改 argocd 的 configmap
[root@k-m-1 jenkins]# kubectl edit cm -n argocd argocd-cm
apiVersion: v1
# 添加这些信息
data:
  resource.customizations: |
    networking.k8s.io/Ingress:
        health.lua: |
          hs = {}
          hs.status = "Healthy"
          return hs
# 到这里是结尾
kind: ConfigMap
metadata:
  annotations:
    kubectl.kubernetes.io/last-applied-configuration: |
      {"apiVersion":"v1","kind":"ConfigMap","metadata":{"annotations":{},"labels":{"app.kubernetes.io/name":"argocd-cm","app.kubernetes.io/part-of":"argocd"},"name":"argocd-cm","namespace":"argocd"}}
  creationTimestamp: "2023-10-02T06:02:03Z"
  labels:
    app.kubernetes.io/name: argocd-cm
    app.kubernetes.io/part-of: argocd
  name: argocd-cm
  namespace: argocd
  resourceVersion: "3892028"
  uid: 78efb195-e8cb-4f7f-bf2c-16dd9d173122


# 这个时候我们就可能会发现，我们的 default 的 namespace 下面就部署了我们这个应用了
[root@k-m-1 ~]# kubectl get pod,svc,ingress -l app=cdk-web
NAME                           READY   STATUS    RESTARTS   AGE
pod/cdk-web-79b8654794-6dg9d   1/1     Running   0          4m45s

NAME              TYPE        CLUSTER-IP    EXTERNAL-IP   PORT(S)    AGE
service/cdk-web   ClusterIP   10.96.1.152   <none>        8080/TCP   4m45s

NAME                                CLASS   HOSTS                            ADDRESS     PORTS   AGE
ingress.networking.k8s.io/cdk-web   nginx   cdk-web.devops-engineer.com.cn   10.0.0.11   80      2m15s

# 看到了吧，的确是部署出来了，这个就是自动的好处，我们不需要同步它了，然后我们就可以去 Jenkins 的 Pipeline 中进行一些修改了
```

```groovy
podTemplate(cloud: "kubernetes", containers: [
    containerTemplate(name: 'golang', image: 'golang:1.21.1-alpine3.18', command: 'cat', ttyEnabled: true),
    containerTemplate(name: 'docker', image: 'docker:latest', command: 'cat', ttyEnabled: true),
    containerTemplate(name: 'sed', image: 'cnych/yq-jq:git', command: 'cat', ttyEnabled: true)
], serviceAccount: 'jenkins', envVars: [
    envVar(key: 'DOCKER_HOST', value: 'tcp://docker-dind:2375')
]) {
    node(POD_LABEL) {
      def Repo = checkout scm
      def GitCommit = Repo.GIT_COMMIT
      def GitBranch = Repo.GIT_BRANCH
      def imageTag = sh(script: "git rev-parse --short HEAD", returnStdout: true).trim()
      def registryUrl = "docker.io"
      def imageEndpoint = "layzer/cdk-web-service"
      def image = "${registryUrl}/${imageEndpoint}:${imageTag}"
      
      stage('构建 Docker 镜像') {
        withCredentials([[$class: 'UsernamePasswordMultiBinding',
                          credentialsId: 'docker-auth',
                          usernameVariable: 'DOCKER_USER',
                          passwordVariable: 'DOCKER_PASSWORD']]) {
          container('docker') {
            echo "打包 --- 构建镜像阶段"
            sh """
              docker login ${registryUrl} -u ${DOCKER_USER} -p ${DOCKER_PASSWORD}
              docker build -t ${image} .
              docker push ${image}
            """
          }
        }
      }
      
      stage('修改镜像') {
        withCredentials([[$class: 'UsernamePasswordMultiBinding',
                          credentialsId: 'gitea-auth',
                          usernameVariable: 'GITEA_USER',
                          passwordVariable: 'GITEA_PASS']]) {
          container('sed') {
            sh """
              git clone http://gitea.kube-ops.svc.cluster.local/gitlayzer/cdk-web-service-deploy.git
              cd cdk-web-service-deploy
              sed -i -E 's|(^[ \\t]*image:).*|\\1 ${image}|' deployment.yaml
              git add .
              git config --global user.name "gitlayzer"
              git config --global user.email "gitlayzer@gmail.com"
              git commit -m "Change Image"
              git config --global credential.helper '!f() { echo "username=${GITEA_USER}"; echo "password=${GITEA_PASS}"; }; f'
              git push origin master
            """
          }
        }        
      }
    }
}
```

```shell
# 这里需要注意的就是，我们需要用到一个凭据，因为修改完的文件需要再次推送到仓库中去，然后，我们就可以删除应用仓库的 deploy，提交修改后的 Jenkinsfile 了，因为我们的应用是设置了自动，所以 argocd 会自动去检测代码仓库内的文件，检测到了之后会自动去满足期望状态，那么我来修改一下代码，将内容换成v2版本
[root@k-m-1 cdk-web-service]# git add .
[root@k-m-1 cdk-web-service]# git commit -m "Change main.go"
[master 4dd3f04] Change main.go
 1 file changed, 1 insertion(+), 1 deletion(-)
[root@k-m-1 cdk-web-service]# git push origin master
Counting objects: 5, done.
Delta compression using up to 8 threads.
Compressing objects: 100% (3/3), done.
Writing objects: 100% (3/3), 287 bytes | 0 bytes/s, done.
Total 3 (delta 2), reused 0 (delta 0)
remote: . Processing 1 references
remote: Processed 1 references in total
To http://git.devops-engineer.com.cn/gitlayzer/cdk-web-service.git
   3174d35..4dd3f04  master -> master
```

```go
package main

import (
	"github.com/gin-gonic/gin"
	"log"
	"net/http"
	"os"
)

var (
	ServiceMode = os.Getenv("GIN_MODE")
)

func init() {
	if ServiceMode == "" {
		ServiceMode = gin.DebugMode
	}
}

func Handler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"msg": "Hello This is cdk-web-service, This Version is v2",
	})
}

func Health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"code": http.StatusOK,
		"msg":  "true",
	})
}

func main() {
	gin.SetMode(ServiceMode)

	r := gin.Default()

	r.GET("/", Handler)
	r.GET("/health", Health)

	if err := r.Run(":8080"); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
```

```shell
# 然后我们测试访问一下
[root@k-m-1 cdk-web-service]# curl cdk-web.devops-engineer.com.cn
{"msg":"Hello This is cdk-web-service, This Version is v2"}

# 欸这个时候我们会发现，我们应用被更新了，那么这个其实就是我们所说的 GitOps 了，其实在前面我们讲过，GitLab，GitHub，Gitea等代码仓库也有一定的 CI/CD 的能力，比如 GitLab 中自带的 CI/CD，它依赖 Gitlab-Runner 来做，依赖的是 .gitlab-ci.yaml 进行 CI/CD 流水线，而 Github 和 Gitea 其实也依赖 Runner，它们是按照 Action 来做 CI/CD 和 Gitlab 其实基本差不多，只不过，我们这里使用的是 Jenkins 来做 CI，ArgoCD 去做发布，然后我们再深度集成一下 Rollout 进行灰度发布等操作，前面其实我的博客有讲 Kruise Rollout，其实 Argo 也有一个 Rollout，它叫 argo rollouts，和前面的博客讲到的差不多，也是一个基于 K8S 一组控制器，实现的功能也是更深度的发布的策略等，下面我们会讲到它
```

### 6：`Argo CD Image Update`

```shell
# 这是一种自动更新由 Argo CD 管理的 Kubernetes 工作负载容器镜像的工具，该工具可以检查与kubernetes工作负载一起部署的容器镜像的版本，并使用 Argo CD 自动将其更新到允许的最新版本，它通过为 Argo CD 应用程序设置适当的参数来工作，类似于 argocd app set --helm-set image.tag=v1.0.1，但是是以完全自动化的方式

# Argo CD Image Updater 会定期轮询 Argo CD 中配置的应用程序，并查询相应的镜像仓库以获取可能的新版本，如果在仓库中找到新版本的镜像，并满足版本约束，Argo CD 镜像更新程序会讲提示 Argo CD 使用新版本的镜像更新应用程序

# 根据您的应用自动同步策略，Argo CD 将自动部署新的镜像版本或将应用程序标记为不同步，您可以通过同步应用程序来手动触发镜像更新

# 特征
1：更新由ArgoCD管理且由helm或者kustomize工具生成的应用程序镜像
2：根据不同的更新策略更新应用镜像
	2.1：semver：根据给定的镜像约束更新到允许的最高版本
	2.2：latest：更新到最近创建的镜像标签
	2.3：name：更新到按字母顺序排序的列表中的最后一个标签
	2.4：digest：更新到可变标签的最新推送版本
3：支持广泛使用的容器镜像仓库
4：通过配置支持私有容器镜像仓库
5：可以将更改写回 Git
6：能够使用匹配器函数过滤镜像仓库返回的标签列表
7：在kubernetes集群中运行，或者可以从命令独立使用
8：能够执行应用程序的并行更新

# 另外需要注意的是使用工具目前有几个限制
1：想要更新容器镜像的应用程序必须使用 Argo CD 进行管理，不支持未使用 Argo CD 管理的工作负载
2：Argo CD 镜像更新程序只能更新其清单使用kustomize 或者 helm 呈现的应用程序的容器镜像，特别是在 helm 情况下，模板需要支持使用参数（image.tag）
3：镜像拉取密钥必须在于 Argo CD Image Updater 运行（或有权访问）的同一个kubernetes集群中，目前无法从其他集群获取这些机密信息

# 安装（建议运行在 Argo CD 相同的命名空间，但是这不是必须的，事实上，它们甚至都可以不在一个集群中运行，或者说，它根本都不需要访问任何集群，但是如果不访问集群，某些功能可能无法使用，所以还是比较建议和 Argo CD 装在一起的）

# URL：https://github.com/argoproj-labs/argocd-image-updater
# DOCS：https://argocd-image-updater.readthedocs.io/en/stable/

[root@k-m-1 ~]# kubectl apply -n argocd -f https://raw.githubusercontent.com/argoproj-labs/argocd-image-updater/stable/manifests/install.yaml
serviceaccount/argocd-image-updater created
role.rbac.authorization.k8s.io/argocd-image-updater created
rolebinding.rbac.authorization.k8s.io/argocd-image-updater created
configmap/argocd-image-updater-config created
configmap/argocd-image-updater-ssh-config created
secret/argocd-image-updater-secret created
deployment.apps/argocd-image-updater created

# 检查部署
[root@k-m-1 ~]# kubectl get pod -n argocd 
NAME                                               READY   STATUS    RESTARTS       AGE
argocd-application-controller-0                    1/1     Running   1 (2d3h ago)   3d2h
argocd-applicationset-controller-c89d9dcc4-lspr2   1/1     Running   1 (2d3h ago)   3d2h
argocd-dex-server-f788954bd-gp4v5                  1/1     Running   1 (2d3h ago)   3d2h
argocd-image-updater-84ffbd4747-w25kf              1/1     Running   0              71s
argocd-notifications-controller-78f668cb45-t4tjf   1/1     Running   1 (2d3h ago)   3d2h
argocd-redis-79c5d7746d-nf5rq                      1/1     Running   1 (2d3h ago)   3d1h
argocd-repo-server-686d8dbdcb-vvjjq                1/1     Running   1 (2d3h ago)   3d2h
argocd-server-695d8655bd-zdspw                     1/1     Running   1 (2d3h ago)   2d23h

# Argo CD Image Updater 安装完成后我们就可以去监听镜像是否发生了变化，而不需要在 CI 流水线中手动提交修改资源清单到代码仓库中了

# 现在我们可以去删除前面的 APP 
[root@k-m-1 ~]# argocd app delete argocd/cdk-web-service --cascade 
Are you sure you want to delete 'argocd/cdk-web-service' and all its resources? [y/n] y
application 'argocd/cdk-web-service' deleted

# 那么我们接下来创建一个新的应用程序（创建一个需要的镜像仓库的 Secret）
[root@k-m-1 ~]# kubectl create -n argocd secret docker-registry dockerhub-auth \
--docker-username <dockerhub_username> \
--docker-password <dockerhub_password> \
--docker-server "https://registry-1.docker.io"

secret/dockerhub-auth created

# 还有一个就是 Git 仓库的账号密码，也要配置一个 Secret 提供使用
[root@k-m-1 ~]# kubectl create secret generic gitea-auth -n argocd \
--from-literal username=gitlayzer \
--from-literal password=gitlayzer 

secret/gitea-auth created

# 创建方法在如下的 URL 内可以找到
# URL：https://argocd-image-updater.readthedocs.io/en/stable/basics/update-methods/
```

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: cdk-web-service
  namespace: argocd
  annotations:
    argocd-image-updater.argoproj.io/image-list: myalias=layzer/cdk-web-service               # 定义镜像仓库
    argocd-image-updater.argoproj.io/myalias.allow-tags: regexp:^.*$                          # 允许使用的tag（支持正则表达式）
    argocd-image-updater.argoproj.io/myalias.pull-secret: pullsecret:argocd/dockerhub-auth    # 镜像仓库的认证 Secret
    argocd-image-updater.argoproj.io/myalias.update-strategy: latest                          # 更新策略（上面讲过）
    argocd-image-updater.argoproj.io/myalias.force-update: "true"                             # 强制更新
    argocd-image-updater.argoproj.io/write-back-method: argocd                                # 通过 ArgoCD 修改镜像（两分钟检测一次）
spec:
  destination:
    namespace: "default"
    server: 'https://10.0.0.11:6443'
  source:
    path: "helm"
    repoURL: "http://gitea.kube-ops.svc.cluster.local/gitlayzer/cdk-web-service-deploy.git"
    targetRevision: "master"
    helm:
      valueFiles:
        - values.yaml
  project: "default"
  syncPolicy:
    automated:
      prune: true
      selfHeal: true
    syncOptions:
      - CreateNmespace=true
```

```shell
# 我们其实前面也说了，它只针对 kustomize 和 helm 生效，那么我们还得针对这个仓库进行一下 helm 的改造，这个改造我会放在 Github 上
# URL：https://github.com/gitlayzer/cdk-web-service

# 然后我们就可以去创建这个应用
[root@k-m-1 ~]# kubectl apply -f cdk-web-service-app.yaml 
application.argoproj.io/cdk-web-service created
[root@k-m-1 ~]# kubectl get app -n argocd 
NAME              SYNC STATUS   HEALTH STATUS
cdk-web-service   Synced        Healthy

# 然后我们就可以去修改 Pipeline 让后去掉部分参数，如下
```

```yaml
podTemplate(cloud: "kubernetes", containers: [
    containerTemplate(name: 'golang', image: 'golang:1.21.1-alpine3.18', command: 'cat', ttyEnabled: true),
    containerTemplate(name: 'docker', image: 'docker:latest', command: 'cat', ttyEnabled: true)
], serviceAccount: 'jenkins', envVars: [
    envVar(key: 'DOCKER_HOST', value: 'tcp://docker-dind:2375')
]) {
    node(POD_LABEL) {
      def Repo = checkout scm
      def GitCommit = Repo.GIT_COMMIT
      def GitBranch = Repo.GIT_BRANCH
      def imageTag = sh(script: "git rev-parse --short HEAD", returnStdout: true).trim()
      def registryUrl = "docker.io"
      def imageEndpoint = "layzer/cdk-web-service"
      def image = "${registryUrl}/${imageEndpoint}:${imageTag}"
      
      stage('构建 Docker 镜像') {
        withCredentials([[$class: 'UsernamePasswordMultiBinding',
                          credentialsId: 'docker-auth',
                          usernameVariable: 'DOCKER_USER',
                          passwordVariable: 'DOCKER_PASSWORD']]) {
          container('docker') {
            echo "打包 --- 构建镜像阶段"
            sh """
              docker login ${registryUrl} -u ${DOCKER_USER} -p ${DOCKER_PASSWORD}
              docker build -t ${image} .
              docker push ${image}
            """
          }
        }
      }
    }
}
```

```shell
# 可以看到，我们只需要这么多就够了，然后我们推送一下，等待看看结果
# 看 argocd-image-updater 的日志，可以看到它监听到了一个新的镜像，并且给我们更新了，然后我们只需要看新的应用是否是新的镜像就好了
time="2023-10-06T07:26:40Z" level=info msg="Starting image update cycle, considering 1 annotated application(s) for update"
time="2023-10-06T07:26:45Z" level=info msg="Setting new image to layzer/cdk-web-service:f4a234f" alias=myalias application=cdk-web-service image_name=layzer/cdk-web-service image_tag=4dd3f04 registry=
time="2023-10-06T07:26:45Z" level=info msg="Successfully updated image 'layzer/cdk-web-service:4dd3f04' to 'layzer/cdk-web-service:f4a234f', but pending spec update (dry run=false)" alias=myalias application=cdk-web-service image_name=layzer/cdk-web-service image_tag=4dd3f04 registry=
time="2023-10-06T07:26:45Z" level=info msg="Committing 1 parameter update(s) for application cdk-web-service" application=cdk-web-service
time="2023-10-06T07:26:45Z" level=info msg="Successfully updated the live application spec" application=cdk-web-service
time="2023-10-06T07:26:45Z" level=info msg="Processing results: applications=1 images_considered=1 images_skipped=0 images_updated=1 errors=0"

[root@k-m-1 ~]# kubectl get pod cdk-web-service-5bcb79f57c-5gx4r -ojsonpath={.spec.containers[0].image}
layzer/cdk-web-service:f4a234f

# 可以看到，的确是更新了，我这里本来使用的是 Git，但是 Git 推送貌似有点问题，这个问题我在代码中也看到了抛出的日志，可能是我走的是 svc 的 FQDN，所以不识别，然后我就换成了不推送了，直接让 ArgoCD 去修改镜像，根据策略，我们现在就是去检测最新的镜像，然后进行更改，那么这个就是 Argo CD Image Updater 的作用，它可以使得我们不再 Jenkins 内再多做任何操作，我们只在 Jenkins 内关注如何进行 CI 操作就可以了，那么这个就是我们目前做的从 Jenkins 的 DevOps 迁移到 GitOps 的方案，其实我们前面说过，可以结合 Argo Rollouts，Kruise Rollouts 之类的实现灰度发布等更高级的玩法
```

### 7：`Argo Rollouts`

```shell
# Argo Rollouts 是一个 Kubernets Operator 实现，它为 Kubernetes 提供了更加高级的部署能力，比如：蓝绿，金丝雀，金丝雀分析，实验和渐进式交付功能，为云原生应用和服务实现自动化，基于 GitOps 的逐步交付

# 它支持如下特性：
1：蓝绿更新策略
2：金丝雀更新策略
3：更加细粒度，加权流量拆分
4：自动回滚
5：手动回滚
6：可定制的指标查询和业务KPI分析
7：Ingress 控制器集成：Nginx，ALB
8：服务网格集成：Istio，Linkerd，SMI（已进入归档）
9：Metrics指标集成：Prometheus，Wavefront，Web，Kubernetes Jobs，Datadog，New Relic，Graphite，InfluxDB

# 实现原理
# 与 Deployment 对象类似，Argo Rollouts 控制器将管理 ReplicaSets 的创建，缩放和删除，这些 ReplicaSet 由 Rollout 资源中的 spec.template 定义，使用与 Deployment 对象相同的 Pod 模板

# 当 spec.template 变更时，这会向 Argo Rollouts 控制器发出信号，表示将引入新的 ReplicaSet，控制器将使用 spec.Strategy 字段内的策略来确定从旧的 ReplicaSet 到新 ReplicaSet 的 rollout 将如何进行，一旦这个新的 ReplicaSet 被放大（可以选择通过一个Analysis），控制器会将其标记为 稳定

# 如果在 spec.template 从稳定的 ReplicaSet 过渡到新的 ReplicaSet 的过程中发生了另一次变更（即在发布过程中变更了应用程序版本），那么之前新的 ReplicaSet 将缩小，并且控制器将尝试发布反映更新 spec.template 字段的 ReplicaSet

# 相关概念
# Rollout（滚动）
# Rollout 是一个 Kubernetes CRD 资源，相当于是 Kubernetes Deployment 对象，在需要更高级的部署或渐进式交付功能的情况下，它皆在取代 Deployment 对象，Rollout 提供了 Kubernetes Deployment 所不能提供的功能
1：蓝绿发布
2：金丝雀发布
3：与 Ingress 控制器和服务网格整合，实现高级流量路由
4：与用于蓝绿和金丝雀分析的指标提供者集成
5：根据成功或失败的指标，自动发布或回滚

# 渐进式交付
# 渐进式交付是以受控和渐进的方式发布产品更新的过程，从而降低发布的风险，通常将自动化和指标分析结合起来以驱动更新的自动升级或回滚
# 渐进式交付通常被描述为持续交付的演变，将 CI/CD 中的速度优势扩展到部署过程，通过将新版本限制在一部分用户，观察和分析正确的行为，然后逐渐增加更多的流量，同时不断验证其正确性

# 部署策略
1：RollingUpdate（滚动更新）：慢慢的用新版本替换旧版本，随着新版本的出现，旧版本慢慢缩减，以保持应用程序的总数量，这是 Deployment 对象的默认策略
2：Reacreate（重新创建）：Recreate会在启动新版本之前删除旧版本的应用程序，这可以确保应用程序的两个版本永远不会同时运行，但是在部署期间会出现停机时间
3：Blue-Green（蓝绿）：蓝绿发布指同时部署了新旧两个版本的应用程序，在此期间，只有旧版本的应用会收到生产流量，这允许开发人员在将实时流量切换到新版本之前针对新版本进行测试
4：Canary（金丝雀）：金丝雀发布指将一部分用户暴露在新版本的应用程序中，而将其余流量给旧版本，一旦新版本被验证是正确的，新版本可以逐渐取代旧版本，Ingress 控制器和服务网格，如：Ingress-nginx 和 Istio，可以使金丝雀和流量拆分模式比原生的更复杂（例如：实现非常细粒度的流量分割，基于 HTTP 头的分割）

# 场景
1：用户希望在新版本开始为生产提供服务之前对其进行最后一分钟的功能测试，通过 BlueGreen 策略，Argo Rollouts 允许用户指定预览服务和活跃服务，Rollout 将配置预览服务以将流量发送到新版本，同时活跃服务将继续接收生产流量，一旦达到要求，则可以将预览服务提升为新的活跃服务
2：在新版本开始接收实时流量之前，需要预先执行一套通用步骤，通过使用 BlueGreen 策略，用户可以在不接收来自活动服务的流量的情况下启动新版本，一旦这些步骤执行完毕，就可以将流量切换到新版本了
3：用户希望在几个小时内将一小部分生产流量提供给他们应用程序的新版本，之后，他们希望缩小版本规模，并查看一些指标确定新版本与旧版本相比是否具有性能问题，然后他们将决定是否为切换到新版本，使用金丝雀策略，Rollout 可以使用新版本扩大 ReplicaSet 的规模，以接收指定百分比的流量，等待指定的时间，然后将百分比设置回 0，然后等待用户满意后再发布，为所有流量提供服务
4：一个用户想慢慢给新版本增加生产流量，先给它一小部分的实时流量，然后等待一段时间再给新版本更多的流量，最终，新版本将接收所有生产流量，使用金丝雀策略，用户指定他们希望新版本接收的百分比以及在百分比之前的等待时间
5：用户想要使用 Deployment 中的正常滚动更新策略，如果用户使用没有步骤的金丝雀策略，Rollout 将使用 maxSurge 和 最大不可用值来滚动到新版本
```

![argo-rollout](https://argoproj.github.io/argo-rollouts/architecture-assets/argo-rollout-architecture.png)

```shell
# Rollout Controller
# 这是主控制器， 用于监视集群的事件并在 Rollout 类型的资源发生变更时做出反应，控制器将读取 rollout 的所有详细信息，并使集群处于 rollout 定义中描述的相同状态
# 请注意，Argo Rollouts 不会篡改或影响正常 Deployment 资源上发生的任何变更，这意味这你可以在一个使用其他方法部署应用在集群中安装 Argo Rollouts

# Rollout 资源
# Rollout 资源是 Argo Rollouts 引入和管理的一种自定义 Kubernetes 资源，它与原生的 kubernetes Deployment 资源基本兼容，但有额外的资源来控制增加高级的部署方法，如金丝雀和蓝绿部署
# Argo Rollouts 控制器将只对 Rollout 资源的变化做出反应，不会对正常的 Deployment 资源做任何事情，所以如果你想用 Argo Rollouts 管理你的 Deployment，你需要将你的 Deployment 迁移到 Rollouts

# 旧版和新版的 ReplicaSets
# 这是标准的 Kubernetes ReplicaSet 资源的实例，Argo Rollouts 给它们添加了一些额外的元数据，以便跟踪属于应用程序的不同版本
# 还要注意的是，参加 Rollout 的 ReplicaSet 完全由控制器自动管理，你不该用外部工具来篡改它们

# Ingress/Service
# 用户的流量进入集群后，被重定向到合适的版本，Argo Rollouts 使用标准的 Kubernetes Service 资源，但有一些额外的元数据
# Argo Rollouts 在网络配置上非常灵活，首先，可以在 Rollout 期间使用不同的服务，这些服务仅适用与新版本，仅适用于旧版本或者两者都适用，特别是对于 Canary 部署，Argo Rollouts 支持多种 服务网格 和 Ingress 解决方案，用于特定百分比拆分流量，而不是基于 Pod 数量进行简单的配置

# AnalysisTemplate 与 AnalysisRun
# Analysis 是一种自定义 Kubernetes 资源，它将 Rollout 连接到指标提供程序，并为某些指标定义特定阈值，这些阈值将决定 Rollout 是否成功，对于每个 Analysis，你可以定义一个或者多个指标查询以及预期结果，如果指标查询正常，则 Rollout 将继续发布，如果指标显示失败，则自动回滚，如果指标无法提供成功/失败的结果，则暂停发布
# 为了执行分析，Argo Rollouts 提供了两个自定义的 Kubernetes 资源：AnalysisTemplate 和 AnalysisRun

# AnalysisTemplate：包含有关要查询哪儿些指标的说明，附加到 Rollout 的实际结果是 AnalysisRun 自定义资源，可以在特定的 Rollout 上定义 AnalysisTemplate，也可以在集群上定义全局的 AnalysisTemplate，以供多个 Rollout 共享作为 ClusterAnalysisTemplate，而 AnalysisRun 资源的规范仅限于特定的 Rollout
# 请注意，在 Rollout 中使用分析和指标是完全可选的，你可以通过 API 或 CLI 手动暂停和继续发布，也可以使用其他外部方法（例如冒烟测试），你不需要仅使用 Argo Rollouts 的指标解决方案，你还可以在 Rollout 中混合自动（即基于分析）和手动步骤
# 除了指标之外，你还可以通过运行 Kubernetes Job 或 运行 webhook 来决定发布的成功与否

# Metric Providers
# Argo Rollouts 包括多个流行指标提供程序的本机集成，你可以在分析资源中使用这些提供程序来自动升级或回滚部署，有关特定设置项，可以查看如下文档
# URL：https://argoproj.github.io/argo-rollouts/features/analysis/

# 安装
[root@k-m-1 ~]# kubectl create namespace argo-rollouts
namespace/argo-rollouts created
[root@k-m-1 ~]# kubectl apply -n argo-rollouts -f https://github.com/argoproj/argo-rollouts/releases/latest/download/install.yaml
customresourcedefinition.apiextensions.k8s.io/analysisruns.argoproj.io created
customresourcedefinition.apiextensions.k8s.io/analysistemplates.argoproj.io created
customresourcedefinition.apiextensions.k8s.io/clusteranalysistemplates.argoproj.io created
customresourcedefinition.apiextensions.k8s.io/experiments.argoproj.io created
customresourcedefinition.apiextensions.k8s.io/rollouts.argoproj.io created
serviceaccount/argo-rollouts created
clusterrole.rbac.authorization.k8s.io/argo-rollouts created
clusterrole.rbac.authorization.k8s.io/argo-rollouts-aggregate-to-admin created
clusterrole.rbac.authorization.k8s.io/argo-rollouts-aggregate-to-edit created
clusterrole.rbac.authorization.k8s.io/argo-rollouts-aggregate-to-view created
clusterrolebinding.rbac.authorization.k8s.io/argo-rollouts created
configmap/argo-rollouts-config created
secret/argo-rollouts-notification-secret created
service/argo-rollouts-metrics created
deployment.apps/argo-rollouts created

# 检查部署
[root@k-m-1 ~]# kubectl get pod,svc -n argo-rollouts 
NAME                                 READY   STATUS    RESTARTS   AGE
pod/argo-rollouts-565d544c47-shvgc   1/1     Running   0          47s

NAME                            TYPE        CLUSTER-IP   EXTERNAL-IP   PORT(S)    AGE
service/argo-rollouts-metrics   ClusterIP   10.96.2.86   <none>        8090/TCP   48s

# 检查增加的新的 CRD 资源
[root@k-m-1 ~]# kubectl get crd | grep argoproj
analysisruns.argoproj.io                              2023-10-06T19:54:53Z
analysistemplates.argoproj.io                         2023-10-06T19:54:53Z
applications.argoproj.io                              2023-10-02T06:02:02Z
applicationsets.argoproj.io                           2023-10-02T06:02:02Z
appprojects.argoproj.io                               2023-10-02T06:02:03Z
clusteranalysistemplates.argoproj.io                  2023-10-06T19:54:53Z
experiments.argoproj.io                               2023-10-06T19:54:53Z
rollouts.argoproj.io                                  2023-10-06T19:54:54Z

# 安装插件
[root@k-m-1 ~]# wget https://ghproxy.com/https://github.com/argoproj/argo-rollouts/releases/download/v1.6.0/kubectl-argo-rollouts-linux-amd64
[root@k-m-1 ~]# chmod +x kubectl-argo-rollouts-linux-amd64 && mv kubectl-argo-rollouts-linux-amd64 /usr/local/bin/kubectl-argo-rollouts
[root@k-m-1 ~]# kubectl argo rollouts version
kubectl-argo-rollouts: v1.6.0+7eae71e
  BuildDate: 2023-09-06T18:36:42Z
  GitCommit: 7eae71ed89f1a3769864435bddebe3ca05384df3
  GitTreeState: clean
  GoVersion: go1.20.7
  Compiler: gc
  Platform: linux/amd64

# 金丝雀发布
# 接下来我们通过几个简单的示例来说明 Rollout 部署，升级，发布和中断等操作，以此来展示 Rollouts 的各种功能
```

#### 7.1：部署 Rollout

```shell
# 首先我们部署一个 Rollout 资源和一个针对该资源的 kuberentes Service 对象，这里我们示例中的 Rollout 采用了金丝雀的更新策略，将 20% 的流量发送到金丝雀上，然后手动发布。最后再升级的剩余时间内逐渐自动增加流量，可以通过如下所示的 Rollout 来描述这个策略
```

```yaml
# basic-rollout.yaml
apiVersion: argoproj.io/v1alpha1
kind: Rollout
metadata:
  name: rollouts-demo
spec:
  replicas: 5                    # 定义5个副本
  strategy:                      # 定义升级策略
    canary:                      # 金丝雀发布
      steps:                     # 发布的节奏
      - setWeight: 20
      - pause: {}                # 会一直暂停
      - setWeight: 40
      - pause: { duration: 10 }  # 暂停10s
      - setWeight: 60
      - pause: { duration: 10 }
      - setWeight: 80
      - pause: { duration: 10 }
  revisionHistoryLimit: 2        # 下面部分其实和 Deployment 兼容
  selector:
    matchLabels:
      app: rollouts-demo
  template:
    metadata:
      labels:
        app: rollouts-demo
    spec:
      containers:
      - name: rollouts-demo
        image: argoproj/rollouts-demo:blue
        ports:
        - name: http
          containerPort: 8080
          protocol: TCP
        resources:
          requests:
            memory: 32Mi
            cpu: 5m
```

```yaml
# basic-service.yaml
apiVersion: v1
kind: Service
metadata:
  name: rollouts-demo
spec:
  type: ClusterIP
  selector:
    app: rollouts-demo
  ports:
  - name: http
    port: 8080
    targetPort: http
    protocol: TCP
```

```yaml
# basic-ingress.yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: rollouts-demo
spec:
  ingressClassName: nginx
  rules:
  - host: blue.devops-engineer.com.cn
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: rollouts-demo
            port:
              name: http
```

```shell
# 部署资源
[root@k-m-1 argo-rollout]# kubectl apply -f .
rollout.argoproj.io/rollouts-demo created
service/rollouts-demo created
ingress.networking.k8s.io/rollouts-demo created

# 检查
[root@k-m-1 argo-rollout]# kubectl get pod,svc
NAME                                 READY   STATUS    RESTARTS   AGE
pod/rollouts-demo-687d76d795-6djqc   1/1     Running   0          95s
pod/rollouts-demo-687d76d795-76nk7   1/1     Running   0          95s
pod/rollouts-demo-687d76d795-r6s2f   1/1     Running   0          95s
pod/rollouts-demo-687d76d795-v5nbz   1/1     Running   0          95s
pod/rollouts-demo-687d76d795-w95p5   1/1     Running   0          95s

NAME                    TYPE        CLUSTER-IP    EXTERNAL-IP   PORT(S)    AGE
service/kubernetes      ClusterIP   10.96.0.1     <none>        443/TCP    19d
service/rollouts-demo   ClusterIP   10.96.1.213   <none>        8080/TCP   95s

# 我们这个时候可能会有一个疑问，为什么我定义的金丝雀没有生效？其实这个是正常的，因为是第一次部署，没有其他版本，所以就直接部署了5个副本，当我们基于这个 Rollout 再去更新的时候就开始走金丝雀策略了，然后我们使用命令行来看看是个什么情况

[root@k-m-1 argo-rollout]# kubectl argo rollouts get rollout rollouts-demo
Name:            rollouts-demo
Namespace:       default
Status:          ✔ Healthy			# 应用发布状态
Strategy:        Canary				# 应用发布策略
  Step:          8/8    			# 发布步骤
  SetWeight:     100    			# 权重
  ActualWeight:  100		     	# 真实权重
Images:          argoproj/rollouts-demo:blue (stable)  # 镜像
Replicas:
  Desired:       5
  Current:       5
  Updated:       5
  Ready:         5
  Available:     5

NAME                                       KIND        STATUS     AGE    INFO
⟳ rollouts-demo                            Rollout     ✔ Healthy  5m14s  
└──# revision:1       # 发布的版本
   └──⧉ rollouts-demo-687d76d795           ReplicaSet  ✔ Healthy  5m14s  stable
      ├──□ rollouts-demo-687d76d795-6djqc  Pod         ✔ Running  5m14s  ready:1/1
      ├──□ rollouts-demo-687d76d795-76nk7  Pod         ✔ Running  5m14s  ready:1/1
      ├──□ rollouts-demo-687d76d795-r6s2f  Pod         ✔ Running  5m14s  ready:1/1
      ├──□ rollouts-demo-687d76d795-v5nbz  Pod         ✔ Running  5m14s  ready:1/1
      └──□ rollouts-demo-687d76d795-w95p5  Pod         ✔ Running  5m14s  ready:1/1

# 我们访问一下这个 Ingress
```

![blue](https://picture.devops-engineer.com.cn/file/723c5db2964aa7062da09.jpg)

```shell
# 可以看到这个是，当我们访问这个应用的时候，所有的请求都是蓝色的

# Argo Rollouts 的 kubectl 插件允许我们可视化 Rollout 以及相关资源对象，并展示实时状态变化，要在部署过程中观察 Rollout，可以通过运行插件的 get rollout --watch 命令
[root@k-m-1 argo-rollout]# kubectl argo rollouts get rollout rollouts-demo --watch
```

#### 7.2：更新 Rollout

```shell
# 上面已经部署完成了一个 Rollout，接下里我们则执行更新，和 Deployment 类似，对 Pod 模板字段进行的任何变更都会导致新的版本（即 ReplicaSet）被部署，更新 Rollout 通常是修改容器镜像的版本，kubectl apply，为了方便，Rollout 插件还单独提供了一个 set image 命令，比如我们这里运行了如下命令，用 yellow 版本的容器更新上面的 Rollout
[root@k-m-1 argo-rollout]# kubectl argo rollouts set image rollouts-demo rollouts-demo=argoproj/rollouts-demo:yellow
rollout "rollouts-demo" image updated

# 在 Rollout 更新期间，控制器将通过 Rollout 更新策略中定义的步骤进行，这个示例的 Rollout 为金丝雀设置了 20% 的流量权重，并一直暂停 Rollout，直到用户取消或者促进发布，在更新镜像后，再次观察 Rollout 直到它达到暂停状态
[root@k-m-1 argo-rollout]# kubectl argo rollouts get rollout rollouts-demo
Name:            rollouts-demo
Namespace:       default
Status:          ॥ Paused           # 暂停状态
Message:         CanaryPauseStep
Strategy:        Canary
  Step:          1/8                # 步骤在第一个
  SetWeight:     20
  ActualWeight:  20
Images:          argoproj/rollouts-demo:blue (stable)    # 原始版本
                 argoproj/rollouts-demo:yellow (canary)  # 金丝雀版本
Replicas:
  Desired:       5
  Current:       5
  Updated:       1
  Ready:         5
  Available:     5

NAME                                       KIND        STATUS     AGE   INFO
⟳ rollouts-demo                            Rollout     ॥ Paused   8h    
├──# revision:2                                                         
│  └──⧉ rollouts-demo-6cf78c66c5           ReplicaSet  ✔ Healthy  114s  canary
│     └──□ rollouts-demo-6cf78c66c5-xwn8l  Pod         ✔ Running  114s  ready:1/1
└──# revision:1                                                         
   └──⧉ rollouts-demo-687d76d795           ReplicaSet  ✔ Healthy  8h    stable
      ├──□ rollouts-demo-687d76d795-6djqc  Pod         ✔ Running  8h    ready:1/1
      ├──□ rollouts-demo-687d76d795-r6s2f  Pod         ✔ Running  8h    ready:1/1
      ├──□ rollouts-demo-687d76d795-v5nbz  Pod         ✔ Running  8h    ready:1/1
      └──□ rollouts-demo-687d76d795-w95p5  Pod         ✔ Running  8h    ready:1/1

# 可以看到，现在有两个版本，但是第二个版本只有一个被发布，然后我们去观察一下流量
```

![argo-rollout](https://picture.devops-engineer.com.cn/file/4a0d3d83c571bedf0d204.jpg)

```shell
# 可以看到，只有很少一部分流量被放进来了，大概只有20%，这个是根据我们 step 定义的，这里我们按照正式环境，测试没问题了，那么我们希望继续发布，这个时候我们让应用继续发布
[root@k-m-1 argo-rollout]# kubectl argo rollouts promote rollouts-demo
rollout 'rollouts-demo' promoted

# 检查发布状态
[root@k-m-1 argo-rollout]# kubectl argo rollouts get rollout rollouts-demo
Name:            rollouts-demo
Namespace:       default
Status:          ✔ Healthy
Strategy:        Canary
  Step:          8/8
  SetWeight:     100
  ActualWeight:  100
Images:          argoproj/rollouts-demo:yellow (stable)
Replicas:
  Desired:       5
  Current:       5
  Updated:       5
  Ready:         5
  Available:     5

NAME                                       KIND        STATUS        AGE  INFO
⟳ rollouts-demo                            Rollout     ✔ Healthy     8h   
├──# revision:2                                                           
│  └──⧉ rollouts-demo-6cf78c66c5           ReplicaSet  ✔ Healthy     22m  stable
│     ├──□ rollouts-demo-6cf78c66c5-xwn8l  Pod         ✔ Running     22m  ready:1/1
│     ├──□ rollouts-demo-6cf78c66c5-fl54s  Pod         ✔ Running     41s  ready:1/1
│     ├──□ rollouts-demo-6cf78c66c5-nsw82  Pod         ✔ Running     31s  ready:1/1
│     ├──□ rollouts-demo-6cf78c66c5-z5n8d  Pod         ✔ Running     21s  ready:1/1
│     └──□ rollouts-demo-6cf78c66c5-lf9g6  Pod         ✔ Running     11s  ready:1/1
└──# revision:1                                                           
   └──⧉ rollouts-demo-687d76d795           ReplicaSet  • ScaledDown  8h   

# 可以发现，现在的流量全部发布到新版本了，然后我们 Rollout 定义的是相隔 10s，看到发布的 AGE 的确是和我们定义的一样的
```

![argo-rollout](https://picture.devops-engineer.com.cn/file/1c80554d29d3f2bb48f52.jpg)

```shell
# 再看流量这个时候我们的流量全部已经到了新版本了
# 注意：promote 命令支持 --full 参数跳过所有剩余步骤和分析
```

#### 7.3：中断 Rollout

```shell
# 解析来我们来了解如何在更新的过程中手动终止 Rollout，首先，使用 set image 命令部署一个新的 red 版本的应用，并等待 Rollout 再次达到暂停的步骤
[root@k-m-1 argo-rollout]# kubectl argo rollouts set image rollouts-demo rollouts-demo=argoproj/rollouts-demo:red
rollout "rollouts-demo" image updated

# 检查部署
[root@k-m-1 argo-rollout]# kubectl argo rollouts get rollout rollouts-demo
Name:            rollouts-demo
Namespace:       default
Status:          ॥ Paused
Message:         CanaryPauseStep
Strategy:        Canary
  Step:          1/8
  SetWeight:     20
  ActualWeight:  20
Images:          argoproj/rollouts-demo:red (canary)
                 argoproj/rollouts-demo:yellow (stable)
Replicas:
  Desired:       5
  Current:       5
  Updated:       1
  Ready:         5
  Available:     5

NAME                                       KIND        STATUS        AGE  INFO
⟳ rollouts-demo                            Rollout     ॥ Paused      8h   
├──# revision:3                                                           
│  └──⧉ rollouts-demo-5747959bdb           ReplicaSet  ✔ Healthy     17s  canary
│     └──□ rollouts-demo-5747959bdb-4k4t6  Pod         ✔ Running     17s  ready:1/1
├──# revision:2                                                           
│  └──⧉ rollouts-demo-6cf78c66c5           ReplicaSet  ✔ Healthy     34m  stable
│     ├──□ rollouts-demo-6cf78c66c5-xwn8l  Pod         ✔ Running     34m  ready:1/1
│     ├──□ rollouts-demo-6cf78c66c5-fl54s  Pod         ✔ Running     12m  ready:1/1
│     ├──□ rollouts-demo-6cf78c66c5-nsw82  Pod         ✔ Running     12m  ready:1/1
│     └──□ rollouts-demo-6cf78c66c5-lf9g6  Pod         ✔ Running     12m  ready:1/1
└──# revision:1                                                           
   └──⧉ rollouts-demo-687d76d795           ReplicaSet  • ScaledDown  8h   
   
# 访问看看流量
```

![argo-rollout](https://picture.devops-engineer.com.cn/file/0ba86c7976d6dd64c46e0.jpg)

```shell
# 那么它模拟的就是故障的应用，我们看到了故障的应用，肯定要中断这次发布的，我们该如何中断这次发布呢？
[root@k-m-1 argo-rollout]# kubectl argo rollouts abort rollouts-demo
rollout 'rollouts-demo' aborted

# 查看中断状态
[root@k-m-1 argo-rollout]# kubectl argo rollouts get rollout rollouts-demo
Name:            rollouts-demo
Namespace:       default
Status:          ✖ Degraded
Message:         RolloutAborted: Rollout aborted update to revision 3
Strategy:        Canary
  Step:          0/8
  SetWeight:     0
  ActualWeight:  0
Images:          argoproj/rollouts-demo:yellow (stable)
Replicas:
  Desired:       5
  Current:       5
  Updated:       0
  Ready:         5
  Available:     5

NAME                                       KIND        STATUS        AGE   INFO
⟳ rollouts-demo                            Rollout     ✖ Degraded    8h    
├──# revision:3                                                            
│  └──⧉ rollouts-demo-5747959bdb           ReplicaSet  • ScaledDown  3m7s  canary
├──# revision:2                                                            
│  └──⧉ rollouts-demo-6cf78c66c5           ReplicaSet  ✔ Healthy     37m   stable
│     ├──□ rollouts-demo-6cf78c66c5-xwn8l  Pod         ✔ Running     37m   ready:1/1
│     ├──□ rollouts-demo-6cf78c66c5-fl54s  Pod         ✔ Running     15m   ready:1/1
│     ├──□ rollouts-demo-6cf78c66c5-nsw82  Pod         ✔ Running     15m   ready:1/1
│     ├──□ rollouts-demo-6cf78c66c5-lf9g6  Pod         ✔ Running     15m   ready:1/1
│     └──□ rollouts-demo-6cf78c66c5-nkt86  Pod         ✔ Running     15s   ready:1/1
└──# revision:1                                                            
   └──⧉ rollouts-demo-687d76d795           ReplicaSet  • ScaledDown  8h    

# 可以看到 revision:2 的副本又被扩容回来了，那么这个时候的流量肯定又全部到了这个版本，但是切记，revision:3 这个版本的 replicaset 还是存在的，只是将副本缩容为 0 了
```

![argo-rollout](https://picture.devops-engineer.com.cn/file/9b31437f8863b1e9bfea4.jpg)

```shell
# 为了使 Rollout 再次被认为是健康而不是有问题的版本，有必要将所需的状态改回以前的稳定版本，在我们的例子中，我们可以简单的使用之前的 yellow 镜像重新运行 set image 命令即可
[root@k-m-1 argo-rollout]# kubectl argo rollouts set image rollouts-demo rollouts-demo=argoproj/rollouts-demo:yellow
rollout "rollouts-demo" image updated

# 再次查看
[root@k-m-1 argo-rollout]# kubectl argo rollouts get rollout rollouts-demo
Name:            rollouts-demo
Namespace:       default
Status:          ✔ Healthy
Strategy:        Canary
  Step:          8/8
  SetWeight:     100
  ActualWeight:  100
Images:          argoproj/rollouts-demo:yellow (stable)
Replicas:
  Desired:       5
  Current:       5
  Updated:       5
  Ready:         5
  Available:     5

NAME                                       KIND        STATUS        AGE  INFO
⟳ rollouts-demo                            Rollout     ✔ Healthy     9h   
├──# revision:4                                                           
│  └──⧉ rollouts-demo-6cf78c66c5           ReplicaSet  ✔ Healthy     97m  stable
│     ├──□ rollouts-demo-6cf78c66c5-xwn8l  Pod         ✔ Running     97m  ready:1/1
│     ├──□ rollouts-demo-6cf78c66c5-fl54s  Pod         ✔ Running     75m  ready:1/1
│     ├──□ rollouts-demo-6cf78c66c5-nsw82  Pod         ✔ Running     75m  ready:1/1
│     ├──□ rollouts-demo-6cf78c66c5-lf9g6  Pod         ✔ Running     75m  ready:1/1
│     └──□ rollouts-demo-6cf78c66c5-nkt86  Pod         ✔ Running     60m  ready:1/1
├──# revision:3                                                           
│  └──⧉ rollouts-demo-5747959bdb           ReplicaSet  • ScaledDown  63m  
└──# revision:1                                                           
   └──⧉ rollouts-demo-687d76d795           ReplicaSet  • ScaledDown  9h   

# 这样就回滚回来了，那么前面我们也讲了，Argo Rollouts 可以与 Ingress 控制器集成，实现用户级的灰度，金丝雀等策略，那么我们下面就来集成看看效果
```

#### 7.4：Argo Rollouts + Ingress-nginx

```shell
# 对于上面的例子，我们会发现没有针对具体的流量进行控制，它使用的就是最简单的 Service 来实现近似金丝雀权重，基于新旧副本数量的比例来实现，所以，这个 Rollout 有限制，为了实现更细粒度的金丝雀，所以我们就要集成 Ingress 控制器或服务网格了

# 支持哪儿些控制器，下面是链接
# URL：https://argoproj.github.io/argo-rollouts/features/traffic-management/

# 检查 Ingress 控制器（我这里使用的就是官方的 Ingress-nginx）
[root@k-m-1 argo-rollout]# kubectl get pod -n ingress-nginx 
NAME                                   READY   STATUS      RESTARTS      AGE
ingress-nginx-admission-create-djlbx   0/1     Completed   0             5d
ingress-nginx-admission-patch-vmhxq    0/1     Completed   0             5d
ingress-nginx-controller-wh9mj         1/1     Running     0             5d

# 接下来我们就该考虑如何将 Argo Rollouts 与 Ingress-nginx 集成进行流量控制了，Rollout 的配置必须有如下几个字段
```

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Rollout
metadata:
  name: rollouts-demo
spec:
  strategy:
    canary:
      # 引用一个 Service，控制器将更新该服务以指向金丝雀 ReplicaSet
      canartService: rollouts-demo-canary
      # 引用一个 Service，控制器将更新该服务以指向稳定的 ReplicaSet
      stableService: rollouts-demo-stable
      trafficRouting:
        nginx:
          # 指向稳定 Service 的规则所引用的 Ingress
          # 该 Ingress 将被克隆并赋予一个新的名称，以实现 NGINX 流量分割
          stableIngress: rollouts-demo-stable
```

```shell
# 其中 canary.trafficRouting.nginx.stableIngress 中引用的 Ingress 需要有一个 host 规则，该规则具有针对 canary.stableService 下引用服务的后端，接下里我们来完善我们的资源清单文件
```

```yaml
# rollout.yaml
apiVersion: argoproj.io/v1alpha1
kind: Rollout
metadata:
  name: rollouts-demo
spec:
  replicas: 1
  strategy:
    canary:
      canaryService: rollouts-demo-canary
      stableService: rollouts-demo-stable
      trafficRouting:
        nginx:
          stableIngress: rollouts-demo-stable
      steps:
      - setWeight: 5
      - pause: {}
  revisionHistoryLimit: 2
  selector:
    matchLabels:
      app: rollouts-demo
  template:
    metadata:
      labels:
        app: rollouts-demo
    spec:
      containers:
      - name: rollouts-demo
        image: argoproj/rollouts-demo:blue
        ports:
        - name: http
          containerPort: 8080
          protocol: TCP
        resources:
          requests:
            memory: 32Mi
            cpu: 5m
```

```shell
# 上面资源清单中，我们定义了一个 rollouts-demo 的 Rollout 资源，它的 canaryService 和 stableService 分别引用了两个 Service 资源，stableIngress 引用了一个 Ingress 资源，steps 定义了金丝雀发布的步骤，这里我们定义了两个步骤，一个步骤将权重设置为 5%，第二个步骤是暂停，这样就可以在第一个步骤中将 5% 的流量发送到金丝雀上，然后手动发布，最后在升级的剩余时间内逐渐自动增大流量，对应的 Service 如下
```

```yaml
# service.yaml
apiVersion: v1
kind: Service
metadata:
  name: rollouts-demo-canary
spec:
  type: ClusterIP
  # 该 selector 将使用金丝雀 ReplicaSet 的 pod-template-hash 进行更新，比如 rollouts-pod-template-hash: xxxxx21
  selector:
    app: rollouts-demo
  ports:
  - name: http
    port: 8080
    targetPort: http
    protocol: TCP
---
apiVersion: v1
kind: Service
metadata:
  name: rollouts-demo-stable
spec:
  type: ClusterIP
  # 该 selector 将使用稳定版 ReplicaSet 的 pod-template-hash 进行更新，比如 rollouts-pod-template-hash: xxxxx12
  selector:
    app: rollouts-demo
  ports:
  - name: http
    port: 8080
    targetPort: http
    protocol: TCP
```

```shell
# 最后还有一个稳定版的 stable 的 Ingress 对象
```

```yaml
# ingress.yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: rollouts-demo-stable
spec:
  ingressClassName: nginx
  rules:
  - host: rollout.devops-engineer.com.cn
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            # 引用服务名称，也在 Rollout spec.strategy.cancry.stableService 字段中指定
            name: rollouts-demo-stable
            port:
              name: http
```

```shell
# 删除最开始我们测试用的对象，然后部署新的资源对象
[root@k-m-1 argo-rollout]# kubectl apply -f .
ingress.networking.k8s.io/rollouts-demo-stable created
rollout.argoproj.io/rollouts-demo created
service/rollouts-demo-canary created
service/rollouts-demo-stable created

# 检查资源
[root@k-m-1 argo-rollout]# kubectl get pod,svc,ingress
NAME                                 READY   STATUS    RESTARTS   AGE
pod/rollouts-demo-687d76d795-jv6rf   1/1     Running   0          109s

NAME                           TYPE        CLUSTER-IP    EXTERNAL-IP   PORT(S)    AGE
service/kubernetes             ClusterIP   10.96.0.1     <none>        443/TCP    20d
service/rollouts-demo-canary   ClusterIP   10.96.1.180   <none>        8080/TCP   2m2s
service/rollouts-demo-stable   ClusterIP   10.96.0.197   <none>        8080/TCP   2m2s

# 可以看到这里有俩 Ingress，而新增的这个就是用于金丝雀的发布对象，我们下面对比一下这两个对象的区别
NAME                                                                  CLASS   HOSTS                            ADDRESS     PORTS   AGE
ingress.networking.k8s.io/rollouts-demo-rollouts-demo-stable-canary   nginx   rollout.devops-engineer.com.cn   10.0.0.11   80      109s
ingress.networking.k8s.io/rollouts-demo-stable                        nginx   rollout.devops-engineer.com.cn   10.0.0.11   80      2m2s

[root@k-m-1 argo-rollout]# kubectl get ingress rollouts-demo-stable -o yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: rollouts-demo-stable
  namespace: default
spec:
  ingressClassName: nginx
  rules:
  - host: rollout.devops-engineer.com.cn
    http:
      paths:
      - backend:
          service:
            name: rollouts-demo-stable
            port:
              name: http
        path: /
        pathType: Prefix

[root@k-m-1 argo-rollout]# kubectl get ingress rollouts-demo-rollouts-demo-stable-canary -oyaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  annotations:
    nginx.ingress.kubernetes.io/canary: "true"
    nginx.ingress.kubernetes.io/canary-weight: "0"
  name: rollouts-demo-rollouts-demo-stable-canary
  namespace: default
  ownerReferences:
  - apiVersion: argoproj.io/v1alpha1
    blockOwnerDeletion: true
    controller: true
    kind: Rollout
    name: rollouts-demo
    uid: 37c67bf2-bf79-4422-9b96-27155525b952
spec:
  ingressClassName: nginx
  rules:
  - host: rollout.devops-engineer.com.cn
    http:
      paths:
      - backend:
          service:
            name: rollouts-demo-canary
            port:
              name: http
        path: /
        pathType: Prefix

# 能看到的就是金丝雀版本的 Ingress 对象内多了针对 ingress-nginx 的金丝雀发布的注解，其次说明了这个金丝雀的对象被 Rollout 所管理，完成之后，我们就可以开始做一次发布，我们可以看看发布前的情况
```

![argo-rollout](https://picture.devops-engineer.com.cn/file/4ff53835b034a897438a0.jpg)

```shell
# 查看当前的 rollout 的情况
[root@k-m-1 argo-rollout]# kubectl argo rollouts get rollout rollouts-demo
Name:            rollouts-demo
Namespace:       default
Status:          ✔ Healthy
Strategy:        Canary
  Step:          2/2
  SetWeight:     100
  ActualWeight:  100
Images:          argoproj/rollouts-demo:blue (stable)
Replicas:
  Desired:       1
  Current:       1
  Updated:       1
  Ready:         1
  Available:     1

NAME                                       KIND        STATUS     AGE  INFO
⟳ rollouts-demo                            Rollout     ✔ Healthy  12m  
└──# revision:1                                                        
   └──⧉ rollouts-demo-687d76d795           ReplicaSet  ✔ Healthy  12m  stable
      └──□ rollouts-demo-687d76d795-jv6rf  Pod         ✔ Running  12m  ready:1/1

# 发布
[root@k-m-1 argo-rollout]# kubectl argo rollouts set image rollouts-demo rollouts-demo=argoproj/rollouts-demo:yellow
rollout "rollouts-demo" image updated

# 查看发布情况
[root@k-m-1 argo-rollout]# kubectl argo rollouts get rollout rollouts-demo
Name:            rollouts-demo
Namespace:       default
Status:          ॥ Paused
Message:         CanaryPauseStep
Strategy:        Canary
  Step:          1/2
  SetWeight:     5
  ActualWeight:  5
Images:          argoproj/rollouts-demo:blue (stable)
                 argoproj/rollouts-demo:yellow (canary)
Replicas:
  Desired:       1
  Current:       2
  Updated:       1
  Ready:         2
  Available:     2

NAME                                       KIND        STATUS     AGE  INFO
⟳ rollouts-demo                            Rollout     ॥ Paused   13m  
├──# revision:2                                                        
│  └──⧉ rollouts-demo-6cf78c66c5           ReplicaSet  ✔ Healthy  20s  canary
│     └──□ rollouts-demo-6cf78c66c5-7dzmg  Pod         ✔ Running  20s  ready:1/1
└──# revision:1                                                        
   └──⧉ rollouts-demo-687d76d795           ReplicaSet  ✔ Healthy  13m  stable
      └──□ rollouts-demo-687d76d795-jv6rf  Pod         ✔ Running  13m  ready:1/1
      
# 我们请求一下看看流量情况
```

![argo-rollout](https://picture.devops-engineer.com.cn/file/def0f0c397f580e30e3df.jpg)

```shell
# 看起来只有一点点的金丝雀版本流量，因为只有 5% 嘛。所以就非常少，具体可以看 Ingress 的 canary 的信息

apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  annotations:
    nginx.ingress.kubernetes.io/canary: "true"
    nginx.ingress.kubernetes.io/canary-weight: "5"
  name: rollouts-demo-rollouts-demo-stable-canary
  namespace: default
  ownerReferences:
  - apiVersion: argoproj.io/v1alpha1
    blockOwnerDeletion: true
    controller: true
    kind: Rollout
    name: rollouts-demo
    uid: 37c67bf2-bf79-4422-9b96-27155525b952
spec:
  ingressClassName: nginx
  rules:
  - host: rollout.devops-engineer.com.cn
    http:
      paths:
      - backend:
          service:
            name: rollouts-demo-canary
            port:
              name: http
        path: /
        pathType: Prefix
        
# 可以看到 canary-weight 被设置到了 5，也就说明了会有 5% 的流量流向 rollouts-demo-canary 这个 Service，最后我们就得到了上图的结果，然后我们想全部发布的话就可以直接按照上面讲的，直接继续流程发布就可以了，当然中断也是可以的

[root@k-m-1 argo-rollout]# kubectl argo rollouts promote rollouts-demo
rollout 'rollouts-demo' promoted

# 检查发布情况
[root@k-m-1 argo-rollout]# kubectl argo rollouts get rollout rollouts-demo
Name:            rollouts-demo
Namespace:       default
Status:          ✔ Healthy
Strategy:        Canary
  Step:          2/2
  SetWeight:     100
  ActualWeight:  100
Images:          argoproj/rollouts-demo:yellow (stable)
Replicas:
  Desired:       1
  Current:       1
  Updated:       1
  Ready:         1
  Available:     1

NAME                                       KIND        STATUS        AGE   INFO
⟳ rollouts-demo                            Rollout     ✔ Healthy     162m  
├──# revision:2                                                            
│  └──⧉ rollouts-demo-6cf78c66c5           ReplicaSet  ✔ Healthy     149m  stable
│     └──□ rollouts-demo-6cf78c66c5-7dzmg  Pod         ✔ Running     149m  ready:1/1
└──# revision:1                                                            
   └──⧉ rollouts-demo-687d76d795           ReplicaSet  • ScaledDown  162m  

# 访问发布应用
```

![argo-rollout](https://picture.devops-engineer.com.cn/file/91f30c26f658f5997f854.jpg)

```shell
# 可以看到，现在应用就全部发布到了金丝雀的版本，现在这个版本就成为了 stable 了，然后我们再去看看 Ingress
[root@k-m-1 argo-rollout]# kubectl get pod,svc,ingress
NAME                                 READY   STATUS    RESTARTS   AGE
pod/rollouts-demo-6cf78c66c5-7dzmg   1/1     Running   0          151m

NAME                           TYPE        CLUSTER-IP    EXTERNAL-IP   PORT(S)    AGE
service/kubernetes             ClusterIP   10.96.0.1     <none>        443/TCP    20d
service/rollouts-demo-canary   ClusterIP   10.96.1.180   <none>        8080/TCP   165m
service/rollouts-demo-stable   ClusterIP   10.96.0.197   <none>        8080/TCP   165m

NAME                                                                  CLASS   HOSTS                            ADDRESS     PORTS   AGE
ingress.networking.k8s.io/rollouts-demo-rollouts-demo-stable-canary   nginx   rollout.devops-engineer.com.cn   10.0.0.11   80      165m
ingress.networking.k8s.io/rollouts-demo-stable                        nginx   rollout.devops-engineer.com.cn   10.0.0.11   80      165m

# 那么其实这个时候的 Ingress 的金丝雀的那个版本的权重已经被降为 0 了，那么这个其实就是 Rollout 的一个基本的功能了
```

#### 7.5：Argo Rollouts Dashboard

```shell
# Argo Rollouts kubectl 插件可以提供一个本地 Dashboard，来可视化我们的 Rollouts
# 要启动这个 Dashboard，需要在包含 Rollouts 资源对象的命名空间中运行 kubectl argo rollouts dashboard 命令，然后访问 localhost:3100 即可

[root@k-m-1 ~]# kubectl argo rollouts dashboard
INFO[0000] Argo Rollouts Dashboard is now available at http://localhost:3100/rollouts 

```

![](https://picture.devops-engineer.com.cn/file/e439e3293dd534108ccb1.jpg)

![argo-rollout](https://picture.devops-engineer.com.cn/file/fe7d63ece3a725ab47259.jpg)

```shell
# 在这里其实可以发现，主要就是针对 Rollout 的操作和可视化做了 UI 展示
```

#### 7.6：Analysis 和 渐进式交互

```shell
# Argo Rollouts 提供了几种执行分析的方法来推动渐进式交付，首先需要了解几个 CRD 资源
1：Rollout：Rollout 是 Deployment 资源的直接替代方案，它提供额外的 blueGreen 和 canary 更新策略，这些策略可以在更新期间创建 AnalysisRuns 和 Experiments，可以推进更新或中止更新
2：Experiments：Experiments CRD 允许用户临时运行 一个 或 多个ReplicaSet，除了运行临时 ReplicaSet 之外，Experiments CRD 还可以与 ReplicaSet 一起启动 AnalysisRuns，通常，这些 AnalysisRun 用于确认新的 ReplicaSet 是否按预期运行
3：AnalysisTemplate：AnalysisTemplate 是一个模板，它定义了如何执行金丝雀分析，例如它应该执行的指标，频率以及被视为成功或失败的值，AnalysisTemplate 可以用输入值机型参数化
4：ClusterAnalysisTemplate：ClusterAnalysisTemplate 和 AnalysisTemplate 类似，但它是全局范围内的，它可以被整个集群的任何 Rollout 使用
5：AnalysisRun：AnalysisRun 是 AnalysisTemplate 的实例化，AnalysisRun 就像一个 Job 一样，它最终会完成，完成的运行被认为是成功的，失败的或不确定的，运行的结果分别影响 Rollout 的更新是否继续，终止或暂停

# 后台分析
# 金丝雀正在执行其部署步骤时，分析可以在后台运行
# 以下示例时每 10 分钟逐渐讲 Canary 权重增加到 100%，直到达到 100%，在后台，基于名为 success-rate 的 AnalysysTemplate 启动 AnalysisRun，success-rate 模板查询 Prometheus 服务器，以 5 分钟间隔/样本测量HTTP成功率，它没有结束时间，一直持续到停止或失败，如果测量的指标小于 95%，并且有 3 个这样的测量值，则分析被视为失败，失败的分析会导致 Rollout 中止，将 Canary 权重设置回 0，并且 Rollout 将被视为降级，否则，如果 Rollout 完成其所有 Canary 步骤，则认为 Rollout 时成功的，并且控制器将停止运行分析
```

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Rollout
metadata:
  name: guestbook\
spec:
  # ...
  strategy:
    canary:
      analysis:
        templates:
        - templateName: success-rate
        startingStep: 2  # 延迟开始分析，到第三步开始
        args:
        - name: service-name
          value: guestbook-svc.default.svc.cluster.local
      steps:
      - setWeight: 20
      - pause: { duration: 10m }
      - setWeight: 40
      - pause: { duration: 10m }
      - setWeight: 60
      - pause: { duration: 10m }
      - setWeight: 80
      - pause: { duration: 10m }
---
apiVersion: argoproj.io/v1alpha1
kind: AnalysisTemplate
metadata:
  name: success-rate
spec:
  args:
  - name: service-name
  metrics:
  - name: success-rate
    interval: 5m
    successCondition: result[0] >= 0.95
    failureLimit: 3
    provider:
      prometheus:
        address: http://prometheus.default.svc.cluster.local:9090
        query: |
          sum(irate(
            istio_requests_total{reporter="source", destination_service=~"{{args.service-name}}", response_code!~"5.*"}[5m]
          )) /
          sum(irate(
            istio_requests_total{reporter="source", destination_service=~"{{args.service-name}}"}[5m]
          ))
```

```shell
# 内联分析
# 分析也可以作为内嵌分析步骤来执行，当分析以内联方式进行时，在到达该步骤时启动 AnalysisRun，并在运行完成之前阻止其推进，分析运行的成功或失败决定了部署是否继续还是中止
# 如下所示的示例中我们将 Canary 权重设置为 20%，暂停 5 分钟，然后运行分析，如果分析成功，则继续推出，否则中止	
```

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Rollout
metadata:
  name: guestbook
spec:
  # ...
  strategy:
    canary:
      steps:
      - setWeight: 20
      - pause: { duration: 5m }
      - analysis:
          templates:
          - templateName: success-rate
          args:
          - name: service-name
            value: guestbook-svc.default.svc.cluster.local
```

```shell
# 上面的对象我们将 analysis 作为一个步骤内联到了 Rollout 步骤中，当 20% 流量暂停了 5 分钟后，开始执行 success-rate 这个分析模板
# 这里 AnalysisTemplate 与上面的后台分析例子相同，但由于没有指定间隔时间，分析将执行一次测量就完成了
```

```yaml
apiVersion: argoproj.io/v1alpha1
kind: AnalysisTemplate
metadata:
  name: success-rate
spec:
  args:
  - name: service-name
  - name: prometheus-port
    value: 9090
  metrics:
  - name: success-rate
    successCondition: result[0] >= 0.95
    provider:
      prometheus:
        address: http://prometheus.default.svc.cluster.local:{{args.prometheus-port}}
        query: |
          sum(irate(
            istio_requests_total{reporter="source", destination_service=~"{{args.service-name}}", response_code!~"5.*"}[5m]
          )) /
          sum(irate(
            istio_requests_total{reporter="source", destination_service=~"{{args.service-name}}"}[5m]
          ))
```

```shell
# 此外我们可以通过指定 count 和 interval 字段，可以在一个较长的时间段内进行多次调整
```

```yaml
metrics:
- name: success-rate
  successCondition: result[0] >= 0.95
  interval: 60s
  count: 5
  provider:
    prometheus:
      address: http://prometheus.default.svc.cluster.local:9090
      query: ...
```

```shell
# 多个模板分析
# Rollout 在构建 AnalysisRun 时可以引用多个 AnalysisTemplate，这样我们就可以从多个 AnalysisTemplate 中来组成分析，如果引用了多个模板，那么控制器将把这些模板合并在一起，控制器会结合所有模板的指标和args字段
```

```yaml
apiVersion: argoproj.io/v1alpha1
kind: AnalysisTemplate
metadata:
  name: argo-example
spec:
  args:
  # required
  - name: service-name
  - name: stable-hash
  - name: latest-hash
  # optional
  - name: api-url
    value: http://example/measure
  # from secret
  - name: api-token
    valueFrom:
      secretKeyRef:
        name: token-secret
        key: apiToken
  metrics:
  - name: webmetric
    successCondition: result == 'true'
    provider:
      web:
        url: "{{ args.api-url }}?service={{ args.service-name }}"
        headers:
        - key: Authorization
          value: "Bearer {{ args.api-token }}"
        jsonPath: "{{$.result.ok}}"
```

```shell
# 在创建 AnalysisRun 时，Rollout 中定义的参数与 AnalysisTemplate 的参数合并，如下
```

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Rollout
metadata:
  name: guestbook
spec:
  strategy:
    canary:
      analysis:
        templates:
        - templateName: args-example
        args:
        # required value
        - name: service-name
          value: guestbook-svc.default.svc.cluster.local
        # override default value
        - name: api-url
          value: http://other-apo
        # pod template hash from the stable ReplicaSet
        - name: stable-hash
          valueFrom:
            podTemplateHashValue: Stable
        # pod template hash from the latest ReplicaSet
        - name: latest-hash
          valueFrom:
            podTemplateHashValue: Latest
```

```shell
# 此外，分析参数也支持 valueFrom，用于读取集群 meta 数据并将其作为参数传递给 AnalysisTemplate，如下例子是引用元数据中的 env 和 region 标签，并将它们传递给 AnalysisTemplate。
```

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Rollout
metadata:
  name: guestbook
  labels:
    appType: demo-app
    buildType: nginx-app
    ...
    env: dev
    region: us-west-2
spec:
...
  strategy:
    canary:
      analysis:
        templates:
        - templateName: args-example
        args:
        ...
        - name: env
          valueFrom:
            fieldRef:
              fieldPath: metadata.labels['env']
        - name: region
          valueFrom:
            fieldRef:
              fieldPath: metadata.labels['region']
```

```shell
# 蓝绿预发布分析
# 使用 blueGreen 策略的 Rollout 可以在使用预发布将流量切换到新版本之前启动一个 AnalysisRun，分析运行的成功或失败决定 Rollout 是否切换流量，或完全中止 Rollout，如下所示
```

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Rollout
metadata:
  name: guestbook
spec:
# ...
  strategy:
    blueGreen:
      activeService: active-svc
      perviewService: perview-svc
      prePromotionAnalysis:
        templates:
        - templateName: smoke-tests
        args:
        - name: service-name
          value: preview-svc.default.svc.cluster.local
```

```shell
# 上面我们的示例中一旦新的 ReplicaSet 完全可用，Rollout 会创建一个预发布的 AnalysisRun，Rollout 不会讲流量切换到新版本，而是会等到分析运行成功完成

# 注意：如果制定了，autoPromotionSeconds 字段，并且 Rollout 已经等待了，auto promotion seconds 的时间，Rollout 会标记 AnalysisRun 成功，并且自动将流量切换到新版本，如果 AnalysisRun 在此之前完成，Rollout 将不会创建另一个 AnalysisRun，并等待 autoPromotionSeconds 的剩余时间

# 蓝绿发布后分析
# 使用 BlueGreen 策略的 Rollout 还可以在流量切换到新版本后使用发布后分析，如果发布后分析失败或出错，Rollout 会进入中止状态，并将流量切换回之前稳定的 ReplicaSet，当后分析成功时，Rollout 被认为是完全发布状态，新的 ReplicaSet 将被标记为稳定，然后旧的 ReplicaSet 将根据 scaleDownDelaySeconds（默认30秒）进行缩减
```

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Rollout
metadata:
  name: guestbook
spec:
# ...
  strategy:
    blueGreen:
      activeService: active-svc
      previewService: preview-svc
      scaleDownDelaySeconds: 600   # 10 分钟
      postPromotionAnalysis:
        templates:
        - templateName: smoke-tests
        args:
        - name: service-name
          value: preview-svc.default.svc.cluster.local
```

```shell
# 失败条件
# fialureCondition 可以用来配置分析运行失败，下面的例子是每隔 5 分钟持续轮询 Prometheus 服务器来获取错误总数，如果遇到 10 个或更多的错误，则认为分析运行失败
```

```yaml
metrics:
- name: total-errors
  interval: 5m
  failureConditiion: result[0] >= 10
  failureLimit: 3
  provider:
    prometheus:
      address: http://prometheus.default.svc.cluster.local:9090
      query: |
        sum(irate(
          istio_requests_total{reporter="source", destination_service=~"{{args.service-name}}", response_code~"5.*"}[5m]
        ))
```

```shell
# 无结果的运行
# 分析运行结果也可以是被认为是不确定的，这表明运行即不成功，也不失败，无结果的运行会导致发布在当前步骤上暂停，这时需要人工干预，以恢复运行，或者中止运行，当一个指标没有定义成功或失败的条件时，分析运行可能成为无结果的一个例子
```

```yaml
metrics:
- name: query
  provider:
    prometheus:
      address: http://prometheus.default.svc.cluster.local:9090
      query: ...
```

```shell
# 此外当同时指定了成功和失败的条件，但测量值没有满足任何一个条件时，也可能发生不确定的分析运行
```

```yaml
metrics:
- name: success-rate
  successCondition: result[0] >= 0.90
  failureCondition: result[0] < 0.50
  provider:
    prometheus:
      address: http://prometheus.default.svc.cluster.local:9090
      query: ...
```

```shell
# 不确定的分析运行的一个场景是使 Argo Rollout 能够自动执行分析运行，并收集测量结果，但仍然允许我们来判断决定测量值是否可以接受，并决定继续或中止

# 延迟分析运行
# 如果分析运行不需要立即开始（即给指标提供者时间来收集金丝雀版本的指标），分析运行可以延迟特定的指标分析，每个指标可以被配置有不同的延迟，除了特定指标的延迟之外，具有后台分析的发布可以延迟创建分析运行，直到达到某个步骤为止
```

```yaml
metrics:
- name: success-rate
  initialDelay: 5m
  successCondition: result[0] >= 0.90
  provider:
    prometheus:
      address: http://prometheus.default.svc.cluster.local:9090
      query: ...
```

```shell
# 延迟开始后台分析运行，直到步骤 3 （设定重量40%）
```

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Rollout
metadata:
  name: guestbook
spec:
  strategy:
    canary:
      analysis:
        templates:
        - templateName: success-rate
        startingStep: 2  
      steps:
      - setWeight: 20
      - pause: { duration: 10m }
      - setWeight: 40
      - pause: { duration: 10m }
```

```shell
# Job Metrics
# 此外 Kubernetes job 还可用于运行分析，使用 Job 时，如果 Job 完成且退出状态码为 0，则指标被视为成功，否则指标失败
```

```yaml
metrics:
- name: test
  provider:
    job:
      metadata:
        annotations:
          foo: bar
        labels:
          foo: bar
      spec:
        backoffLimit: 1
        template:
          spec:
            containers:
            - name: test
              image: my-image:latest
              command: [my-test-script, my-service.default.svc.cluster.local]
            restartPolicy: Never
```

```shell
# Web Metrics
# 同样还可以针对某些外部服务执行 HTTP 请求来获取测量结果，下面是向某个 URL 发送 HTTP GET 请求，WebHook 响应必须返回 JSON 内容，jsonPath 表达式的结果将分配给可在 successCondition 和 failureCondition 表达式中引用的 result 变量，如果省略，将使用整个 body 作为结果变量
```

```yaml
metrics:
- name: webmetric
  successCondition: result == true
  provider:
    web:
      url: "http://my-server.com/api/v1/measurement?service={{ args.service-name }}"
      timeoutSeconds: 20
      headers:
      - key: Authorization
        value: "Bearer {{ args.api-token }}"
      jsonPath: "{$.data.ok}"
```

```shell
# 比如下面的示例表示结果 data.ok 字段为真且 data.successPercent 大于 0.90，测量将是成功的
```

```yaml
{
  "data": {
    "ok": true,
    "successPercent": 0.95
  }
}
```

```yaml
metrics:
- name: webmetric
  successCondition: "result.ok && result.successPercent >= 0.90"
  provider:
    web:
      url: "http://my-server.com/api/v1/measurement?service={{ args.service-name }}"
      headers:
      - key: Authorization
        value: "Bearer {{ args.api-token }}"
      jsonPath: "{$.data}"
```

```shell
# 当然了关于 Argo Rollouts 的更多使用的系列，可以参考官网：https://argoproj.github.io/argo-rollouts
```

