# Project Notes

This legacy MVP is an OpenAI-compatible API forwarding and usage metering reference.

- Keep request/response body logging off by default. Usage monitoring should store metadata, status, latency, byte counts, and token usage only.
- Prefer small, explicit FastAPI modules and SQLite-backed state until the project outgrows a single-node deployment.
