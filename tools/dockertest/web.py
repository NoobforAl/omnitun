from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer


class H(BaseHTTPRequestHandler):
    def do_GET(self):
        if self.path.startswith("/big"):
            # /big?mb=N: N megabytes, for a transfer that lasts long enough to watch.
            mb = int(self.path.split("mb=")[-1] or 1) if "mb=" in self.path else 1
            self.send_response(200)
            self.send_header("Content-Length", str(mb << 20))
            self.end_headers()
            chunk = b"x" * 65536
            try:
                for _ in range(mb * 16):
                    self.wfile.write(chunk)
            except OSError:
                pass
            return
        body = ("client=%s\n" % self.client_address[0]).encode()
        self.send_response(200)
        self.send_header("Content-Length", str(len(body)))
        self.end_headers()
        self.wfile.write(body)

    def log_message(self, *a):
        pass


ThreadingHTTPServer(("0.0.0.0", 8080), H).serve_forever()
