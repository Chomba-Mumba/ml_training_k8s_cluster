eks_managed_node_groups = {
    one = {
        name = "node-group-1"
        instance_types = ["t3.small"]

        min_size = 1
        max_size = 3
        desired_size = 2
    }

    two = {
        name = "node-group-2"
        instance_types = ["t3.small"]'

        min_size = 1
        max_size = 3
        desired_size = 2
    }
}
resource "aws_eks_cluster" "ml_training_cluster" {
    name = "ml_training_cluster"

    access_config {
        authentication_mode = "API"
    }
    role_arn = aws_iam_role.ml_training_cluster_role.arn
    version = "1.31"

    vpc_config{
        subnet_ids = [
            aws_subnet.public_subnet.id
        ]
    }

    #ensure iam permissions created before cluster
    depends_on = [
        aws_iam_role_policy_attachment.cluster_AmazonEKSClusterPolicy,
    ]
}

resource "aws_iam_role" "ml_training_cluster_role" {
    name = "ml_training_cluster_role"
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

resource "aws_iam_role_policy_attachment" "cluster_AmazonEKSClusterPolicy" {
    policy_arn = "arn:aws:iam::aws:policy/AmazonEKSClusterPolicy"
    role = aws_iam_role.ml_training_cluster.name
}



module "eks_al2" {
    source = "terraform-aws-modules/eks/aws"
    version = "~> 20.0"

    cluster_name = "ml_training_cluster"
    cluster_version = "1.31"

    cluster_endpoint_public_access = true

    enable_cluster_creator_admin_permissions = true

    cluster_compute_config = {
        enabled = true
        node_pools = ["general-purpose"]
    }
    
    subnet_ids = [aws_subnet.public_subnet.id]
    tags {
        Environment = "dev"
        Terraform = "true"
        Project = "ml_training"
    }
}