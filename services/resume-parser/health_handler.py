import json
from http.server import BaseHTTPRequestHandler


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
