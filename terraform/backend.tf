terraform {
  backend "s3" {
    bucket  = "mrembiasz-blog-terraform-state"
    key     = "mrembiasz-blog/aws.tfstate"
    region  = "eu-central-1"
    encrypt = true
  }
}
