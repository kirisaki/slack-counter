resource "aws_lb" "alb" {
  name               = "${var.service}-alb"
  internal           = false
  load_balancer_type = "application"
  subnets            = aws_subnet.subnets.*.id
}

resource "aws_lb_target_group" "group" {
  name     = "${var.service}-target"
  port     = 80
  protocol = "HTTP"
  vpc_id   = aws_vpc.vpc.id
}

resource "aws_lb_listener" "listener" {
  load_balancer_arn = aws_lb.alb.arn
  port              = "443"
  protocol          = "HTTPS"
  ssl_policy        = "ELBSecurityPolicy-2016-08"
  certificate_arn   = var.cert_arn

  default_action {
    type             = "forward"
    target_group_arn = aws_lb_target_group.group.arn
  }
}
