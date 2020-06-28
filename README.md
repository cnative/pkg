# Common Module for building go servers

This repo provides a common set of modules for building micro-services that run on K8S with conistent

    - logging
    - tracing
    - metrics
    - auth runtime (OpenID Connect)
    - authz support (experimental)
    - TLS handling (mTLS including)
    - debug server (with pprof)
    - health checks

grpc, grpc-gateway and http/rest micro services are some the server types that are supported.