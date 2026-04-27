FROM python:3.10-slim

WORKDIR /app

RUN apt-get update && apt-get install -y \
    ffmpeg \
    build-essential \
    && rm -rf /var/lib/apt/lists/*

COPY requirements.txt .

# Увеличиваем таймаут и используем ретроми
RUN pip install --default-timeout=100 --retries=5 --upgrade pip && \
    pip install --default-timeout=100 --retries=5 --no-cache-dir -r requirements.txt

COPY . .

EXPOSE 8000

CMD ["uvicorn", "main:app", "--host", "0.0.0.0", "--port", "8000"]