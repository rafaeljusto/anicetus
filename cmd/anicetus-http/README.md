# Anicetus HTTP

Anicetus HTTP is an HTTP proxy server that protects the backend against
thundering herd sittuation. For now only `GET` HTTP requests are analyzed.

It uses an in-memory token bucket algorithm to detect when a thundering herd is
happening, returning the 503 HTTP status code (Service Unavailable) for all
blocked requests.

The backend will receive some extra HTTP headers to help it understand the state
of the proxy:

* `Anicetus-Status`: The status of the request. It can be `open-gates` (no
  thundering herd detected) or `process` (thundering herd detected and this is
  the single request allowed for caching).

* `Anicetus-Fingerprint`: A unique identifier for the thundering herd.

Besides the token bucket algorithm kept in-memory, it will also store the state
of the thundering herds in-memory as well. This means that if the server goes
down or restarts, the state will be lost. This will improved in the future
giving more configuration options to allow external storage.

The following environment variables can be used to configure the server:

| Environment Variable                    | Description                                   |
| --------------------------------------- | --------------------------------------------- |
| `ANICETUS_BACKEND_ADDRESS`              | Backend address and port                      |
| `ANICETUS_BACKEND_TIMEOUT`              | Backed processing timeout                     |
| `ANICETUS_DETECTOR_COOLDOWN`            | Cooldown period                               |
| `ANICETUS_DETECTOR_REQUESTS_PER_MINUTE` | Allowed requests per minute                   |
| `ANICETUS_FINGERPRINT_COOKIES`          | Cookies that are part of the fingerprint      |
| `ANICETUS_FINGERPRINT_FIELDS`           | URL fields that are part of the fingerprint   |
| `ANICETUS_FINGERPRINT_HEADERS`          | HTTP headers that are part of the fingerprint |
| `ANICETUS_LOG_LEVEL`                    | Log level                                     |
| `ANICETUS_PORT`                         | HTTP port to listen                           |