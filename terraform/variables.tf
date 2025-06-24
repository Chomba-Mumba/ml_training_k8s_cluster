variable "aws_region" {
  type = string
}

variable "ecr_registry" {
  type    = string
  default = "${aws_acc}.dkr.ecr.${aws_region}.amazonaws.com"
}

variable "image_tag" {
  type    = string
  default = "latest"
}

variable "cluster_name" {
  type    = string
  default = "ml_training_cluster"
}