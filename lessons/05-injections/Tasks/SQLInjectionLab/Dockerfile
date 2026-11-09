FROM python:3.13-alpine

ENV PYTHONDONTWRITEBYTECODE=1 \
    PYTHONUNBUFFERED=1

WORKDIR /app

COPY requirements.txt ./
RUN pip install --no-cache-dir -r requirements.txt

COPY app.py ./
COPY templates ./templates

RUN addgroup -S appgroup && adduser -S -G appgroup appuser
USER appuser

EXPOSE 8080

CMD ["python", "app.py"]
