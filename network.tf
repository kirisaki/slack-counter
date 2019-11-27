resource "aws_vpc" "vpc" {
  cidr_block           = var.local_cidr
  enable_dns_support   = true
  enable_dns_hostnames = true
}

resource "aws_subnet" "subnets" {
  count  = length(var.az_suffixies)
  vpc_id = aws_vpc.vpc.id

  cidr_block        = cidrsubnet(aws_vpc.vpc.cidr_block, 8, count.index * 2)
  availability_zone = "${var.region}${var.az_suffixies[count.index]}"
}

resource "aws_internet_gateway" "gw" {
  vpc_id = aws_vpc.vpc.id
}

resource "aws_eip" "eip" {
  vpc = true
}
