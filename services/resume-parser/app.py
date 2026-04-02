import logging
import os
import threading
from http.server import HTTPServer

from health_handler import HealthHandler
from resume_parser_core import ResumeParser

logging.basicConfig(level=logging.INFO, format="%(asctime)s [%(levelname)s] %(message)s")


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
