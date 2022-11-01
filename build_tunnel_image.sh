IMAGE_REPO=edge-paas-registry.cn-hangzhou.cr.aliyuncs.com/edgepaas
IMAGE_TAG=internal
TARGET_PLATFORMS=linux/amd64,linux/arm64


docker buildx rm tunnel-server-container-builder || true
docker buildx create --use --name=tunnel-server-container-builder
# enable qemu for arm64 build
# https://github.com/docker/buildx/issues/464#issuecomment-741507760
docker run --privileged --rm tonistiigi/binfmt --uninstall qemu-aarch64
docker run --rm --privileged tonistiigi/binfmt --install all
docker buildx build --no-cache --push  --platform ${TARGET_PLATFORMS} -f hack/dockerfiles/Dockerfile.yurt-tunnel-server . -t ${IMAGE_REPO}/yurt-tunnel-server:${IMAGE_TAG}
docker buildx build --no-cache --push  --platform ${TARGET_PLATFORMS} -f hack/dockerfiles/Dockerfile.yurt-tunnel-agent . -t ${IMAGE_REPO}/yurt-tunnel-agent:${IMAGE_TAG}
