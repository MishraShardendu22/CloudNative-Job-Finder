FROM python:3.12-slim
WORKDIR /app
ENV PYTHONDONTWRITEBYTECODE=1
ENV PYTHONUNBUFFERED=1

COPY services/resume-parser/requirements.txt /app/requirements.txt
RUN pip install --no-cache-dir -r /app/requirements.txt && python -m nltk.downloader stopwords

COPY services/resume-parser/app.py /app/app.py
EXPOSE 8083
CMD ["python", "app.py"]
