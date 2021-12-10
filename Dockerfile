FROM alpine:3.6

RUN apk add --no-cache ca-certificates
ADD bin/netlox-cloud-controller-manager /bin/
CMD ["/bin/netlox-cloud-controller-manager"]