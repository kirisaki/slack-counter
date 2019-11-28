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

resource "aws_nat_gateway" "nat_gw" {
  allocation_id = aws_eip.eip.id
  subnet_id     = aws_subnet.subnets.0.id
  depends_on    = [aws_internet_gateway.gw]
}

resource "aws_route_table" "route" {
  vpc_id = aws_vpc.vpc.id

  route {
    cidr_block = "0.0.0.0/0"
    gateway_id = aws_internet_gateway.gw.id
  }
}

resource "aws_route_table_association" "pub" {
  count          = length(var.az_suffixies)
  route_table_id = aws_route_table.route.id
  subnet_id      = element(aws_subnet.subnets.*.id, count.index)
}
