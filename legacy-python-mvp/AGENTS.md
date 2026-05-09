# Project Notes

This legacy MVP is an OpenAI-compatible API forwarding and usage metering reference.

- Keep request/response body logging off by default. Usage monitoring should store metadata, status, latency, byte counts, and token usage only.
- Prefer small, explicit FastAPI modules and SQLite-backed state until the project outgrows a single-node deployment.
- When deploying this legacy MVP to a shared low-disk Docker host, build images locally or in CI and let the server only load and run them. After the new container is healthy, clean only unused images and build cache with `docker image prune -a -f` and `docker builder prune -f`; do not prune volumes or delete database/config directories such as `/data`, compose `.env`, or upload folders.
