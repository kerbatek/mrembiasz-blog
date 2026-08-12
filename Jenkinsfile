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
    stage('Install dependencies') {
      steps {
        sh 'node --version'
        sh 'npm ci'
      }
    }

    stage('Build Astro') {
      steps {
        sh 'npm run build'
      }
    }
  }

  post {
    success {
      echo 'Astro build passed; dist artifact archived.'
      archiveArtifacts artifacts: 'dist/**', fingerprint: true
    }
  }
}
