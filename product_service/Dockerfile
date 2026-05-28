# Stage 1 — install dependencies
FROM python:3.12-slim AS builder
WORKDIR /app
COPY requirements.txt .
RUN pip install --no-cache-dir --prefix=/install -r requirements.txt

# Stage 2 — lean production image
FROM python:3.12-slim
WORKDIR /app
COPY --from=builder /install /usr/local
COPY main.py .
EXPOSE 3002
CMD ["uvicorn", "main:app", "--host", "0.0.0.0", "--port", "3002"]


