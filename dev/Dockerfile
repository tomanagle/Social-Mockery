FROM golang:1.26-bookworm AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=1 go build -trimpath -o /out/social-mockery ./cmd

FROM debian:bookworm-slim
RUN apt-get update \
    && apt-get install -y --no-install-recommends ca-certificates \
    && rm -rf /var/lib/apt/lists/*
RUN groupadd --gid 10001 simulator \
    && useradd --uid 10001 --gid simulator --no-create-home --shell /usr/sbin/nologin simulator \
    && mkdir -p /app/profiles /data \
    && chown simulator:simulator /data
WORKDIR /app
COPY --from=build /out/social-mockery /usr/local/bin/social-mockery
COPY profiles/example.yaml /app/profiles/example.yaml
USER 10001:10001
EXPOSE 3000
ENTRYPOINT ["social-mockery"]
CMD ["start", "--addr", ":3000"]
