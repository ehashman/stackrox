FROM alpine:3.14

RUN mkdir /stackrox-data

RUN apk update && \
    apk add --no-cache \
        openssl zip \
        && \
    apk --purge del apk-tools \
    ;

RUN mkdir -p /stackrox-data/cve/istio && \
    wget -O /stackrox-data/cve/istio/checksum "https://definitions.stackrox.io/cve/istio/checksum" && \
    wget -O /stackrox-data/cve/istio/cve-list.json "https://definitions.stackrox.io/cve/istio/cve-list.json"

RUN mkdir -p /stackrox-data/external-networks && \
    latest_prefix="$(wget -q https://definitions.stackrox.io/external-networks/latest_prefix -O -)" && \
    wget -O /stackrox-data/external-networks/checksum "https://definitions.stackrox.io/${latest_prefix}/checksum" && \
    wget -O /stackrox-data/external-networks/networks "https://definitions.stackrox.io/${latest_prefix}/networks" && \
    test -s /stackrox-data/external-networks/checksum && test -s /stackrox-data/external-networks/networks

RUN zip -jr /stackrox-data/external-networks/external-networks.zip /stackrox-data/external-networks

COPY ./docs/api/v1/swagger.json /stackrox-data/docs/api/v1/swagger.json
