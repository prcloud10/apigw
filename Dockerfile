
ARG GO_VERSION=1.17.6

FROM golang:${GO_VERSION}-alpine AS build
RUN apk add --no-cache git
WORKDIR /src
COPY ./go.mod ./go.sum ./
RUN go mod download
COPY ./ ./
RUN CGO_ENABLED=0 go build -installsuffix 'static' -o /app ./main.go
 


FROM gcr.io/distroless/static AS final
LABEL maintainer="luis"
USER nonroot:nonroot
COPY --from=build --chown=nonroot:nonroot /app /app
ENTRYPOINT ["/app"]
