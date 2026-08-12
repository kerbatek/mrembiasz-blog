pipeline {
  agent {
    kubernetes {
      defaultContainer 'node'
      yaml '''
apiVersion: v1
kind: Pod
spec:
  containers:
    - name: node
      image: node:24-alpine
      command:
        - cat
      tty: true
'''
    }
  }

  stages {
    stage('Build Astro') {
      steps {
        sh 'npm ci'
        sh 'npm run build'
      }
    }
  }

  post {
    success {
      archiveArtifacts artifacts: 'dist/**', fingerprint: true
    }
  }
}
