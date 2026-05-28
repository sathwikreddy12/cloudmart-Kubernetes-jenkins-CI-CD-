pipeline {
    agent any

    environment {
        AWS_REGION = 'us-east-2'
        ECR_REGISTRY = '231063060932.dkr.ecr.us-east-2.amazonaws.com'
        IMAGE_TAG = "${BUILD_NUMBER}"
    }

    stages {

        stage('Checkout') {
            steps {
                echo 'Cloning repository...'
                checkout scm
            }
        }

        stage('Build Docker Images') {
            steps {
                echo 'Building Docker images...'
                sh '''
                    docker build -t ${ECR_REGISTRY}/cloudmart/user-service:${IMAGE_TAG} ./user_service
                    docker build -t ${ECR_REGISTRY}/cloudmart/product-service:${IMAGE_TAG} ./product_service
                    docker build -t ${ECR_REGISTRY}/cloudmart/order-service:${IMAGE_TAG} ./order_service
                '''
            }
        }

        stage('Push to ECR') {
            steps {
                echo 'Pushing images to ECR...'
                withAWS(credentials: 'aws-credentials', region: "${AWS_REGION}") {
                    sh '''
                        aws ecr get-login-password --region ${AWS_REGION} | \
                        docker login --username AWS --password-stdin ${ECR_REGISTRY}

                        docker push ${ECR_REGISTRY}/cloudmart/user-service:${IMAGE_TAG}
                        docker push ${ECR_REGISTRY}/cloudmart/product-service:${IMAGE_TAG}
                        docker push ${ECR_REGISTRY}/cloudmart/order-service:${IMAGE_TAG}
                    '''
                }
            }
        }

        stage('Deploy to EKS') {
            steps {
                echo 'Deploying to EKS...'

                withAWS(credentials: 'aws-credentials', region: "${AWS_REGION}") {
                    sh '''
                        aws eks update-kubeconfig \
                            --region ${AWS_REGION} \
                            --name cloudmart-eks

                        kubectl get nodes

                        kubectl set image deployment/user-service \
                            user-service=${ECR_REGISTRY}/cloudmart/user-service:${IMAGE_TAG} \
                            --namespace=default

                        kubectl set image deployment/product-service \
                            product-service=${ECR_REGISTRY}/cloudmart/product-service:${IMAGE_TAG} \
                            --namespace=default

                        kubectl set image deployment/order-service \
                            order-service=${ECR_REGISTRY}/cloudmart/order-service:${IMAGE_TAG} \
                            --namespace=default

                        kubectl rollout status deployment/user-service --namespace=default
                        kubectl rollout status deployment/product-service --namespace=default
                        kubectl rollout status deployment/order-service --namespace=default
                    '''
                }
            }
        }
    }

    post {
        success {
            echo 'Pipeline completed successfully! New version deployed to EKS.'
        }
        failure {
            echo 'Pipeline failed! Check the logs above.'
        }
    }
}
