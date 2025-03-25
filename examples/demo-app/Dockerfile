# SPDX-FileCopyrightText: 2025 Stanislav Zaprudskiy <stanislav.zaprudskiy@gmail.com>
#
# SPDX-License-Identifier: Apache-2.0

FROM ghcr.io/astral-sh/uv:python3.13-bookworm-slim AS builder

ENV UV_COMPILE_BYTECODE=1 UV_LINK_MODE=copy
ENV UV_PYTHON_DOWNLOADS=0

WORKDIR /app
ADD app.py .
RUN uv venv \
  && . .venv/bin/activate \
  && uv sync --script app.py --active

FROM python:3.13-slim-bookworm

ENV PATH="/app/.venv/bin:$PATH"
ENV PYTHONUNBUFFERED=1
WORKDIR /app
COPY --from=builder --chown=1000:1000 /app /app
CMD ["./app.py"]
