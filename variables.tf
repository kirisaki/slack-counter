variable "role_arn" {
  type = string
}

variable "region" {
  type = string
  default = "ap-northeast-1"
}

variable "az_suffix" {
  type = string
  default = "c"
}

variable "local_cidr" {
  type = string
  default = "10.2.0.0/16"
}

variable "domain" {
  type = string
}

variable "ssh_port" {
  type = number
}

variable "ami" {
  type = string
  default = "ami-068a6cefc24c301d2"
}
variable "key_file" {
  type = string
}

variable "key_name" {
  type = string
}

variable "zone_id" {
  type = string
}
