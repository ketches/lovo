FROM alpine

ARG TARGETOS TARGETARCH

WORKDIR /

COPY bin/${TARGETOS}/${TARGETARCH}/lovo-provisioner .

ENTRYPOINT [ "./lovo-provisioner" ]