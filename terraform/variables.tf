variable "aws_region" {
  type = string
}

variable "ecr_registry" {
  type    = string
}

variable "image_tag" {
  type    = string
  default = "latest"
}

variable "cluster_name" {
  type    = string
  default = "ml_training_cluster"
}

variable "ami_id" {
    type = string
    default = "ami-00f7e79ebcafba5e4"
}