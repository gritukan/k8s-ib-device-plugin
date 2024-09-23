FROM ubuntu:focal

COPY k8s-ib-device-plugin /usr/local/bin

ENTRYPOINT ["k8s-ib-device-plugin"]
