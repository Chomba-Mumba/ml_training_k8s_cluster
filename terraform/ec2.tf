data "aws_ssm_parameter" "eks_ami" {
  name = "/aws/service/eks/optmized-ami/${aws_eks_cluster.eks_cluster.version}/amazon-linux-2/recommended/image_id"
}

resource "aws_launch_template" "eks_nodes_lt" {
  name                   = "${var.cluster_name}_node_template"
  instance_type          = "t3.micro" #TODO - decide on ami
  image_id               = data.aws_ssm_parameter.eks_ami.value
  vpc_security_group_ids = [aws_eks_cluster.eks_cluster.vpc_config[0].cluster_security_group_id]

  user_data = base64encode(<<-EOF
        #!/bin/bash
        set -o xtrace
        /etc/eks/bootstrap.sh ${var.cluster_name}
    EOF
  )

  tag_specifications {
    resource_type = "instance"
    tags = {
      Name = "${var.cluster_name}_node"
    }
  }
}

resource "aws_autoscaling_group" "eks_nodes" {
  name                = "${var.cluster_name}_nodes"
  desired_capacity    = 2
  max_size            = 3
  min_size            = 1
  target_group_arns   = []
  vpc_zone_identifier = aws_subnet.eks_public_subnets[*].id

  launch_template {
    id      = aws_launch_template.eks_nodes_lt.id
    version = "$latest"
  }

  tag {
    key                 = "kubernetes.io/cluster/${var.cluster_name}"
    value               = "owned"
    propagate_at_launch = true
  }
}