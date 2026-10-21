"""Development-only live process check; Python is not an application dependency."""
import html
import http.cookiejar
import json
import os
from pathlib import Path
import re
import shutil
import signal
import subprocess
import tempfile
import time
import urllib.error
import urllib.parse
import urllib.request


def main():
    repo = Path(__file__).resolve().parent.parent
    binary = repo / "bin" / ("asteriade.exe" if os.name == "nt" else "asteriade")
    root = Path(tempfile.mkdtemp(prefix="asteriade-live-"))
    data = root / "data"
    process = None
    try:
        def start(directory):
            child = subprocess.Popen(
                [str(binary), "-data-dir", str(directory), "-no-browser"],
                stdout=subprocess.PIPE, stderr=subprocess.PIPE, text=True,
            )
            for _ in range(200):
                if child.poll() is not None:
                    raise RuntimeError("Application startup failed; inspect local diagnostics.")
                for launch in directory.glob("launch-*.html"):
                    match = re.search(r'href="([^"]+)"', launch.read_text())
                    if not match:
                        continue
                    target = urllib.parse.urlsplit(html.unescape(match.group(1)))
                    origin = target.scheme + "://" + target.netloc
                    token = urllib.parse.parse_qs(target.fragment).get("bootstrap", [""])[0]
                    client = urllib.request.build_opener(
                        urllib.request.HTTPCookieProcessor(http.cookiejar.CookieJar()))
                    req = urllib.request.Request(
                        origin + "/api/v1/session", json.dumps({"token": token}).encode(),
                        {"Origin": origin, "Content-Type": "application/json"}, method="POST")
                    try:
                        with client.open(req, timeout=2) as response:
                            csrf = json.load(response)["csrf_token"]
                        return child, client, origin, csrf
                    except (urllib.error.URLError, TimeoutError):
                        pass
                time.sleep(0.05)
            child.kill()
            child.communicate()
            raise RuntimeError("Application startup timeout")

        def request(path, body=None, key="live-key", extra=None):
            headers = {"Origin": origin, "Content-Type": "application/json",
                       "X-CSRF-Token": csrf, "Idempotency-Key": key}
            headers.update(extra or {})
            req = urllib.request.Request(
                origin + path, json.dumps(body).encode() if body is not None else None,
                headers, method="POST" if body is not None else "GET")
            try:
                with client.open(req, timeout=5) as response:
                    return response.status, json.load(response)
            except urllib.error.HTTPError as response:
                return response.code, json.load(response)

        def stop(child):
            child.send_signal(signal.SIGINT)
            child.communicate(timeout=10)
            assert child.returncode == 0, "Graceful shutdown failed"

        process, client, origin, csrf = start(data)
        assert request("/api/v1/diagnostics")[1]["execution_enabled"] is False
        body = {"title": "Live restart check", "goal": "Persist without tool execution"}
        status, project = request("/api/v1/projects", body)
        assert status == 201
        assert request("/api/v1/projects", body)[1]["id"] == project["id"]
        assert request("/api/v1/projects/" + project["id"] + "/start", {})[0] == 409
        assert request("/api/v1/projects", body, "bad-origin",
                       {"Origin": "https://attacker.invalid"})[0] == 403
        stop(process)
        process = None
        assert not list(data.glob("launch-*.html"))
        process, client, origin, csrf = start(data)
        assert request("/api/v1/projects/" + project["id"])[1]["goal"] == body["goal"]
        stop(process)
        process = None
        backup = root / "backup.db"
        result = subprocess.run([str(binary), "-data-dir", str(data), "-backup", str(backup)],
                                capture_output=True, text=True, timeout=20)
        assert result.returncode == 0, "Backup failed"
        restored = root / "restored"
        restored.mkdir(mode=0o700)
        shutil.copyfile(backup, restored / "asteriade.db")
        os.chmod(restored / "asteriade.db", 0o600)
        process, client, origin, csrf = start(restored)
        assert request("/api/v1/projects/" + project["id"])[1]["id"] == project["id"]
        stop(process)
        process = None
        print("PASS: live session, idempotency, refused execution/Origin, restart, backup, restore")
    finally:
        if process is not None and process.poll() is None:
            process.kill()
            process.communicate()
        shutil.rmtree(root)


if __name__ == "__main__":
    if os.name == "nt":
        raise SystemExit("Use scripts/verify-windows.ps1 and the manual Windows validation procedure.")
    main()
