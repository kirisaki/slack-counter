resource "aws_vpc" "vpc" {
  cidr_block           = var.local_cidr
  enable_dns_support   = true
  enable_dns_hostnames = true
}

resource "aws_subnet" "pub" {
  vpc_id = aws_vpc.vpc.id

  cidr_block        = cidrsubnet(aws_vpc.vpc.cidr_block, 8, 0)
  availability_zone = "${var.region}${var.az_suffix}"
}

resource "aws_internet_gateway" "gw" {
  vpc_id = aws_vpc.vpc.id
}

resource "aws_eip" "eip" {
  vpc = true
}

resource "aws_eip_association" "assoc" {
  instance_id = aws_instance.server.id
  allocation_id = aws_eip.eip.id
}

resource "aws_route_table" "pub" {
  vpc_id = aws_vpc.vpc.id

  route {
    cidr_block = "0.0.0.0/0"
    gateway_id = aws_internet_gateway.gw.id
  }
}

resource "aws_route_table_association" "pub" {
  route_table_id = aws_route_table.pub.id
  subnet_id      = aws_subnet.pub.id
}

