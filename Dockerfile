# syntax=docker/dockerfile:1

# ▄▄▄▄    █    ██  ██▓ ██▓    ▓█████▄ ▓█████  ██▀███  
# ▓█████▄  ██  ▓██▒▓██▒▓██▒    ▒██▀ ██▌▓█   ▀ ▓██ ▒ ██▒
# ▒██▒ ▄██▓██  ▒██░▒██▒▒██░    ░██   █▌▒███   ▓██ ░▄█ ▒
# ▒██░█▀  ▓▓█  ░██░░██░▒██░    ░▓█▄   ▌▒▓█  ▄ ▒██▀▀█▄  
# ░▓█  ▀█▓▒▒█████▓ ░██░░██████▒░▒████▓ ░▒████▒░██▓ ▒██▒
# ░▒▓███▀▒░▒▓▒ ▒ ▒ ░▓  ░ ▒░▓  ░ ▒▒▓  ▒ ░░ ▒░ ░░ ▒▓ ░▒▓░
# ▒░▒   ░ ░░▒░ ░ ░  ▒ ░░ ░ ▒  ░ ░ ▒  ▒  ░ ░  ░  ░▒ ░ ▒░
#  ░    ░  ░░░ ░ ░  ▒ ░  ░ ░    ░ ░  ░    ░     ░░   ░ 
#  ░         ░      ░      ░  ░   ░       ░  ░   ░     
#       ░                       ░                      
#
FROM golang:1.24-alpine AS builder

ARG BUILD_VCS_REF
ARG BUILD_VERSION

WORKDIR /usr/src/anicetus-http
COPY --chown=root:root . /usr/src/anicetus-http
RUN go build -o /app/anicetus-http ./cmd/anicetus-http


# ██▀███   █    ██  ███▄    █  ███▄    █ ▓█████  ██▀███  
# ▓██ ▒ ██▒ ██  ▓██▒ ██ ▀█   █  ██ ▀█   █ ▓█   ▀ ▓██ ▒ ██▒
# ▓██ ░▄█ ▒▓██  ▒██░▓██  ▀█ ██▒▓██  ▀█ ██▒▒███   ▓██ ░▄█ ▒
# ▒██▀▀█▄  ▓▓█  ░██░▓██▒  ▐▌██▒▓██▒  ▐▌██▒▒▓█  ▄ ▒██▀▀█▄  
# ░██▓ ▒██▒▒▒█████▓ ▒██░   ▓██░▒██░   ▓██░░▒████▒░██▓ ▒██▒
# ░ ▒▓ ░▒▓░░▒▓▒ ▒ ▒ ░ ▒░   ▒ ▒ ░ ▒░   ▒ ▒ ░░ ▒░ ░░ ▒▓ ░▒▓░
#   ░▒ ░ ▒░░░▒░ ░ ░ ░ ░░   ░ ▒░░ ░░   ░ ▒░ ░ ░  ░  ░▒ ░ ▒░
#   ░░   ░  ░░░ ░ ░    ░   ░ ░    ░   ░ ░    ░     ░░   ░ 
#    ░        ░              ░          ░    ░  ░   ░     
#
FROM alpine:3 AS runner

COPY --from=builder /app/anicetus-http /bin/anicetus-http

LABEL org.label-schema.build-date=$BUILD_DATE \
      org.label-schema.description="Microservice to control thundering herd problem" \
      org.label-schema.name="anicetus-http" \
      org.label-schema.schema-version="1.0" \
      org.label-schema.url="https://github.com/rafaeljusto/anicetus" \
      org.label-schema.vcs-url="https://github.com/rafaeljusto/anicetus" \
      org.label-schema.vcs-ref=$BUILD_VCS_REF \
      org.label-schema.vendor="Rafael Dantas Justo" \
      org.label-schema.version=$BUILD_VERSION

EXPOSE 80
ENV ANICETUS_PORT=80
ENTRYPOINT ["/bin/anicetus-http"]