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