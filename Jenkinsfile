podTemplate(cloud: "kubernetes", containers: [
    containerTemplate(name: 'golang', image: 'golang:1.21.1-alpine3.18', command: 'cat', ttyEnabled: true),
    containerTemplate(name: 'docker', image: 'docker:latest', command: 'cat', ttyEnabled: true),
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
