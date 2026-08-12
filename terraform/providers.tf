provider "aws" {
  region = "eu-central-1"
}

provider "aws" {
  alias  = "use1"
  region = "us-east-1"
}
