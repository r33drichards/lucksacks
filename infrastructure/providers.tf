terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = ">= 3.20.0"
    }
  }

  required_version = "~> 1.0"
  backend "http" {

  }
}
provider "aws" {
  region = "us-east-2"
}
