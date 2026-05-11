# main.py
import whisper
import tempfile
import os
import sys
import platform
from fastapi import FastAPI, UploadFile, File, HTTPException
from fastapi.responses import JSONResponse
import uvicorn

# Локальный Windows-путь к ffmpeg (если есть); в Docker ffmpeg уже в PATH.
if platform.system() == "Windows":
    FFMPEG_PATH = os.environ.get(
        "TSK_FFMPEG_BIN",
        r"C:\Users\user\Downloads\ffmpeg-8.0.1-essentials_build\ffmpeg-8.0.1-essentials_build\bin",
    )
    if os.path.isdir(FFMPEG_PATH):
        os.environ["PATH"] = FFMPEG_PATH + ";" + os.environ["PATH"]

print("=" * 60)
print("Проверка FFmpeg...")
try:
    import subprocess
    result = subprocess.run(['ffmpeg', '-version'], capture_output=True, text=True, timeout=5)
    print("FFmpeg найден и работает!")
    print(f"   Версия: {result.stdout.split(chr(10))[0]}")
except Exception as e:
    print(f"FFmpeg не найден: {e}")
    if platform.system() == "Windows":
        print("Укажите каталог с ffmpeg в переменной TSK_FFMPEG_BIN или установите ffmpeg в PATH.")
    sys.exit(1)
print("=" * 60)

app = FastAPI()

print("Загружаем модель Whisper...")
try:
    model = whisper.load_model("small") 
    print("Модель загружена!")
except Exception as e:
    print(f"шибка загрузки моде: {e}")
    model = None

@app.get("/")
async def root():
    return {
        "status": "running", 
        "service": "Whisper Transcription API",
        "model": "small",
        "ffmpeg": "available"
    }

@app.post("/transcribe")
async def transcribe(file: UploadFile = File(...)):
    if not model:
        raise HTTPException(status_code=500, detail="Модель не загружена")
    
    print(f"Получен файл: {file.filename}")
    
    allowed_extensions = {'.wav', '.mp3', '.ogg', '.oga', '.m4a', '.flac', '.aac'}
    suffix = os.path.splitext(file.filename)[1].lower()
    
    if suffix not in allowed_extensions:
        raise HTTPException(
            status_code=400, 
            detail=f"Неподдерживаемый формат файла. Допустимые: {', '.join(allowed_extensions)}"
        )
    
    content = await file.read()
    print(f"Размер файла: {len(content)} байт")
    
    with tempfile.NamedTemporaryFile(delete=False, suffix=suffix) as tmp:
        tmp.write(content)
        tmp_path = tmp.name
    
    print(f"Сохранен во временный файл: {tmp_path}")
    
    if not os.path.exists(tmp_path):
        raise HTTPException(status_code=500, detail="Не удалось создать временный файл")
    
    try:
        print("Начинаем транскрипцию...")
        
        try:
            result = subprocess.run(
                ['ffprobe', '-i', tmp_path, '-show_format', '-v', 'quiet'],
                capture_output=True,
                text=True
            )
            print(f"FFprobe: файл корректен")
        except:
            print(f"FFprobe: не удалось проверить файл")
        
        result = model.transcribe(
            tmp_path, 
            language="ru", 
            fp16=False,  
            verbose=True  
        )
        
        text = result["text"].strip()
        print(f"Результат транскрипции: '{text}'")
        print(f"⏱Длительность аудио: {result.get('duration', 0):.2f} сек")
        
        return JSONResponse({
            "text": text,
            "duration": result.get("duration", 0),
            "language": "ru",
            "success": True
        })
        
    except Exception as e:
        print(f"Ошибка транскрипции: {e}")
        import traceback
        traceback.print_exc()
        
        print("\nДиагностика:")
        print(f"Путь к файлу: {tmp_path}")
        print(f"Файл существует: {os.path.exists(tmp_path)}")
        print(f"Размер файла: {os.path.getsize(tmp_path) if os.path.exists(tmp_path) else 0} байт")
        
        raise HTTPException(status_code=500, detail=f"Ошибка транскрипции: {str(e)}")
        
    finally:
        if os.path.exists(tmp_path):
            os.remove(tmp_path)
            print(f"Временный файл удален: {tmp_path}")

if __name__ == "__main__":
    print("\n" + "=" * 60)
    print("Whisper Transcription API запущен!")
    print("=" * 60)
    print("Доступные эндпоинты:")
    print("   - GET  /           → Проверка работы сервера")
    print("   - POST /transcribe → Транскрипция аудио")
    print("   - GET  /docs       → Документация API (Swagger)")
    print("   - GET  /redoc      → Альтернативная документация")
    print("\nАдреса для доступа:")
    print("   - http://localhost:8000/")
    print("   - http://localhost:8000/docs")
    print("   - http://127.0.0.1:8000/docs")
    print("=" * 60 + "\n")
    
    uvicorn.run(
        app, 
        host="0.0.0.0", 
        port=8000,
        log_level="info"
    )