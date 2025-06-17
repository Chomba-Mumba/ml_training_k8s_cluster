module "eks" {
    source = "terraform-aws-modules/eks/aws"
    version = "~> 20.31"

    cluster_name = "ml_training_cluster"
    cluster_version = "1.31"

    cluster_endpoint_public_access = true

    enable_cluster_creator_admin_permissions = true

    cluster_compute_config = {
        enabled = true
        node_pools = ["general-purpose"]
    }
    vpc_id = aws_vpc.ml_training_vpc.id
    subnet_ids = [aws_subnet.public_subnet.id]
    tags {
        Environment = "dev"
        Terraform = "true"
        Project = "ml_training"
    }
}