# syntax=docker/dockerfile:1

FROM golang:1.24-alpine AS builder

WORKDIR /usr/src/anicetus-example-backend
COPY --chown=root:root . /usr/src/anicetus-example-backend
RUN go build -o /app/anicetus-example-backend .

EXPOSE 80
ENV ANICETUS_EXAMPLE_BACKEND_PORT=80
ENTRYPOINT ["/app/anicetus-example-backend"]