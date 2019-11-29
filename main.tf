provider "aws" {
  region = var.region
  assume_role {
    role_arn     = var.role_arn
  }
}

terraform {
  backend "s3" {
    bucket  = "slack-counter-tfstate"
    region  = "ap-northeast-1"
    key     = "terraform.tfstate"
    encrypt = true
  }
}
