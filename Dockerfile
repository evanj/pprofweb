# Extract graphviz and dependencies
FROM golang:1.14.3-buster AS deb_extractor
RUN cd /tmp && \
    apt-get update && apt-get download \
        graphviz libgvc6 libcgraph6 libltdl7 libxdot4 libcdt5 libpathplan4 libexpat1 zlib1g && \
    mkdir /dpkg && \
    for deb in *.deb; do dpkg --extract $deb /dpkg || exit 10; done

FROM golang:1.14.3-buster AS builder
COPY go.mod go.sum pprofweb.go /go/src/pprofweb/
WORKDIR /go/src/pprofweb
RUN go build --mod=readonly pprofweb.go

FROM gcr.io/distroless/base-debian10:latest AS run
COPY --from=builder /go/src/pprofweb/pprofweb /pprofweb
COPY --from=deb_extractor /dpkg /
# Configure dot plugins
RUN ["dot", "-c"]

# Use a non-root user: slightly more secure (defense in depth)
USER nobody
WORKDIR /
EXPOSE 8080
ENTRYPOINT ["/pprofweb"]
