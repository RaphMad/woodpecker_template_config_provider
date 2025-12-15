FROM alpine AS dev

RUN apk add --no-cache go \
                       make
COPY . /src/
RUN cd /src && \
    go mod init woodpecker_template_config_provider && \
    go mod tidy


FROM dev AS build
RUN cd /src/ && \
    make


FROM scratch

COPY --from=build /out/woodpecker_template_config_provider /woodpecker_template_config_provider

HEALTHCHECK ["/woodpecker_template_config_provider", "ping"]
ENTRYPOINT ["/woodpecker_template_config_provider"]
