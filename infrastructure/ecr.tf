resource "aws_ecr_repository" "lucksacks" {
  name                 = "lucksacks"
  image_tag_mutability = "MUTABLE"

  image_scanning_configuration {
    scan_on_push = true
  }
}
