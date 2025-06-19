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

resource "aws_iam_role_policy_attachment" "eks_cluster_policy" {
    policy_arn = "arn:aws:iam::aws:policy/AmazonEKSClusterPolicy"
    role = aws_iam_role.eks_cluster_role.name
}

resource "aws_eks_cluster" "cluster" {
    name = var.cluster_name
    role_arn = aws_iam_role.ml_training_cluster_role.arn

    access_config {
        authentication_mode = "API"
    }
    
    version = "1.31"

    vpc_config{
        subnet_ids = aws_subnet.public_subnets[*].id
    }

    #ensure iam permissions created before cluster
    depends_on = [
        aws_iam_role_policy_attachment.eks_cluster_policy,
    ]
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
    role = aws_iam_role.eks_node_role.name
}

resource "aws_iam_role_policy_attachment" "eks_cni_policy" {
    policy_arn = "arn:aws:iam::aws:policy/AmazonEKS_CNI_Policy"
    role = aws_iam_role.eks_node_role.name
}

resource "aws_iam_role_policy_atachment" "eks_container_register_policy" {
    policy_arn = "arn:aws:iam::aws:policy/AmazonEC2ContainerRegistryReadOnly"
    role = aws_iam_role.eks_node_role.name
}