provider "helm" {
    kubernetes {
        host = data.aws_eks_cluster.cluster.endpoint
        cluster_ca_certificate = base64encode(data.aws_eks_cluster.cluster.certificate_authority.0.data)
        exec {
            api_version = "client.authentication.k8s.io/v1beta1"
            args = ["eks", "get-token", "--cluster-name", data.aws_eks_cluster.cluster.name]
            command = "aws"
        }
    }
}

resource "helm_release" "efs_controller" {
    name = "aws-efs-csi-driver"
    chart = "aws-efs-csi-driver"
    repository = "https://kubernetes-sigs.github.io/aws-efs-csi-driver/"
    version = "2.3.6"
    namespace = "kube-system"

    values = [
        <<EOF
        clusterName: ${var.cluster_name}
        controller:
            create: true
            deleteAccessPointRootDir: true # automatically delete data on EFS when PVC removed
            serviceAccount:
                name: efs-csi-controller-sa
                annotations:
                eks.amazonaws.com/role-arn: ${module.efs_csi_iam_assumable_role.iam_role_arn}
            tags:
                efs.csi.aws.com/cluster: ${var.cluster_name}
            node:
                serviceAccount:
                    name: efs-csi-node-sa
                    annotations:
                        eks.amazonaws.com/role-arn: ${module.efs_csi_iam_assumable_role.iam_role_arn}
            EOF
    ]
}