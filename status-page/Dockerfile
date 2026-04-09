ARG GO_IMAGE=golang:1.26.2-alpine3.22
ARG RUNTIME_IMAGE=alpine:3.22

FROM ${GO_IMAGE} AS build

WORKDIR /src

COPY go.mod ./
COPY go.sum* ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -trimpath -ldflags="-s -w" -o /out/app .

FROM ${RUNTIME_IMAGE}

WORKDIR /srv

RUN addgroup -S app && adduser -S -G app app

COPY --from=build /out/app /srv/app
COPY --from=build /src/templates /srv/templates
COPY --from=build /src/static /srv/static
COPY --from=build /src/data /srv/data

RUN mkdir -p /srv/data && chown -R app:app /srv

USER app:app

EXPOSE 8080

VOLUME ["/srv/data"]

CMD ["/srv/app"]