variable "role_arn" {}

variable "region" {
  default = "ap-northeast-1"
}

variable "local_cidr" {
  default = "10.2.0.0/16"
}

variable "az_suffixies" {
  default = ["a", "c"]
}

variable "service" {
  default = "slack-counter"
}

variable "domain" {}

variable "cert_arn" {}
