resource "aws_vpc" "vpc" {
  cidr_block           = var.local_cidr
  enable_dns_support   = true
  enable_dns_hostnames = true
}
