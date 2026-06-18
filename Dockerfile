FROM scratch
ARG TARGETPLATFORM
COPY $TARGETPLATFORM/helm-sops /helm
COPY _helm /_helm
