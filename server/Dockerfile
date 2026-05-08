# server/Dockerfile

# 默认保持 linux/amd64 运行时；若后续需要其他架构，统一覆盖平台与 Go 目标参数。
ARG BUILDPLATFORM=linux/amd64
ARG RUNTIMEPLATFORM=linux/amd64
ARG GO_TARGETOS=linux
ARG GO_TARGETARCH=amd64
ARG NODE_BUILDER_IMAGE=node:24.14.0
ARG GO_BUILDER_IMAGE=golang:1.25.9
ARG RUNTIME_BASE_IMAGE=alpine:3.21

# ======================
# 1) 构建前端静态文件
# ======================
# 与根目录 .n-node-version 对齐，避免容器与本地 Node 小版本漂移；同时保留 build arg，便于在受限网络下切换镜像源。
FROM --platform=$BUILDPLATFORM ${NODE_BUILDER_IMAGE} AS web-builder

WORKDIR /web

COPY web/package.json web/pnpm-lock.yaml ./
RUN npm install -g pnpm@10.13.1 && \
    pnpm config set store-dir /web/.pnpm-store && \
    pnpm config set registry https://registry.npmmirror.com && \
    pnpm install --frozen-lockfile

COPY web/src ./src
COPY web/public ./public
COPY web/index.html ./
COPY web/vite.config.mjs ./
COPY web/tailwind.config.js ./
COPY web/postcss.config.js ./
# 关键兜底：前端 prebuild 依赖服务端错误码真源与生成脚本，镜像构建时必须一并带入。
COPY scripts /scripts
COPY server/internal/errcode /server/internal/errcode
RUN pnpm run build   # 输出到 web/build

# ======================
# 2) Builder 阶段
# ======================
FROM --platform=$BUILDPLATFORM ${GO_BUILDER_IMAGE} AS builder

ARG GO_TARGETOS
ARG GO_TARGETARCH

WORKDIR /src

ADD server/go.mod ./go.mod
ADD server/go.sum ./go.sum

RUN go env -w GOPROXY=https://goproxy.cn,direct

RUN --mount=type=cache,id=gomod,target=/go/pkg/mod \
    --mount=type=cache,id=gobuild,target=/root/.cache/go-build \
    go mod download

COPY server/ /src

ENV CGO_ENABLED=0

ARG GIT_SHA
ARG GIT_SHA_SHORT
ARG BUILD_TIME
ARG IMAGE_TAG

ENV GIT_SHA=${GIT_SHA}
ENV GIT_SHA_SHORT=${GIT_SHA_SHORT}
ENV BUILD_TIME=${BUILD_TIME}
ENV IMAGE_TAG=${IMAGE_TAG}
ENV TZ=Asia/Shanghai

# Go 二进制跟随目标参数，避免镜像架构与可执行文件不一致。
RUN --mount=type=cache,id=gomod,target=/go/pkg/mod \
    --mount=type=cache,id=gobuild,target=/root/.cache/go-build \
    mkdir -p bin && \
    GOOS=${GO_TARGETOS} GOARCH=${GO_TARGETARCH} go build \
      -ldflags "-X main.Version=${GIT_SHA}" \
      -o ./bin/server ./cmd/server

# ======================
# 3) Runtime 阶段
# ======================
FROM --platform=$RUNTIMEPLATFORM ${RUNTIME_BASE_IMAGE}

ARG GIT_SHA
ARG GIT_SHA_SHORT
ARG BUILD_TIME
ARG IMAGE_TAG

ENV GIT_SHA=${GIT_SHA}
ENV GIT_SHA_SHORT=${GIT_SHA_SHORT}
ENV BUILD_TIME=${BUILD_TIME}
ENV IMAGE_TAG=${IMAGE_TAG}
ENV TZ=Asia/Shanghai

RUN apk add --no-cache tzdata ca-certificates

WORKDIR /app

COPY --from=builder /src/bin/server /app/server
COPY --from=builder /src/configs/prod /app/configs
COPY --from=web-builder /web/build /app/public

EXPOSE 8400
EXPOSE 9400

# 默认用 /app/configs （里面只有 prod/config.yaml）
CMD ["/app/server", "-conf", "/app/configs"]
