resource "aws_iam_role" "eks_cluster_role" {
  name = "${var.cluster_name}_role"
  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Action = [
          "sts:AssumeRole",
          "sts:TagSession"
        ]
        Effect = "Allow"
        Principal = {
          Service = "eks.amazonaws.com"
        }
      }
    ]
  })
}

resource "aws_eks_cluster" "eks_cluster" {
  name     = var.cluster_name
  role_arn = aws_iam_role.eks_cluster_role.arn

  access_config {
    authentication_mode = "API"
  }

  version = "1.31"

  vpc_config {
    subnet_ids = aws_subnet.eks_public_subnets[*].id
  }

  #ensure iam permissions created before cluster
  depends_on = [
    aws_iam_role_policy_attachment.eks_cluster_policy,
  ]

  #after cluster is provisioned configure kubectl
  provisioner "local-exec" {
    command = "aws eks update-kubeconfig --name ${self.name} --region ${var.aws_region}"
  }
}

resource "aws_security_group" "eks_node_sg" {
    name = "eks_node_sg"
    description = "Allow traffic for eks nodes"
    vpc_id = aws_vpc.eks_vpc.id

    ingress {
        from_port = 0
        to_port = 65534
        protocol = "tcp"
        self = true #allow node-to-node traffic
    }

    #TODO - Add rules to allow for other AWS services, etc
    egress {
        from_port = 2049
        to_port = 2049
        protocol = "tcp"
        security_groups = [aws_security_group.efs_sg.id]
    }
}

resource "aws_iam_role" "eks_node_role" {
  name = "${var.cluster_name}_node_role"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Action = "sts:AssumeRole"
        Effect = "Allow"
        Principal = {
          Service = "ec2.amazonaws.com"
        }
      }
    ]
  })
}

resource "aws_iam_role_policy_attachment" "eks_worker_node_policy" {
  policy_arn = "arn:aws:iam::aws:policy/AmazonEKSWorkerNodePolicy"
  role       = aws_iam_role.eks_node_role.name
}

resource "aws_iam_role_policy_attachment" "eks_cni_policy" {
  policy_arn = "arn:aws:iam::aws:policy/AmazonEKS_CNI_Policy"
  role       = aws_iam_role.eks_node_role.name
}

resource "aws_iam_role_policy_attachment" "eks_container_register_policy" {
  policy_arn = "arn:aws:iam::aws:policy/AmazonEC2ContainerRegistryReadOnly"
  role       = aws_iam_role.eks_node_role.name
}

resource "aws_iam_role_policy_attachment" "eks_cluster_policy" {
  policy_arn = "arn:aws:iam::aws:policy/AmazonEKSClusterPolicy"
  role       = aws_iam_role.eks_cluster_role.name
}