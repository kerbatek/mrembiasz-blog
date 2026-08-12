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
    - name: terraform
      image: hashicorp/terraform:1.13
      command:
        - cat
      tty: true
'''
    }
  }

  environment {
    AWS_ROLE_ARN = 'arn:aws:iam::047588357922:role/mrembiasz-blog-jenkins-deploy'
    TF_IN_AUTOMATION = 'true'
  }

  stages {
    stage('Build Astro') {
      steps {
        sh 'npm ci'
        sh 'npm run build'
      }
    }

    stage('Terraform Plan') {
      steps {
        container('terraform') {
          withCredentials([file(credentialsId: 'aws-oidc-token', variable: 'AWS_WEB_IDENTITY_TOKEN_FILE')]) {
            dir('terraform') {
              sh 'terraform init -input=false'
              sh 'terraform fmt -check'
              sh 'terraform validate'
              sh 'terraform plan -input=false -out=tfplan'
              sh 'terraform show -no-color tfplan > tfplan.txt'
            }
          }
        }
      }
    }

    stage('Terraform Apply') {
      when {
        branch 'main'
      }
      steps {
        container('terraform') {
          withCredentials([file(credentialsId: 'aws-oidc-token', variable: 'AWS_WEB_IDENTITY_TOKEN_FILE')]) {
            dir('terraform') {
              sh 'terraform apply -input=false tfplan'
            }
          }
        }
      }
    }
  }

  post {
    success {
      archiveArtifacts artifacts: 'dist/**, terraform/tfplan, terraform/tfplan.txt', fingerprint: true
    }
  }
}
