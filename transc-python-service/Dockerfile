# Python API Dockerfile
FROM python:3.10-slim

WORKDIR /app

# Установка FFmpeg (ОБЯЗАТЕЛЬНО для Whisper!)
RUN apt-get update && apt-get install -y \
    ffmpeg \
    && rm -rf /var/lib/apt/lists/*

# Копируем зависимости
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

# Копируем код
COPY . .

EXPOSE 8000

CMD ["uvicorn", "main:app", "--host", "0.0.0.0", "--port", "8000"]
