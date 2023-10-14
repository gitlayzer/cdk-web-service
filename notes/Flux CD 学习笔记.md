## Flux CD 学习笔记

### 1：`Flux` 是什么

```shell
# Flux 是一套针对 Kubernetes 的持续交付和渐进式交付的解决方案，可以让我们以 GitOps 的方式轻松的交付应用，它和 Argo CD 不同，Flux CD 是许多工具集的集合，天然松耦合且有良好的扩展性，用户可以按需取用，最新版本的 Flux 引入了许多新功能，使其更加灵活多样，Flux 是 CNCF 孵化的毕业项目
```

### 2：`Flux` 的组件

```shell
# Flux 是使用 GitOps Tookit 组件构建的，它是一组
1：专用工具 和 Flux控制器
2：可组合的 API
3：在 Flux CD Github 组织下，为构建基于 Kubernetes 的持续交付提供可重用的 Go 依赖包

# 用于在 Kubernetes 之上构建持续交付
```

![flux-cd](https://fluxcd.io/img/diagrams/gitops-toolkit.png)

```shell
# 如上是 Flux CD 的基本架构，这些 API 包括 Kubernetes 自定义资源，可以由集群用户或其他自动化工具进行创建和更新，我们可以使用这个工具包扩展 Flux，并构建自己的持续交付系统

# Flux 的核心工具包括如下 5 个
1：Source Controller
2：Kustomize Controller
3：Helm Controller
4：Notification Controller
5：Image automation controllers

# 每个工具包都是一个控制器，都有自己的自定义资源，用于定义其行为
```

### 3：`Flux` 安装

```shell
# Flux 项目由命令行工具（Flux CLI）和一系列 Kubernetes 控制器组成，要安装 Flux，首先需要下载 Flux CLI，然后使用 CLI，可以在集群上部署 Flux 控制器并配置 GitOps 交付流水线
# Flux CLI 是一个二进制的可执行文件，可以从 Github 上下载

# 不过这里有个问题就是 Flux2 它检查的时候要求的是你的 K8S 版本 大于等于 1.25，所以最好是版本升级一下

# 安装
[root@k-m-1 ~]# wget https://ghproxy.com/https://github.com/fluxcd/flux2/releases/download/v2.1.1/flux_2.1.1_linux_amd64.tar.gz
[root@k-m-1 ~]# tar xf flux_2.1.1_linux_amd64.tar.gz
[root@k-m-1 ~]# mv flux /usr/local/bin/
[root@k-m-1 ~]# flux -v
flux version 2.1.1

# Flux CLI 提供一个 bootstrap 命令在 Kubernetes 集群上部署 Flux 控制器，并配置控制器从 Git 存储库同步集群状态，除了安装控制器之外，bootstrap 命令还将 Flux 清单推送到 Git 存储库，并将 Flux 配置为从 Git 进行自我更新
```

![flux-cd](https://fluxcd.io/flux/img/flux-bootstrap-diagram.png)

```shell
# 我们可以先检查一下环境
[root@k-m-1 ~]# flux check
► checking prerequisites
✔ Kubernetes 1.26.8 >=1.25.0-0
► checking controllers
✗ no controllers found in the 'flux-system' namespace with the label selector 'app.kubernetes.io/part-of=flux'
► checking crds
✗ no crds found with the label selector 'app.kubernetes.io/part-of=flux'
✗ check failed

# 这里检查告诉我们，没有控制器，也没有 CRD，因为我们还没有去安装它，所以这个检查是没有任何问题的，我们等下安装好了再检查就会不一样了
# 如果集群上存在 Flux 控制器，则 Bootstrap 命令将在需要执行时执行升级，Bootstrap 时幂等的，可以安全的运行该命令任意多次

# Flux 与主流的 Git 提供集成，以简化部署密钥和其他身份验证机制的初始化设置，比如我们这里选择与 Gitea 集成，但是 Flux 没有针对 Gitea 的集成，所以我们先择 Git 的形式，这样就可以集成第三方的 Git 仓库了

flux bootstrap git \
--url=http://git.devops-engineer.com.cn/gitlayzer/fluxcd.git \
--username=gitlayzer \
--password=gitlayzer \
--branch=master \
--allow-insecure-http=true \
--path=clusters/dev-cluster

# 部署的过程是很蛋疼的，因为它走的是 ghcr 的镜像，可能需要自己改改，否则拉不下来，代理地址 ghcr.dockerproxy.com，多说无益，自己换吧

helm-controller：ghcr.dockerproxy.com/fluxcd/helm-controller:v0.36.1
kustomize-controller：ghcr.dockerproxy.com/fluxcd/kustomize-controller:v1.1.0
notification-controller：ghcr.dockerproxy.com/fluxcd/notification-controller:v1.1.0
source-controller：ghcr.dockerproxy.com/fluxcd/source-controller:v1.1.1

# 检查部署
[root@k-m-1 ~]# kubectl get pod -n flux-system 
NAME                                       READY   STATUS    RESTARTS   AGE
helm-controller-58695c7c56-97gtn           1/1     Running   0          26m
kustomize-controller-859c949c64-58nlj      1/1     Running   0          26m
notification-controller-7d7747dd84-6fp48   1/1     Running   0          26m
source-controller-6d9b9567bf-49ztw         1/1     Running   0          25m

# 使用 flux CLI 检查
[root@k-m-1 ~]# flux check
► checking prerequisites
✔ Kubernetes 1.26.8 >=1.25.0-0
► checking controllers
✔ helm-controller: deployment ready
► ghcr.io/fluxcd/helm-controller:v0.36.1
✔ kustomize-controller: deployment ready
► ghcr.io/fluxcd/kustomize-controller:v1.1.0
✔ notification-controller: deployment ready
► ghcr.io/fluxcd/notification-controller:v1.1.0
✔ source-controller: deployment ready
► ghcr.io/fluxcd/source-controller:v1.1.1
► checking crds
✔ alerts.notification.toolkit.fluxcd.io/v1beta2
✔ buckets.source.toolkit.fluxcd.io/v1beta2
✔ gitrepositories.source.toolkit.fluxcd.io/v1
✔ helmcharts.source.toolkit.fluxcd.io/v1beta2
✔ helmreleases.helm.toolkit.fluxcd.io/v2beta1
✔ helmrepositories.source.toolkit.fluxcd.io/v1beta2
✔ kustomizations.kustomize.toolkit.fluxcd.io/v1
✔ ocirepositories.source.toolkit.fluxcd.io/v1beta2
✔ providers.notification.toolkit.fluxcd.io/v1beta2
✔ receivers.notification.toolkit.fluxcd.io/v1
✔ all checks passed

# 其实它帮我们创建了一个项目，这个项目里面包含了两个资源，这个在我们的 Git 仓库里面也可以看到，但是因为我本地没有解析到这个域名，所以我需要修改一下
[root@k-m-1 ~]# kubectl get kustomization,gitrepositories -n flux-system 
NAME                                                    AGE   READY   STATUS
kustomization.kustomize.toolkit.fluxcd.io/flux-system   28m   False   Source is not ready, artifact not found

NAME                                                 URL                                                               AGE   READY   STATUS
gitrepository.source.toolkit.fluxcd.io/flux-system   ssh://gitlayzer@git.devops-engineer.com.cn/gitlayzer/fluxcd.git   28m   False   failed to checkout and determine revision: unable to clone 'ssh://gitlayzer@git.devops-engineer.com.cn/gitlayzer/fluxcd.git': dial tcp: lookup git.devops-engineer.com.cn on 10.96.0.10:53: no such host

# 改完地址就正常了
[root@k-m-1 ~]# kubectl get kustomizations,gitrepositories -n flux-system 
NAME                                                    AGE     READY   STATUS
kustomization.kustomize.toolkit.fluxcd.io/flux-system   3m30s   True    Applied revision: master@sha1:cd5650f8def7eb4b53d9f7da9b46603f808e68f0

NAME                                                 URL                                                                                AGE     READY   STATUS
gitrepository.source.toolkit.fluxcd.io/flux-system   http://gitlayzer:gitlayzer@gitea.kube-ops.svc.cluster.local/gitlayzer/fluxcd.git   3m30s   True    stored artifact for revision 'master@sha1:cd5650f8def7eb4b53d9f7da9b46603f808e68f0'

# 这样我们的 Flux CD 就部署好了，然后我们再来看看一些 CRD
[root@k-m-1 ~]# kubectl get crd | grep flux
alerts.notification.toolkit.fluxcd.io                 2023-10-09T00:26:56Z
buckets.source.toolkit.fluxcd.io                      2023-10-09T00:26:56Z
gitrepositories.source.toolkit.fluxcd.io              2023-10-09T00:26:56Z
helmcharts.source.toolkit.fluxcd.io                   2023-10-09T00:26:56Z
helmreleases.helm.toolkit.fluxcd.io                   2023-10-09T00:26:56Z
helmrepositories.source.toolkit.fluxcd.io             2023-10-09T00:26:56Z
kustomizations.kustomize.toolkit.fluxcd.io            2023-10-09T00:26:56Z
ocirepositories.source.toolkit.fluxcd.io              2023-10-09T00:26:56Z
providers.notification.toolkit.fluxcd.io              2023-10-09T00:26:56Z
receivers.notification.toolkit.fluxcd.io              2023-10-09T00:26:56Z

# 然后我们就开始示例了
```

### 4：`Flux` 示例

```shell
# 这里我们还是拿上一篇文章的 Jenkinsfile 来做，因为里面有一个 helm 的目录，下面是一个写好的 Chart 包，因为我们做的是通过 argocd-image-updater，所以我们的 Jenkinsfile 只做了 CI 的部分，其他部分交给了 Argo CD，然后发布是通过 Argo CD + Argo Image Updater 进行检测仓库的最新镜像并通过 Argo CD 更新镜像触发发布的，那么接下来我们可以通过 Flux 来部署应用了，首先需要为 Flux CD 创建一个仓库连接信息，这就需要用到一个名为 GitRepository 的 CRD 对象，该对象可以定义一个 Source 代码源来为 Git 存储库一个版本生成一个制品，如下所示
```

```yaml
# k8s-demo-git-repo.yaml
apiVersion: source.toolkit.fluxcd.io/v1
kind: GitRepository
metadata:
  name: k8s-demo
spec:
  url: http://gitea.kube-ops.svc.cluster.local/gitlayzer/cdk-web-service-deploy.git  # 应用仓库地址
  timeout: 60s   # 超时时间 60 秒
  interval: 60s  # 60 秒检查一次
  ref:
    branch: master  # 检出 master 分支
  # secretRef:
  #   name: k8s-demo  # 如果是私有仓库，则需要配置这个 Secret，这个 Secert 包含一个账号密码
---
# apiVersion: v1
# kind: Secret
# metadata:
#   name: k8s-demo
# type: Opaque
# stringData:
#   username: <base64编码后的账号>
#   password: <base64编码后的密码>
```

```shell
# 这里我们创建一个名为 k8s-demo 的 GitRepository 对象，其中 spec 字段定义了如何从 Git 存储库提取数据，url 字段指定了 Git 存储库的URL，ref 字段指定了要提取的代码分支，interval 字段指定了从 Git 存储库提取数据的频率，secretRef 字段指定了包含 GitRepository 身份验证凭据的 Secret，我这里是个公有库，我就不使用了

# 对于 HTTPS 仓库，Secret 必须包含基本认证的 username 和 passwrod 字段，或者令牌认证的 bearerToken 字段，对于 SSH 仓库，Secret 必须包含 identity 和 known_hosts 字段
[root@k-m-1 fluxcd]# kubectl apply -f k8s-demo-git-repo.yaml
gitrepository.source.toolkit.fluxcd.io/k8s-demo created

# 检查
[root@k-m-1 fluxcd]# kubectl get gitrepositories.source.toolkit.fluxcd.io 
NAME       URL                                                                            AGE   READY   STATUS
k8s-demo   http://gitea.kube-ops.svc.cluster.local/gitlayzer/cdk-web-service-deploy.git   4s    True    stored artifact for revision 'master@sha1:d053f202085b8277d2dadf84b45b486c17b3c98e'

# 这个操作证明我们现在把这个应用的仓库和K8S集群关联起来了，意思就是将代码仓库映射成了K8S资源对象，然后我们要做的就是使用这个映射的资源对象去部署我们的项目，比如：kustomize，或者 helm 的 CRD 去部署，因为我们这里的应用是个 Helm，所以我们选择使用 Helm 的方式部署

# 接下来我们只要为该应用创建一个部署清单策略即可，我们需要创建一个 HelmRelease 对象，该对象可以定义一个包含 Chart 的源（可以是 HelmRepository，GitRepository 或 Bucket）告知 source-controller，以便 HelmRelease 能够引用它，很明显，我们这里的源就是上面定义的 GitRepository 对象，我们创建的 HelmRelease 对象如下
```

```yaml
# k8s-demo-helm-release.yaml
apiVersion: helm.toolkit.fluxcd.io/v2beta1
kind: HelmRelease
metadata:
  name: k8s-demo
  namespace: default
spec:
  interval: 60s
  chart:
    spec:
      reconcileStrategy: Revision  # 每次 Chart 变更都会触发更新
      chart: helm    # Chart 是 Helm Chart 在 SourceRef 中可用的名称或路径
      sourceRef:
        kind: GitRepository
        name: k8s-demo
        namespace: default
      valuesFiles:
      - helm/values.yaml
      interval: 60s
  values:
    replicaCount: 3
```

```shell
# 上面我们定义了一个 HelmRelease 对象，其中 chart 字段指定了 Helm Chart 的源，因为我们这里的 Helm Chart 是存储在 Git 代码库中的，所以我们通过 sourceRef 字段来指定 GitRepository 对象，interval 字段指定了从 Git 存储库提取数据的频率，values 字段指定了 Chart 的 values 值

# 其中 valuesFiles 字段是一个备选的值文件列表，用作 Chart 的 Values 值（默认情况不包括velues.yaml），是相对于 SourceRef 的路径，Values 文件按照此列表的顺序合并，最后一个文件将覆盖第一个文件

# 然后我们部署它
[root@k-m-1 fluxcd]# kubectl apply -f k8s-demo-helm-release.yaml 
helmrelease.helm.toolkit.fluxcd.io/k8s-demo created

# 检查部署
[root@k-m-1 fluxcd]# kubectl get helmreleases.helm.toolkit.fluxcd.io
NAME       AGE   READY   STATUS
k8s-demo   18s   True    Release reconciliation succeeded

# 检查 Helm Chart 的 CRD
[root@k-m-1 fluxcd]# kubectl get helmcharts.source.toolkit.fluxcd.io 
NAME               CHART   VERSION   SOURCE KIND     SOURCE NAME   AGE   READY   STATUS
default-k8s-demo   helm    *         GitRepository   k8s-demo      44m   True    packaged 'cdk-web-service' chart with version '0.0.1+1' and merged values files [helm/values.yaml]


# 检查应用
[root@k-m-1 fluxcd]# helm ls
NAME    	NAMESPACE	REVISION	UPDATED                                	STATUS  	CHART                  	APP VERSION
k8s-demo	default  	1       	2023-10-09 18:00:12.290054096 +0000 UTC	deployed	cdk-web-service-0.0.1+1	0.0.1      

# 检查资源
[root@k-m-1 fluxcd]# kubectl get pod,svc,ingress
NAME                                   READY   STATUS    RESTARTS   AGE
pod/cdk-web-service-79648bcf94-5hvsw   1/1     Running   0          91s
pod/cdk-web-service-79648bcf94-8bqsb   1/1     Running   0          91s
pod/cdk-web-service-79648bcf94-dpdw6   1/1     Running   0          91s

NAME                      TYPE        CLUSTER-IP   EXTERNAL-IP   PORT(S)    AGE
service/cdk-web-service   ClusterIP   10.96.2.98   <none>        8080/TCP   91s
service/kubernetes        ClusterIP   10.96.0.1    <none>        443/TCP    20h

NAME                                        CLASS   HOSTS                            ADDRESS     PORTS   AGE
ingress.networking.k8s.io/cdk-web-service   nginx   cdk-web.devops-engineer.com.cn   10.0.0.11   80      91s

# 请求资源
[root@k-m-1 fluxcd]# curl cdk-web.devops-engineer.com.cn
{"msg":"Hello This is cdk-web-service, This Version is v2"}

# 可以发现，我们指定的参数也是生效了，因为最开始的副本我们在仓库中定义的是 1 个副本，我们用 HelmRelease 的时候使用了 values 指定了这个参数为 3，结果根据需求也是 3 个副本，这也就说明，我们的 HelmRelease 是按照我们的要求部署的，针对于我们现在这个项目来说当我们变更了仓库内 values.yaml 的参数时，它就会通过 interval 的时间去更新应用，那么我们来变更一下镜像的版本试一下

# 老镜像版本
[root@k-m-1 fluxcd]# kubectl get deployments.apps cdk-web-service -o jsonpath="{.spec.template.spec.containers[0].image}"
layzer/cdk-web-service:4dd3f04
# 新版本
layzer/cdk-web-service:3174d35

# 直接修改 values.yaml 内的 tag 就可以了
[root@k-m-1 cdk-web-service-deploy]# git add .
[root@k-m-1 cdk-web-service-deploy]# git commit -m "Change Image Tag"
[master 9d2552b] Change Image Tag
 1 file changed, 1 insertion(+), 1 deletion(-)
[root@k-m-1 cdk-web-service-deploy]# git push origin master 
Counting objects: 7, done.
Delta compression using up to 8 threads.
Compressing objects: 100% (4/4), done.
Writing objects: 100% (4/4), 347 bytes | 0 bytes/s, done.
Total 4 (delta 3), reused 0 (delta 0)
remote: . Processing 1 references
remote: Processed 1 references in total
To http://git.devops-engineer.com.cn/gitlayzer/cdk-web-service-deploy.git
   2ff59a6..9d2552b  master -> master

# 不过这里有一个问题要说一下 关于 Helm 里面的 interval 最低要 60 秒哦

# 检查更新
[root@k-m-1 fluxcd]# kubectl get deployments.apps cdk-web-service -o jsonpath="{.spec.template.spec.containers[0].image}"
layzer/cdk-web-service:3174d35

# 检查 gitrepository，主要看 sha1 的值是否有变化
[root@k-m-1 fluxcd]# kubectl get gitrepositories.source.toolkit.fluxcd.io k8s-demo 
NAME       URL                                                                            AGE     READY   STATUS
k8s-demo   http://gitea.kube-ops.svc.cluster.local/gitlayzer/cdk-web-service-deploy.git   5h15m   True    stored artifact for revision 'master@sha1:9d2552b1b0a1057d24fe7c3eeb5f6514369ee8fd'

# 再查看 Helm Chart 的更新版本，看 chart with version
[root@k-m-1 fluxcd]# kubectl get helmcharts.source.toolkit.fluxcd.io default-k8s-demo 
NAME               CHART   VERSION   SOURCE KIND     SOURCE NAME   AGE   READY   STATUS
default-k8s-demo   helm    *         GitRepository   k8s-demo      86m   True    packaged 'cdk-web-service' chart with version '0.0.1+2' and merged values files [helm/values.yaml]

# 测试访问应用
[root@k-m-1 fluxcd]# curl cdk-web.devops-engineer.com.cn
{"msg":"Hello This is cdk-web-service, This Version is v1"}

# 看到现在我们可以确定，的确是更新了，而且是更新了仓库之后自己就更新了应用的，不过我们要提一下，这里我们关注一下 helmcharts 这个对象，里面有一个 Reconcile Strategy，它的有效值是 ChartVersion/Revision，而默认是 ChartVersion，那么意思就是 Chart 版本变化才会去更新新的制品，所以我们如果真的要使用自动更新，可能还需要改一下自动生成的 HelmCharts 这个对象中的 Reconcile Strategy

[root@k-m-1 fluxcd]# kubectl get helmcharts.source.toolkit.fluxcd.io 
NAME               CHART   VERSION   SOURCE KIND     SOURCE NAME   AGE    READY   STATUS
default-k8s-demo   helm    *         GitRepository   k8s-demo      129m   True    packaged 'cdk-web-service' chart with version '0.0.1+4' and merged values files [helm/values.yaml]

# 不过我们像 Argo CD 那样它利用 ArgoCD Image Updater 去自动检测镜像并更新，那么 Flux CD 该如何去更新呢？
# 其实在最上面我们讲组件的时候就提到了 Image automation controllers，这个控制器其实和 ArgoCD Image Updater 的功能是类似的，它是可以监测镜像仓库的最新的镜像版本，然后可以去修改资源清单的镜像，然后再推送到代码仓库中去，那么我们下面来使用一下它，不过，这个控制器默认是没有安装的哦
```

### 5：`Flux` 镜像自动化

```shell
# 但是这样的话，我们每次都需要在 CI 流水线去手动更新 Git 代码仓库中的 Values 文件的镜像的 tag，这样就感觉比较麻烦了，和 ArgoCD 类似，Flux 也提供了一个 Image Automation 控制器的功能

# 当新的容器镜像可用时，iamge-reflector-controller 和 image-automation-controller 可以协同工作来更新 Git 存储库
1：image-reflector-controller：扫描镜像存储库并反射到 Kubernetes 资源中的镜像元数据
2：image-automation-controller：根据扫描的最新镜像更新 YAML 文件，并将更改提交到指定的 Git 存储库
```

![flux-cd](https://fluxcd.io/img/image-update-automation.png)

```shell
# 但是需要注意的是默认情况下 Flux 不会自动安装 iamge-reflector-controller 和 image-automation-controller，所以我们需要手动安装这两个控制器，通过 --components-extra 参数来指定要安装的组件

flux bootstrap git \
--url=http://git.devops-engineer.com.cn/gitlayzer/fluxcd.git \
--username=gitlayzer \
--password=gitlayzer \
--branch=master \
--allow-insecure-http=true \
--path=clusters/dev-cluster
--components-extra image-reflector-controller,image-automation-controller

# 如果上面的安装方法不行，可以使用下面的方法，直接安装，因为上面的方法可能会卡在如下过程，但是具体原因又不太清楚
◎ waiting for Kustomization "flux-system/flux-system" to be reconciled

# URL：https://github.com/fluxcd/flux2/tree/main/manifests/bases
# 在上面的 URL 中找到这两个应用然后将4个文件分别拉下来，然后再部署
[root@k-m-1 image-automation-controller]# kubectl apply -k ./ -n flux-system 
customresourcedefinition.apiextensions.k8s.io/imageupdateautomations.image.toolkit.fluxcd.io unchanged
serviceaccount/image-automation-controller created
deployment.apps/image-automation-controller created

[root@k-m-1 image-reflector-controller]# kubectl apply -k ./ -n flux-system 
customresourcedefinition.apiextensions.k8s.io/imagepolicies.image.toolkit.fluxcd.io created
customresourcedefinition.apiextensions.k8s.io/imagerepositories.image.toolkit.fluxcd.io created
serviceaccount/image-reflector-controller created
deployment.apps/image-reflector-controller created

# 检查部署
[root@k-m-1 ~]# kubectl get pod -n flux-system 
NAME                                               READY   STATUS    RESTARTS      AGE
pod/helm-controller-58695c7c56-vs6r2               1/1     Running   1 (13h ago)   3d3h
pod/image-automation-controller-78645c9469-68sh6   1/1     Running   0             2m59s
pod/image-reflector-controller-8568b49675-j87g4    1/1     Running   0             84s
pod/kustomize-controller-859c949c64-nm57k          1/1     Running   1 (13h ago)   3d3h
pod/notification-controller-7d7747dd84-np6sz       1/1     Running   1 (13h ago)   3d3h
pod/source-controller-6d9b9567bf-fdfwb             1/1     Running   1 (13h ago)   3d3h

# 这两个控制器安装完成之后，我们就可以使用 Flux 配置容器镜像扫描和部署发布了，对于容器镜像，可以将 Flux 配置为：
1：扫描镜像仓库并获取镜像标签
2：根据定义的策略（semver，calver，regex）选择最新的标签
3：替换 Kubernetes 清单中的标签（YAML格式）
4：检出分支，提交并更改推送到远程 Git 仓库
5：在集群中应用更改并变更容器镜像

# 对于生产环境，此功能允许你自动部署应用程序补丁（CVE和错误修复），并在 Git 历史记录中保留所有部署记录

# 那么下面我们就来思考一下对于生产环境和测试环境的 CI/CD 工作流是怎样的
# 生产环境 CI/CD 工作流
DEV：将错误修复推送到应用程序存储库
DEV：修改补丁版本发布，例如：v1.0.1
CI：构建并推送标记为 registry.domain/org/app:v1.0.1 的容器镜像
CD：从镜像仓库中提取最新的镜像元数据（Flux镜像扫描）
CD：将应用程序清单中的镜像标签更改为 v1.0.1 （Flux 集群到 Git 调谐）
CD：将 v1.0.1 部署到争产集群（Flux Git 到集群调谐）

# Staging 环境 CI/CD 工作流
DEV：将代码更改推送到应用程序存储库主分支
CI：构建并推送标记为：${GIT_BRANCH}-${GIT_SHA:0:7}-${date +%s} 的容器镜像
CD：从镜像仓库中提取最新的镜像元数据（Flux镜像扫描）
CD：将应用程序清单中的镜像标签更新为 ${GIT_BRANCH}-${GIT_SHA:0:7}-${date +%s} 生成的tag（Flux 集群到 Git 调谐）
CD：将 ${GIT_BRANCH}-${GIT_SHA:0:7}-${date +%s} 这个tag生成的镜像部署到 Staging 环境集群（Flux Git 到集群调谐）

# 那么这里我们的示例使用的镜像是 layzer/cdk-web-service 我们可以先创建一个 ImageRepository 来告诉 Flux 扫描哪儿个镜像仓库查找标签
flux create image repository k8s-demo \
--namespace=default \
--image=layzer/cdk-web-service \
--interval=60s \
--export > k8s-demo-registry.yaml

# 这样就会生成如下文件内容
[root@k-m-1 fluxcd]# cat k8s-demo-registry.yaml 
```

```yaml
apiVersion: image.toolkit.fluxcd.io/v1beta2
kind: ImageRepository
metadata:
  name: k8s-demo
  namespace: default
spec:
  image: layzer/cdk-web-service
  interval: 1m0s
```

```shell
# 当我们不会编写 ImageRepository 时就可以使用命令帮我们生成
# 当然对于私有镜像仓库可以使用 kubectl create secret docker-registry 与 ImageRepository 相同的命名空间中创建一个 Secreet，同样使用的密码需要通过 Docker hub 去生成 Token，然后通过如下方法指定它
```

```yaml
apiVersion: image.toolkit.fluxcd.io/v1beta2
kind: ImageRepository
metadata:
  name: k8s-demo
  namespace: default
spec:
  image: layzer/cdk-web-service
  secretRef:
    name: dockerhub-auth
  interval: 1m0s
```

```shell
# 不过我这里是共有的仓库，是不需要创建 Secret 的，我们直接部署这个资源
[root@k-m-1 fluxcd]# kubectl apply -f k8s-demo-registry.yaml 
imagerepository.image.toolkit.fluxcd.io/k8s-demo created

# 检查之后会发现有一次扫描，并列出有多少个 tag
[root@k-m-1 fluxcd]# kubectl get imagerepositories.image.toolkit.fluxcd.io 
NAME       LAST SCAN              TAGS
k8s-demo   2023-10-12T19:35:13Z   21

# 如果说你要告诉 Flux 过滤标签的时候使用 semver 版本范围的标签，则开源创建一个 ImagePolicy 对象，比如选择标签为 ${GIT_BRANCH}-${GIT_SHA:0:7}-${date +%s} 的最新主分支构建，则可以使用如下 ImagePolicy
```

```yaml
apiVersion: image.toolkit.fluxcd.io/v1beta2
kind: ImagePolicy
metadata:
  name: k8s-demo
spec:
  filterTags:
    pattern: "^main-[a-fA-F0-9]+-(?P<ts>.*)"
    extract: "$ts"
  policy:
    numerical:
      order: asc
```

```shell
# 选择稳定版本
```

```yaml
apiVersion: image.toolkit.fluxcd.io/v1beta2
kind: ImagePolicy
metadata:
  name: k8s-demo
spec:
  policy:
    semver:
      range: ">=1.0.0"
```

```shell
# 选择 1.x 范围内的最新稳定补丁版本（semver）
```

```yaml
apiVersion: image.toolkit.fluxcd.io/v1beta2
kind: ImagePolicy
metadata:
  name: k8s-demo
spec:
  policy:
    semver:
      range: ">=1.0.0 < 2.0.0"
```

```shell
# 选择最新版本，包括预发行版（semver）
```

```yaml
apiVersion: image.toolkit.fluxcd.io/v1beta2
kind: ImagePolicy
metadata:
  name: k8s-demo
spec:
  policy:
    semver:
      range: ">=1.0.0-0"
```

```shell
# 由于 ImagePolicy 对象的策略只支持三种
1：SemVer：语义版本
2：Alphabetical：字母顺序
3：Numerical：数字顺序

# 而前面我们的镜像标签都是通过 git commit id 生成的，不符合这里的规范，所以我们可以改一下镜像 Tag 的生成策略
```

```groovy
podTemplate(cloud: "kubernetes", containers: [
    containerTemplate(name: 'golang', image: 'golang:1.21.1-alpine3.18', command: 'cat', ttyEnabled: true),
    containerTemplate(name: 'docker', image: 'docker:latest', command: 'cat', ttyEnabled: true),
], serviceAccount: 'jenkins', envVars: [
    envVar(key: 'DOCKER_HOST', value: 'tcp://docker-dind:2375')
]) {
    node(POD_LABEL) {
      def Repo = checkout scm
      def GitCommit = Repo.GIT_COMMIT.substring(0,8)
      def GitBranch = Repo.GIT_BRANCH
      GitBranch = GitBranch.replace("origin/", "")
      def unixTime = (new Date().time.intdiv(1000))  
      
      def imageTag = "${GitBranch}-${GitCommit}-${unixTime}"
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
# 我们提交完之后就会触发一次构建，那么这次构建给到的镜像是什么呢？我们可以利用 ImageRepository 这个资源来看看
[root@k-m-1 cdk-web-service]# kubectl describe imagerepositories.image.toolkit.fluxcd.io k8s-demo
......
  Last Scan Result:
    Latest Tags:
      master-654ecee5-1697193906
      fdfdf4a
      f8971ad
      f4a234f
      ec13c16
      e6bd670
      dabae40
      d2d518f
      d28bd6d
      cfabea0
    Scan Time:  2023-10-13T10:46:03Z
    Tag Count:  23


# 可以看到，现在这个镜像的 Tag 最新的，然后我们将 ImagePolicy 也去落地部署一下
```

```yaml
apiVersion: image.toolkit.fluxcd.io/v1beta1
kind: ImagePolicy
metadata:
  name: k8s-demo
spec:
  imageRepositoryRef:
    name: k8s-demo
  filterTags:
    pattern: "^master-[a-fA-F0-9]+-(?P<ts>.*)"
    extract: "$ts"
  policy:
    numerical:
      order: asc
```

```shell
# 部署它
[root@k-m-1 fluxcd]# kubectl apply -f k8s-demo-image-policy.yaml 
imagepolicy.image.toolkit.fluxcd.io/k8s-demo created

# 检查它查找出来的镜像
[root@k-m-1 fluxcd]# kubectl get imagepolicies.image.toolkit.fluxcd.io 
NAME       LATESTIMAGE
k8s-demo   layzer/cdk-web-service:master-654ecee5-1697193906

# 可以看到，它已经按照我们的过滤规则去找到了最新的匹配的镜像了，然后这个时候我们就要考虑，如何将这个 Tag 想办法更新到我们的 Helm Charts 上去了，接下来我们就需要创建一个 IamgeUpdateAutomation 如何知道把我们更新后的镜像标签写入到哪儿个 values 文件中，写入到哪儿个位置

# 这个时候就需要用到 marker 功能了，用来标记 image Automation Controller 自动更新的位置，比如我们这里使用的是 Helm Chart 来部署的应用，决定使用哪儿个版本的镜像通过 values 这个 yaml 指定
```

```yaml
image:
  repository: layzer/cdk-web-service # {"$imagepolicy": "default:k8s-demo:name"}
  pullPolicy: IfNotPresent # {"$imagepolicy": "default:k8s-demo:tag"}
  tag: "3174d35"
```

```shell
# 我们需要在这个文件中添加一个 marker 来告诉 flux 将镜像标签写入到哪儿个位置，这个镜像策略的 marker 标签格式有如下几种
{"imagepolicy": "<policy-namespace>:<policy-name>"}
{"imagepolicy": "<policy-namespace>:<policy-name>:tag"}
{"imagepolicy": "<policy-namespace>:<policy-name>:name"}

# 这些标记作为注释内联放置在目标 YAML 中，Setter 策略是指 Flux 可以在调谐期间找到并替换的 kyaml setter

# 我们重新修改一下 values.yaml 添加 marker 标记
```

```yaml
replicaCount: 2

namespace: ""

labels:
  app: cdk-web-service

env:
- name: GIN_MODE
  value: release

image:
  repository: layzer/cdk-web-service # {"$imagepolicy": "default:k8s-demo:name"}
  pullPolicy: IfNotPresent
  tag: "3174d35"  # {"$imagepolicy": "default:k8s-demo:tag"}

imagePullSecrets: []
nameOverride: "cdk-web-service"
fullnameOverride: "cdk-web-service"

service:
  type: ClusterIP
  port: 8080

ingress:
  enabled: true
  className: "nginx"
  annotations: {}
    # kubernetes.io/ingress.class: nginx
    # kubernetes.io/tls-acme: "true"
  hosts:
  - host: cdk-web.devops-engineer.com.cn
    paths:
    - path: /
      pathType: Prefix
  tls: []
  #  - secretName: chart-example-tls
  #    hosts:
  #      - chart-example.local

resources:
  limits:
    cpu: 100m
    memory: 128Mi
  requests:
    cpu: 100m
    memory: 128Mi
```

```shell
# 注意，这里的标记的注释是存在的哦，然后我们推送到仓库中去，下面还需要创建一个 ImageUpdateAutomation 对象来告诉 Flux 将镜像更新写入到哪个 Git 存储库，同样可以使用 Flux CLI 创建

flux create image update k8s-demo \
--namespace=default \
--interval=1m \
--git-repo-ref=k8s-demo \
--git-repo-path="./helm/values.yaml" \
--checkout-branch=master \
--push-branch=master \
--author-name=fluxcdbot \
--author-email=fluxcd@devops-engineer.com.cn \
--commit-template="{{range .Updated.Images}}{{println .}}{{end}}" \
--export > k8s-demo-automation.yaml

# 生成的资源对象如下
```

```yaml
---
apiVersion: image.toolkit.fluxcd.io/v1beta1
kind: ImageUpdateAutomation
metadata:
  name: k8s-demo
  namespace: default
spec:
  git:
    checkout:
      ref:
        branch: master
    commit:
      author:
        email: fluxcd@devops-engineer.com.cn
        name: fluxcdbot
      messageTemplate: '{{range .Updated.Images}}{{println .}}{{end}}'
    push:
      branch: master
  interval: 1m0s
  sourceRef:
    kind: GitRepository
    name: k8s-demo
  update:
    path: ./helm/values.yaml
    strategy: Setters
```

```shell
# 部署这个资源
[root@k-m-1 fluxcd]# kubectl apply -f k8s-demo-automation.yaml 
imageupdateautomation.image.toolkit.fluxcd.io/k8s-demo created

# 然后我们测试修改代码并创建一个新的镜像看看结果怎样
[root@k-m-1 cdk-web-service]# kubectl get imagepolicies.image.toolkit.fluxcd.io -w
NAME       LATESTIMAGE
k8s-demo   layzer/cdk-web-service:master-654ecee5-1697193906
k8s-demo   layzer/cdk-web-service:master-654ecee5-1697193906
k8s-demo   layzer/cdk-web-service:master-18bedbdf-1697196908

# watch 到新的镜像了，然后我们查看一下 iamgeupdateautomation 控制器的资源与日志
[root@k-m-1 fluxcd]# kubectl get imageupdateautomations.image.toolkit.fluxcd.io 
NAME       LAST RUN
k8s-demo   2023-10-13T12:56:48Z

# 日志
{"level":"info","ts":"2023-10-13T12:55:48.567Z","msg":"pushed commit to origin","controller":"imageupdateautomation","controllerGroup":"image.toolkit.fluxcd.io","controllerKind":"ImageUpdateAutomation","ImageUpdateAutomation":{"name":"k8s-demo","namespace":"default"},"namespace":"default","name":"k8s-demo","reconcileID":"952df1dc-f5bc-4e23-90f7-70f5f314462b","revision":"db981b873576864610372402450cf8f8bd5f536d","branch":"master"}

# 那么这个时候我们就可以去查看是否更新应用了
[root@k-m-1 fluxcd]# kubectl get helmcharts.source.toolkit.fluxcd.io 
NAME               CHART   VERSION   SOURCE KIND     SOURCE NAME   AGE     READY   STATUS
default-k8s-demo   helm    *         GitRepository   k8s-demo      3d23h   True    packaged 'cdk-web-service' chart with version '0.0.1+6' and merged values files [helm/values.yaml]

# 查看应用
[root@k-m-1 fluxcd]# kubectl get pod
NAME                              READY   STATUS    RESTARTS   AGE
cdk-web-service-8d4cbcbfd-m5zr8   1/1     Running   0          67s
cdk-web-service-8d4cbcbfd-m9s7c   1/1     Running   0          57s
cdk-web-service-8d4cbcbfd-zf6hz   1/1     Running   0          55s

# 查看镜像
[root@k-m-1 fluxcd]# kubectl get deployments.apps cdk-web-service -o jsonpath="{.spec.template.spec.containers[0].image}"
layzer/cdk-web-service:master-1f1d20bb-1697217817

# 访问测试
[root@k-m-1 fluxcd]# curl cdk-web.devops-engineer.com.cn
{"msg":"Hello cdk-web-service!!!"}[

# OK 那么到这里我们的自动化也就完成了，不过这里整体做下来，总结了几个坑点
1：flux bootstrap 命令安装集群的时候可能会卡在一个步骤，但是flux的控制器又安装好了（这个时候停掉命令就可以了）
2：创建 HelmRelease 时注意一个参数 reconcileStrategy: ChartVersion，它的可选项是 ChartVersion/Revision，一个是更新 Charts 的版本才会触发，一个是没更新一次都会触发
3：当创建 GitRepository 的时候，不管你是否需要用到 Secret，你都应该指定它，因为在使用自动化镜像的时候会用到它
4：自动化更新镜像时，匹配镜像的的策略比较难懂
5：创建的 YAML 资源比较多，比较容易混淆，本次部署一共创建了 5 个 YAML 的资源对象，显得太多了，它比较适合使用在开发 DevOps 的平台中使用

# 总之 FluxCD 没有人用到生产是有一定的原因的，上手难度比 Argo 全家桶还有难度，所以说它比较适合拿去开发平台，那么到这里 Flux 的基本的使用方法就做完了，后面我们还会针对它做一个 Dashboard 和 通知的操作
```

### 6：`Flux Dashboard`

```shell
# Flux 官方其实现在并没有给我们提供 Dashboard，在早期 1.0 的版本是有的，现在我们则需要去利用 Weave 为 Flux CD 提供的一个 Web UI 叫做 weave-gitops，它上面提供了关于 Flux CD 一些 Dashboard 的操作

# Weave Gitops 可以帮助应用程序的运维人员轻松的发现和解决问题，简化和扩展 GitOps 和持续交付的采用，UI 提供了引导式体验，可以帮助用户轻松的发现 Flux 对象之间的关系并加深理解，同时提供对应用程序部署的见解

# Weave GitOps 除了提供一个开源版本之外，还有一个企业版，其 OSS 版本是一个简单的开源开发者平台，适合那些没有 Kubernetes 专业知识但想要云原生应用程序的人，它包括 UI 和许多其他的功能，使团队超越简单的 CI/CD 系统，体验启用 GitOps 并在集群中运行应用程序是多么的容易，当然我们这里使用的是开源版本

# Weave GitOps 提供一个命令行界面，可以帮助用户创建和管理资源，下面我们将安装它
[root@k-m-1 ~]# wget "https://ghproxy.com/https://github.com/weaveworks/weave-gitops/releases/download/v0.32.0/gitops-$(uname)-$(uname -m).tar.gz"
[root@k-m-1 ~]# tar xf gitops-Linux-x86_64.tar.gz
[root@k-m-1 ~]# mv gitops /usr/local/bin/
[root@k-m-1 ~]# gitops version
Current Version: 0.32.0
GitCommit: 49a4249d8c205f14f0777c921cd69c04951e208f
BuildTime: 2023-09-13T17:23:13Z
Branch: releases/v0.32.0

# CLI 工具安装完成之后，我们就可以来部署 Weave GitOps 了
1：使用 GitOps CLI 工具生成 HelmRelease 和 HelmRepository 对象
2：创建一些登录凭据里访问 Dashboard
3：将生成的 yaml 提价到 Git 仓库
4：观察它们是否被同步到集群内

# 前面安装 Flux 的基础组件的仓库是 http://git.devops-engineer.com.cn/gitlayzer/fluxcd，我们需要克隆这个仓库到本地
[root@k-m-1 fluxcd]# git clone http://git.devops-engineer.com.cn/gitlayzer/fluxcd
Cloning into 'fluxcd'...
remote: Enumerating objects: 40, done.
remote: Counting objects: 100% (40/40), done.
remote: Compressing objects: 100% (28/28), done.
remote: Total 40 (delta 10), reused 0 (delta 0), pack-reused 0
Unpacking objects: 100% (40/40), done.
[root@k-m-1 fluxcd]# cd fluxcd/

# 然后使用如下命令创建 HelmRepository 和 HelmRelease 来部署 Weave GitOps
[root@k-m-1 fluxcd]# PASSWORD="gitlayzer"
[root@k-m-1 fluxcd]# gitops create dashboard weave-gitops \
--password $PASSWORD \
--export > clusters/dev-cluster/weave-gitops-dashboard.yaml
```

```yaml
---
apiVersion: source.toolkit.fluxcd.io/v1beta2
kind: HelmRepository
metadata:
  annotations:
    metadata.weave.works/description: This is the source location for the Weave GitOps
      Dashboard's helm chart.
  labels:
    app.kubernetes.io/component: ui
    app.kubernetes.io/created-by: weave-gitops-cli
    app.kubernetes.io/name: weave-gitops-dashboard
    app.kubernetes.io/part-of: weave-gitops
  name: weave-gitops
  namespace: flux-system
spec:
  interval: 1h0m0s
  type: oci
  url: oci://ghcr.io/weaveworks/charts
---
apiVersion: helm.toolkit.fluxcd.io/v2beta1
kind: HelmRelease
metadata:
  annotations:
    metadata.weave.works/description: This is the Weave GitOps Dashboard.  It provides
      a simple way to get insights into your GitOps workloads.
  name: weave-gitops
  namespace: flux-system
spec:
  chart:
    spec:
      chart: weave-gitops
      sourceRef:
        kind: HelmRepository
        name: weave-gitops
  interval: 1h0m0s
  values:
    adminUser:
      create: true
      passwordHash: $2a$10$BKhrHhFYBJi00LQ5EyIaMu.xHCr1zoJCjVZ1WpfD1/zUUwbYG2VyO
      username: admin
```

```shell
# 我们推送到 Git 仓库中，其实这里没有别的意思，就是将这个原始的代码托管到 Git 上，然后部署它
[root@k-m-1 fluxcd]# kubectl apply -f clusters/dev-cluster/weave-gitops-dashboard.yaml 
helmrepository.source.toolkit.fluxcd.io/weave-gitops created
helmrelease.helm.toolkit.fluxcd.io/weave-gitops created

# 检查部署
[root@k-m-1 fluxcd]# kubectl get pod,svc,ingress -n flux-system 
NAME                                               READY   STATUS    RESTARTS       AGE
pod/helm-controller-58695c7c56-vs6r2               1/1     Running   1 (2d5h ago)   4d19h
pod/image-automation-controller-78645c9469-gqpkn   1/1     Running   0              8h
pod/image-reflector-controller-8568b49675-j87g4    1/1     Running   0              39h
pod/kustomize-controller-859c949c64-nm57k          1/1     Running   1 (2d5h ago)   4d19h
pod/notification-controller-7d7747dd84-np6sz       1/1     Running   1 (2d5h ago)   4d19h
pod/source-controller-6d9b9567bf-fdfwb             1/1     Running   1 (2d5h ago)   4d19h
pod/weave-gitops-58fffd65bf-9zc7d                  1/1     Running   0              54s

NAME                              TYPE        CLUSTER-IP    EXTERNAL-IP   PORT(S)    AGE
service/notification-controller   ClusterIP   10.96.1.104   <none>        80/TCP     4d19h
service/source-controller         ClusterIP   10.96.1.78    <none>        80/TCP     4d19h
service/weave-gitops              ClusterIP   10.96.2.166   <none>        9001/TCP   54s
service/webhook-receiver          ClusterIP   10.96.3.156   <none>        80/TCP     4d19h

# 如果你想定制 Ingress，可以去官网看看如何定制 Ingress 的 values
# URL：https://docs.gitops.weave.works/docs/references/helm-reference/

[root@k-m-1 fluxcd]# kubectl port-forward svc/weave-gitops -n flux-system 9001:9001 --address 0.0.0.0
Forwarding from 0.0.0.0:9001 -> 9001
```

![flux-cd](https://picture.devops-engineer.com.cn/file/7366469e854824519316e.jpg)

![flux-cd](https://picture.devops-engineer.com.cn/file/0d301b47a7ba59041d3b8.jpg)

```shell
# 当然还有一个 Flamingo 是 Argo 的一个子系统，就是一个 ArgoCD 版本的子系统，不过这里就不赘述了，因为有点套娃的意思 
```

