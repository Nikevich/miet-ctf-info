FROM --platform=$BUILDPLATFORM golang:1.24-alpine AS build

ARG TARGETOS
ARG TARGETARCH

WORKDIR /src
COPY go.mod ./
COPY main.go ./
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH \
    go build -trimpath -ldflags="-s -w" -o /out/cookie-shop ./main.go

FROM alpine:3.21

RUN addgroup -S appgroup && adduser -S -G appgroup appuser
WORKDIR /app
COPY --from=build /out/cookie-shop ./cookie-shop
COPY --chown=appuser:appgroup templates ./templates
COPY --chown=appuser:appgroup static ./static
USER appuser
EXPOSE 8080
CMD ["./cookie-shop"]