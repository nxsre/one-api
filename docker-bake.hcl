# docker buildx bake
#
# 默认构建完整运行镜像（最终 stage：runtime）：
#   docker buildx bake image
#
# 仅构建某一前端阶段（便于预热缓存或排查）：
#   docker buildx bake build-nacos-console
#   docker buildx bake build-one-api-web
#
# 变量示例：
#   IMAGE_NAME=registry.example.com/one-api TAG=v1 docker buildx bake image

variable "IMAGE_NAME" {
  default = "one-api"
}

variable "TAG" {
  default = "local"
}

target "build-nacos-console" {
  context = "."
  dockerfile = "Dockerfile"
  target = "build-nacos-console"
}

target "build-one-api-web" {
  context = "."
  dockerfile = "Dockerfile"
  target = "build-one-api-web"
}

target "builder-backend" {
  context = "."
  dockerfile = "Dockerfile"
  target = "builder-backend"
}

target "image" {
  context = "."
  dockerfile = "Dockerfile"
  target = "runtime"
  tags = ["${IMAGE_NAME}:${TAG}"]
}

group "default" {
  targets = ["image"]
}
