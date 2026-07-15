#!/usr/bin/env python3
"""KnowledgeOS contract eval: every KB question must be answered from the KB,
otherwise handed off to an operator. Runs each of the 798 QA questions through
the real bot chat pipeline and classifies the outcome."""
import json, os, sys, time, threading, urllib.request, urllib.error
from concurrent.futures import ThreadPoolExecutor, as_completed

BASE = os.environ.get("BASE", "http://localhost:8080/api/v1")
EMAIL = os.environ.get("EMAIL", "admin@example.com")
PASSWORD = os.environ.get("PASSWORD", "admin12345")
COMPANY_ID = os.environ.get("COMPANY_ID", "1c5b91c2-df91-42c4-a88e-b2e3b1a34c05")
QA_FILE = os.environ.get("QA_FILE", "qa_pairs.json")
OUT = os.environ.get("OUT", "results.jsonl")
WORKERS = int(os.environ.get("WORKERS", "5"))
LIMIT = int(os.environ.get("LIMIT", "0"))  # 0 = all

_lock = threading.Lock()
_done = 0
_token = None


def http(method, path, body=None, token=None, timeout=180):
    url = BASE + path
    data = json.dumps(body).encode() if body is not None else None
    req = urllib.request.Request(url, data=data, method=method)
    req.add_header("Content-Type", "application/json")
    if token:
        req.add_header("Authorization", "Bearer " + token)
        req.add_header("X-Company-ID", COMPANY_ID)
    last = None
    for attempt in range(6):
        try:
            with urllib.request.urlopen(req, timeout=timeout) as r:
                return r.getcode(), json.loads(r.read().decode())
        except urllib.error.HTTPError as e:
            code = e.code
            try:
                payload = json.loads(e.read().decode())
            except Exception:
                payload = {"error": "http %d" % code}
            if code in (429, 500, 502, 503, 504):
                last = (code, payload)
                time.sleep(min(2 ** attempt, 30))
                continue
            return code, payload
        except Exception as e:  # network / timeout
            last = (0, {"error": str(e)})
            time.sleep(min(2 ** attempt, 30))
    return last if last else (0, {"error": "unknown"})


def login():
    code, resp = http("POST", "/auth/login", {"email": EMAIL, "password": PASSWORD})
    if code != 200:
        print("LOGIN FAILED", code, resp); sys.exit(1)
    return resp["data"]["access_token"]


def classify(qa_id, msg, session):
    ga = (msg or {}).get("guardrail_action", "")
    content = (msg or {}).get("content", "") or ""
    sources = (msg or {}).get("sources", []) or []
    cited = (msg or {}).get("cited_source_ids", []) or []
    state = (session or {}).get("state", "")
    src_ids = [str(s.get("entity_id", "")) for s in sources]
    grounded_self = qa_id in src_ids
    cited_self = any(qa_id in str(c) for c in cited)
    escalated = state in ("waiting_operator", "operator") or ga in ("escalate", "handoff")
    answered = ga == "answer" and content.strip() != ""
    if escalated:
        outcome = "HANDOFF"
    elif answered and grounded_self:
        outcome = "PASS_ANSWER"
    elif answered and not grounded_self:
        outcome = "ANSWER_NO_SELF_SOURCE"
    elif content.strip() and not answered:
        outcome = "REFUSED"
    else:
        outcome = "EMPTY"
    return {
        "outcome": outcome, "guardrail_action": ga, "state": state,
        "grounded_self": grounded_self, "cited_self": cited_self,
        "n_sources": len(sources), "answer": content[:400],
    }


def run_one(qa):
    global _done
    qid = qa["id"]
    # create session
    code, resp = http("POST", "/admin/bot/chat/sessions", {"title": "eval"}, _token)
    if code not in (200, 201):
        res = {"id": qid, "question": qa["question"][:200], "outcome": "ERROR",
               "error": "session %d %s" % (code, resp)}
    else:
        sid = resp["data"]["id"] if "id" in resp.get("data", {}) else resp["data"].get("session", {}).get("id")
        code, resp = http("POST", "/admin/bot/chat/sessions/%s/messages" % sid,
                          {"content": qa["question"]}, _token)
        if code != 200:
            res = {"id": qid, "question": qa["question"][:200], "outcome": "ERROR",
                   "error": "message %d %s" % (code, resp)}
        else:
            data = resp.get("data", {})
            c = classify(qid, data.get("message"), data.get("session"))
            res = {"id": qid, "question": qa["question"][:200], "session_id": sid, **c}
    with _lock:
        _done += 1
        with open(OUT, "a") as f:
            f.write(json.dumps(res, ensure_ascii=False) + "\n")
        if _done % 25 == 0 or _done == TOTAL:
            print("  %d/%d  last=%s" % (_done, TOTAL, res["outcome"]), flush=True)
    return res


if __name__ == "__main__":
    qas = json.load(open(QA_FILE))
    if LIMIT:
        qas = qas[:LIMIT]
    TOTAL = len(qas)
    open(OUT, "w").close()  # truncate
    _token = login()
    print("Logged in. Running %d questions with %d workers -> %s" % (TOTAL, WORKERS, OUT), flush=True)
    t0 = time.time()
    with ThreadPoolExecutor(max_workers=WORKERS) as ex:
        futs = [ex.submit(run_one, qa) for qa in qas]
        for _ in as_completed(futs):
            pass
    dt = time.time() - t0
    # summary
    counts = {}
    for line in open(OUT):
        o = json.loads(line)["outcome"]
        counts[o] = counts.get(o, 0) + 1
    print("\n==== SUMMARY (%.0fs) ====" % dt)
    for k in sorted(counts):
        print("  %-22s %d" % (k, counts[k]))
    print("  TOTAL                  %d" % TOTAL)
