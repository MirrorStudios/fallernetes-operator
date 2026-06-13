FROM golang:1.26.4 as builder

WORKDIR /workspace

COPY operator/go.mod operator/go.sum operator/
COPY service/go.mod service/go.sum service/

WORKDIR /workspace/service
RUN go mod download

WORKDIR /workspace
COPY operator/ operator/
COPY service/ service/

WORKDIR /workspace/service
RUN CGO_ENABLED=0 GOARCH=amd64 go build -a -o service cmd/main.go

FROM gcr.io/distroless/static:nonroot
WORKDIR /
COPY --from=builder /workspace/service/service .
USER 65532:65532

ENTRYPOINT ["/service"]
