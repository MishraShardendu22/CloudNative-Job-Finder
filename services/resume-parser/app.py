import json
import logging
import os
import re
import tempfile
import threading
import time
from collections import Counter
from datetime import datetime, timezone
from http.server import BaseHTTPRequestHandler, HTTPServer

import fitz
import nltk
import pika
import requests
import spacy
from minio import Minio
from nltk.corpus import stopwords
from pdfminer.high_level import extract_text as pdf_extract_text

logging.basicConfig(level=logging.INFO, format="%(asctime)s [%(levelname)s] %(message)s")


class ParserState:
    def __init__(self):
        self.last_processed_at = None
        self.last_error = None


class ResumeParser:
    def __init__(self):
        self.state = ParserState()

        self.rabbitmq_url = os.getenv("RABBITMQ_URL", "amqp://guest:guest@rabbitmq:5672/")
        self.exchange = os.getenv("RABBITMQ_EXCHANGE", "events")
        self.queue = os.getenv("RABBITMQ_QUEUE", "resume-parser.queue")
        self.routing_key = "resume_uploaded"

        self.minio_endpoint = os.getenv("MINIO_ENDPOINT", "minio:9000")
        self.minio_access_key = os.getenv("MINIO_ACCESS_KEY", "minioadmin")
        self.minio_secret_key = os.getenv("MINIO_SECRET_KEY", "minioadmin")
        self.minio_secure = os.getenv("MINIO_USE_SSL", "false").lower() == "true"

        self.resume_service_url = os.getenv("RESUME_SERVICE_URL", "http://resume-service:8082")
        self.internal_token = os.getenv("INTERNAL_API_TOKEN", "internal-secret")

        self.minio = Minio(
            self.minio_endpoint,
            access_key=self.minio_access_key,
            secret_key=self.minio_secret_key,
            secure=self.minio_secure,
        )

        try:
            nltk.data.find("corpora/stopwords")
        except LookupError:
            nltk.download("stopwords", quiet=True)

        self.stop_words = set(stopwords.words("english"))
        self.nlp = spacy.blank("en")

        self.skill_lexicon = {
            "go",
            "golang",
            "python",
            "java",
            "docker",
            "kubernetes",
            "terraform",
            "postgresql",
            "redis",
            "rabbitmq",
            "microservices",
            "aws",
            "gcp",
            "linux",
            "sql",
            "grpc",
            "rest",
            "ci",
            "cd",
            "git",
        }

        self.job_title_candidates = [
            "software engineer",
            "backend engineer",
            "backend developer",
            "platform engineer",
            "devops engineer",
            "site reliability engineer",
            "data engineer",
            "full stack engineer",
            "engineering manager",
        ]

    def run_consumer_forever(self):
        while True:
            connection = None
            try:
                params = pika.URLParameters(self.rabbitmq_url)
                connection = pika.BlockingConnection(params)
                channel = connection.channel()
                channel.exchange_declare(exchange=self.exchange, exchange_type="topic", durable=True)
                channel.queue_declare(queue=self.queue, durable=True)
                channel.queue_bind(queue=self.queue, exchange=self.exchange, routing_key=self.routing_key)
                channel.basic_qos(prefetch_count=1)
                channel.basic_consume(queue=self.queue, on_message_callback=self._on_message)
                logging.info("resume-parser consumer started")
                channel.start_consuming()
            except Exception as exc:
                self.state.last_error = str(exc)
                logging.exception("consumer loop failed: %s", exc)
                time.sleep(5)
            finally:
                if connection and connection.is_open:
                    connection.close()

    def _on_message(self, ch, method, properties, body):
        del properties
        try:
            payload = json.loads(body.decode("utf-8"))
            self.process_resume_event(payload)
            ch.basic_ack(delivery_tag=method.delivery_tag)
        except Exception as exc:
            self.state.last_error = str(exc)
            logging.exception("failed to process message: %s", exc)
            ch.basic_nack(delivery_tag=method.delivery_tag, requeue=False)

    def process_resume_event(self, payload):
        resume_id = payload.get("resume_id")
        bucket = payload.get("bucket")
        object_key = payload.get("object_key")
        if not resume_id or not bucket or not object_key:
            raise ValueError("resume_uploaded payload is missing required fields")

        logging.info("processing resume_id=%s object=%s/%s", resume_id, bucket, object_key)
        response = self.minio.get_object(bucket, object_key)
        try:
            file_bytes = response.read()
        finally:
            response.close()
            response.release_conn()

        parsed = self.parse_resume(file_bytes, object_key)

        endpoint = f"{self.resume_service_url.rstrip('/')}/internal/resumes/{resume_id}/parsed"
        headers = {
            "Content-Type": "application/json",
            "X-Internal-Token": self.internal_token,
        }
        resp = requests.put(endpoint, headers=headers, data=json.dumps(parsed), timeout=20)
        if resp.status_code >= 400:
            raise RuntimeError(f"resume-service update failed [{resp.status_code}]: {resp.text[:300]}")

        self.state.last_processed_at = datetime.now(timezone.utc).isoformat()
        logging.info("resume parsed and updated resume_id=%s", resume_id)

    def parse_resume(self, file_bytes, object_key):
        text = self.extract_text(file_bytes, object_key)
        if not text.strip():
            return {"skills": [], "technologies": [], "keywords": [], "job_titles": []}

        lowered = text.lower()
        token_candidates = re.findall(r"[a-zA-Z][a-zA-Z0-9+#.-]{1,}", lowered)
        token_candidates = [t for t in token_candidates if t not in self.stop_words and len(t) > 2]

        counts = Counter(token_candidates)
        top_tokens = [token for token, _ in counts.most_common(35)]

        skills = sorted({token for token in top_tokens if token in self.skill_lexicon})

        technologies = sorted({token for token in token_candidates if token in self.skill_lexicon})

        phrases = []
        for phrase in ["distributed systems", "backend development", "microservices", "cloud infrastructure"]:
            if phrase in lowered:
                phrases.append(phrase)

        job_titles = [title for title in self.job_title_candidates if title in lowered]

        doc = self.nlp(text)
        noun_chunks = []
        for token in doc:
            if token.is_alpha and not token.is_stop and len(token.text) > 3:
                noun_chunks.append(token.text.lower())

        keywords = []
        keywords.extend(top_tokens[:20])
        keywords.extend(phrases)
        keywords.extend(noun_chunks[:20])

        deduped_keywords = []
        seen = set()
        for kw in keywords:
            kw = kw.strip().lower()
            if not kw or kw in seen:
                continue
            seen.add(kw)
            deduped_keywords.append(kw)

        return {
            "skills": skills,
            "technologies": technologies,
            "keywords": deduped_keywords[:40],
            "job_titles": job_titles,
        }

    def extract_text(self, file_bytes, object_key):
        if object_key.lower().endswith(".pdf"):
            text = self._extract_pdf_text_pymupdf(file_bytes)
            if text.strip():
                return text
            return self._extract_pdf_text_pdfminer(file_bytes)

        try:
            return file_bytes.decode("utf-8", errors="ignore")
        except Exception:
            return ""

    @staticmethod
    def _extract_pdf_text_pymupdf(file_bytes):
        doc = fitz.open(stream=file_bytes, filetype="pdf")
        chunks = []
        for page in doc:
            chunks.append(page.get_text("text"))
        doc.close()
        return "\n".join(chunks)

    @staticmethod
    def _extract_pdf_text_pdfminer(file_bytes):
        with tempfile.NamedTemporaryFile(suffix=".pdf") as temp_file:
            temp_file.write(file_bytes)
            temp_file.flush()
            return pdf_extract_text(temp_file.name) or ""


class HealthHandler(BaseHTTPRequestHandler):
    parser_instance = None

    def do_GET(self):
        if self.path != "/health":
            self.send_response(404)
            self.end_headers()
            return

        payload = {
            "status": "ok",
            "service": "resume-parser",
            "last_processed_at": self.parser_instance.state.last_processed_at,
            "last_error": self.parser_instance.state.last_error,
        }
        body = json.dumps(payload).encode("utf-8")

        self.send_response(200)
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(body)))
        self.end_headers()
        self.wfile.write(body)

    def log_message(self, format, *args):
        del format, args


def main():
    parser = ResumeParser()
    HealthHandler.parser_instance = parser

    consumer_thread = threading.Thread(target=parser.run_consumer_forever, daemon=True)
    consumer_thread.start()

    port = int(os.getenv("PORT", "8083"))
    server = HTTPServer(("0.0.0.0", port), HealthHandler)
    logging.info("resume-parser health server listening on :%s", port)
    server.serve_forever()


if __name__ == "__main__":
    main()
