resource "aws_ecr_repository" "ml_ecr_repo"{
    name = "ml_ecr_repo"
    image_tag_mutability = "MUTABLE"
    image_scanning_configuration {
        scan_on_push = true
    }
}

resource "aws_ecr_lifecycle_policy" "my_policy" {
  repository = aws_ecr_repository.ml_ecr_repo.name

  policy = jsonencode({
    rules = [
      {
        rule_priority = 1
        description   = "Keep only 5 images"
        selection     = {
          count_type        = "imageCountMoreThan"
          count_number      = 5
          tag_status        = "tagged"
          tag_prefix_list   = ["prod"]
        }
        action = {
          type = "expire"
        }
      }
    ]
  })
}