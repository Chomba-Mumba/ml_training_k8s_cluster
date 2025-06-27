output "cluster_endpoint" {
    description = "Endpoint for EKS control plane"
    value = aws_eks_cluster.eks_cluster.cluster_endpoint
}

output "cluster_security_group_id" {
    description = "Security Group ids attached to the cluster control plane"
    value = aws_security_group.eks_node_sg
}

output "cluster_name" {
    description = "EKS cluster name"
    value = aws_eks_cluster.eks_cluster.name
}