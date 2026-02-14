variable "REGISTRY" {
  default = ""
}

variable "IMAGE_NAME" {
  default = "click-trainer"
}

variable "TAG" {
  default = "latest"
}

group "default" {
  targets = ["app"]
}

target "app" {
  context    = "."
  dockerfile = "Dockerfile"
  platforms  = ["linux/amd64", "linux/arm64"]
  tags = [
    "${REGISTRY}${IMAGE_NAME}:${TAG}",
  ]
}

target "app-local" {
  inherits = ["app"]
  platforms = []
  output   = ["type=docker"]
}
