FROM golang:1.23-alpine AS build

WORKDIR /src
COPY go.mod ./
COPY cmd ./cmd
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/training-login ./cmd/web

FROM scratch
COPY --from=build /out/training-login /training-login
EXPOSE 8080
USER 65532:65532
ENTRYPOINT ["/training-login"]
