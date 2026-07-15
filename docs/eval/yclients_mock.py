#!/usr/bin/env python3
"""Mock YCLIENTS API for verifying the bot booking flow end-to-end."""
import json, re
from http.server import HTTPServer, BaseHTTPRequestHandler

SERVICES = [
    {"id": 101, "title": "Приём терапевта", "price_min": 800, "price_max": 1200, "seance_length": 1800},
    {"id": 102, "title": "Вакцинация комплексная", "price_min": 1500, "price_max": 2000, "seance_length": 1200},
    {"id": 103, "title": "УЗИ брюшной полости", "price_min": 1800, "price_max": 1800, "seance_length": 1800},
]
STAFF = [
    {"id": 7, "name": "Иванова Мария Петровна", "specialization": "Терапевт", "bookable": True},
    {"id": 9, "name": "Сидоров Алексей Викторович", "specialization": "Хирург, УЗИ", "bookable": True},
]
BOOKINGS = []


class H(BaseHTTPRequestHandler):
    def _send(self, data):
        body = json.dumps({"success": True, "data": data, "meta": {}}).encode()
        self.send_response(200)
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(body)))
        self.end_headers()
        self.wfile.write(body)

    def do_GET(self):
        if re.match(r"^/book_services/", self.path):
            self._send({"services": SERVICES})
        elif re.match(r"^/book_staff/", self.path):
            self._send(STAFF)
        elif re.match(r"^/book_times/", self.path):
            m = re.match(r"^/book_times/[^/]+/(\d+)/(\d{4}-\d{2}-\d{2})", self.path)
            date = m.group(2) if m else "2026-07-15"
            slots = [{"time": t, "datetime": f"{date}T{t}:00+03:00"} for t in ("10:00", "12:30", "16:00")]
            self._send(slots)
        else:
            self._send({})

    def do_POST(self):
        length = int(self.headers.get("Content-Length", 0))
        payload = json.loads(self.rfile.read(length) or b"{}")
        if re.match(r"^/book_record/", self.path):
            BOOKINGS.append(payload)
            with open("yclients_bookings.jsonl", "a") as f:
                f.write(json.dumps(payload, ensure_ascii=False) + "\n")
            self._send([{"record_id": 555001, "record_hash": "abc123"}])
        else:
            self._send({})

    def log_message(self, fmt, *args):
        print("MOCK:", fmt % args, flush=True)


if __name__ == "__main__":
    HTTPServer(("0.0.0.0", 9911), H).serve_forever()
