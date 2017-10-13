# --- BUILD
FROM golang:1.9

WORKDIR /go/src/github.com/sprungknoedl/kogia
COPY . .

# static compilation for linux
ENV CGO_ENABLED=0
ENV GOOS=linux
RUN go build 

# --- RUN
FROM scratch

WORKDIR /
ENTRYPOINT ["/kogia"]

VOLUME "/kogia.yml"
VOLUME "/run/docker.sock"

COPY --from=0 /go/src/github.com/sprungknoedl/kogia/kogia /kogia
