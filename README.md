# Anicetus

[![Go Reference](https://pkg.go.dev/badge/github.com/rafaeljusto/anicetus.svg)](https://pkg.go.dev/github.com/rafaeljusto/anicetus)
![Test](https://github.com/rafaeljusto/anicetus/actions/workflows/test.yml/badge.svg)

The [thundering herd problem](https://en.wikipedia.org/wiki/Thundering_herd_problem)
is a performance problem that occurs when a large number of processes or threads
wake up at the same time and compete for a limited resource.

This project attempts to resolve this by detecting the sittuation, and only
allowing a single request to pass through to the resource, while the others are
blocked. Once the resource is available, the blocked requests are released. The
reason is that allowing a single request to finish first creates a caching
effect, and the other requests can be served from the cache.

Out of the box, a web server (`anicetus-http`) is provided as a micro-service
for your network, but this library can be used in any Go project to plug-in with
an existing solution.

## Design

```mermaid
flowchart LR
  A((Start)):::start --> B[Fingerprint]
  B[Fingerprint]:::fingerprint --> C[Detector]
  C[Detector]:::detector --> D{Cooldown?}:::detector
  D:::detector -->|yes| F(((Backend))):::success
  D:::detector -->|no| E{Thundering herd?}:::detector
  E:::detector -->|yes| G[Gatekeeper]:::gatekeeper
  E:::detector -->|no| F(((Backend))):::success
  G[Gatekeeper]:::gatekeeper --> H{First request?}:::gatekeeper
  H:::gatekeeper -->|yes| F(((Backend))):::success
  H:::gatekeeper -->|no| I{First request finished?}:::gatekeeper
  I:::gatekeeper -->|yes| F(((Backend))):::success
  I:::gatekeeper -->|no| J(((Block))):::error

  classDef start fill:#092c9e,stroke:#333,stroke-width:2px;
  classDef fingerprint fill:#a1880d,stroke:#333,stroke-width:2px;
  classDef detector fill:#4c147a,stroke:#333,stroke-width:2px;
  classDef gatekeeper fill:#0a6e6c,stroke:#333,stroke-width:2px;
  classDef success fill:#118f0d,stroke:#333,stroke-width:2px;
  classDef error fill:#a30303,stroke:#333,stroke-width:2px;
```

There are some key concepts for this project that will be explained in the next
sections.

### Fingerprint

The fingerprint is a unique identifier for the request. It is used to identify a
group of requests that are considered the same. For example, in a web
application, depending on the use case, the fingerprint could be the URL, some
specific HTTP headers, or a combination of both.

Allowing to configure how the fingerprint is generated gives the flexibility to
target specific users or not.

> [!IMPORTANT]
> Defining the correct fingerprint that matches the backend caching group is
> crucial for this solution to work. If the backend application cannot cache the
> incoming requests from the group of requests identified by the fingerprint,
> the thundering herd will still be a problem.

### Detector

Once the fingerprint is generated, the detector will check if there are other
requests with the same fingerprint. Different algorithms can be used to detect a
thundeting herd, and this library provides the
[token bucket](https://en.wikipedia.org/wiki/Token_bucket) out-of-the-box. When
using the token bucket algorithm the detector needs to know how many same
fingerprint occurences are allowed in a time window.

After the thundering herd is handled, the detector will stop analysing the
requests for a while (cooldown period). This is to avoid the thundering herd to
be detected again in a short period of time. The cooldown period shoud be
configured taking the stale cache risk of the backend into account.

### Gatekeeper

After a thundering herd is detected, the gatekeeping will allow the first
request to pass through, and block the others. The blocked requests will be
allowed to hit the backend once the first request execution is done.

> [!IMPORTANT]
> The backend MUST create a cache response for requests with the same
> fingerprint to avoid the thundering herd to hit the infrastructure.

## Library

The library is the core of the project. It provides the necessary functions to
be used in any Go project. The library is designed to be flexible so you can
implement your own detector and gatekeeper storage.

```go
package main

import (
  "context"
  "net/http"
  "time"

  "github.com/rafaeljusto/anicetus"
  "github.com/rafaeljusto/anicetus/detector"
  "github.com/rafaeljusto/anicetus/storage"
)

func main() {
  detector := detector.NewTokenBucketInMemory(
    detector.WithLimitersBurst(1000),
    detector.WithLimitersInterval(time.Minute),
    detector.WithCoolDownInterval(10*time.Minute),
  )

  gatekeeperStorage := storage.NewInMemory()

  anicetus := anicetus.NewAnicetus[fingerprint.HTTPRequest](detector, gatekeeperStorage)

  // ...

  // For each request in your application (here we simulate an HTTP request)
  requestFingerprint := fingerprint.NewHTTPRequest(&http.Request{})
  status, err := anicetus.Evaluate(requestFingerprint)
  if err != nil {
    // handle error
  }

  switch status {
  case anicetus.StatusProcess:
    // thundering herd detected, allowing this single request to pass through
  case anicetus.StatusWait:
    // thundering herd detected, blocking this request
  case anicetus.StatusOpenGates:
    // business as usual
  case anicetus.StatusFailed:
    // something went wrong while evaluating the request

    // you can optionally cleanup the request fingerprint state
    if err := aniceuts.Cleanup(requestFingerprint); err != nil {
      // handle error
    }
  }

  // the single request needs to inform the gatekeeper that it finished
  // processing the request
  if err := anicetus.RequestDone(requestFingerprint); err != nil {
    // handle error
  }
}
```

## FAQ

You will find here some common questions and answers.

**Q: Is this a load balancer?**

A: No, this is not a load balancer. This works as a semaphore service allowing
requests to pass through.

**Q: Does it cache the responses?**

A: No, it does not cache the responses. It only allows a single request to pass
through. It's the responsibility of the backend application to cache the
responses and serve them from the cache. This solution will just give enough
time for the cache to be populated.

**Q: Why Anicetus?**

A: Anicetus comes from the Greek word Ἀνίκητος (Aníkētos, literally
"Unconquerable"). A god and one of the guardians and gatekeepers of the gates of
Mount Olympus. It's a metaphor for the protector of the backend application. 😄🤞

**Q: Why there are not too many options for detectors, gatekeeper storages and
fingerprints in this library?**

A: At first we are trying to minimize the number of Go dependencies to make this
library lightweight. We may in the near future add support for Redis storages,
creating dependencies to other third-party libraries.