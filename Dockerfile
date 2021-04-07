FROM scratch
ARG GOARCH
LABEL maintainer="leonnicolas <leonloechner@gmx.de>"
COPY bin/linux/$GOARCH/webhook /opt/bin/
ENTRYPOINT ["/opt/bin/webhook"]
