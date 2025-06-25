resource "aws_efs_file_system" "eks_efs" {
    creation_toke = "efs-eks-token"
    perfomance_mode = "generalPurpose"
    throughput_mode = "bursting"
    lifecycle_policy {
        transition_to_ia = "AFTER_30_DAYS"
    }
    tags = {
        Name = "${var.cluster_name}_efs"
    }
}

resource "aws_security_group" "efs_sg" {
    name = "eks_efs_sg"
    description = "Allow traffic from eks nodes only"
    vpc_id = aws_vpc.eks_vpc.id

    dynamic "ingress" {
        for_each = var.efs_sg_ports
        content {
            from_port = ingress.value["port"]
            to_port = ingress.value["port"]
            protocol = "tcp"
            security_groups = [aws_security_group.eks_node_sg.id]
        }
    }

    egress {
        from_port = 2049
        to_port = 2049
        protocol = "tcp"
        security_groups = [aws_security_group.eks_node_sg.id]
    }

    tags = {
        "Name" = "efs_sg"
    }
}

#mount target
resource "aws_efs_mount_target" "efs_mout_targets" {
    count = 2
    file_system_id = aws_efs_file_system.eks_efs.id
    subnet_id = local.public_subnets[count.index]
    security_groups = [aws_security_group.efs_sg.id]
}

#script for mounting efs
resource "null_resource" "generate_efs_mount_script" {
    provisioner "local-exec" {
        command = templatefile("efs_mount.tpl",{
            efs_mount_point = var.efs_mount_point
            file_system_id = local.file_system_id
        })
        interpreter = [
            "bash",
            "-c"
        ]
    }
}