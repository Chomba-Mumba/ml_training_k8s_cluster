# create iam role for service accounts
module "vpc_cni_irsa" {
    source = "terraform-aws-modules/iam/aws/modules/iam-role-for-service-accounts-eks"
    version = "~> 5.0"

    role_name_prefix = "VPC-CNI-IRSA"
    attach_vpc_cni_policy = true
    vpc_cni_enable_ipv4 = true
    attach_ebs_csi_policy = true

    oidc_providers = {
        efs = {
            provider_arn = 
            namespace_service_accounts = ["kube-system:efs-csi-controller-sa"]
        }
    }
}

resource "aws_eks_addon" "aws_efs_csi_driver" {

    cluster_name = aws_eks_cluster.eks_cluster.name
    addon_name = "aws-efs-csi-driver"
    addon_version = var.eks_addon_version_efs_csi_driver

    resolve_conflicts_on_create = "OVERWRITE"
    resolve_conflicts_on_update = "OVERWRITE"

    service_account_role_arn = module.vpc_cni_irsa.iam_role_arn

    configuration_values = jsonencode({
        controller = {
            tolerations : [
                {
                    key : "system",
                    operator : "Equal",
                    value : "owned",
                    effect : "NoSchedule"
                }
            ]
        }
    })

    preserve = true

    tags = {
        Name = "${var.cluster_name}_efs_addon"
    }
}