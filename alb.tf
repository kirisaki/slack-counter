resource "aws_lb" "alb" {
  name = "slack-counter-alb"
  internal = false
  load_balancer_type = "application"
  security_groups = [aws_security_group.web.id]
  subnets = [aws_subnet.pub.id]

}
