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
        sh 'echo "intentional failure to test GitHub status" && exit 1'
      }
    }
  }

  post {
    success {
      echo 'Astro build passed; dist artifact archived.'
      githubNotify context: 'astro/build',
        description: 'Astro build passed; dist artifact archived.',
        status: 'SUCCESS'
      archiveArtifacts artifacts: 'dist/**', fingerprint: true
    }
    failure {
      githubNotify context: 'astro/build',
        description: 'Astro build failed.',
        status: 'FAILURE'
    }
  }
}
