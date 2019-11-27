output "alb" {
  value = {
    dns_name         = aws_lb.alb.dns_name
    arn              = aws_lb.alb.arn
    target_group_arn = aws_lb_target_group.group.arn
  }
}
