# Project Notes

This legacy MVP is an OpenAI-compatible API forwarding and usage metering reference built on official API keys.

- Do not add code that extracts, stores, reuses, or shares Codex/ChatGPT login sessions, browser cookies, device codes, or personal account tokens.
- The upstream credential source is `OPENAI_API_KEY`, preferably a project key or service-account key managed in the OpenAI platform.
- Keep request/response body logging off by default. Usage monitoring should store metadata, status, latency, byte counts, and token usage only.
- Prefer small, explicit FastAPI modules and SQLite-backed state until the project outgrows a single-node deployment.
