#!/usr/bin/env python3
"""wa3lite — pure-Python fallback pipeline for WA3.

Purpose
-------
The canonical implementation is Go (``conformance/tools/wa3``). This module is
the fallback *channel*: when an agent environment cannot run Go, wa3lite can
drive the whole journey in Python and emit a working front+back app:

    init  -> build -> sign -> trust -> operate            (CLI parity with Go)
    app / serve                                           (generate + run app)

It reproduces Go's canonical form byte-for-byte, so a contract signed by
wa3lite verifies under ``wa3 trust`` and vice-versa. It enforces the same
safety rules: scheme allowlist + E-VALUE-SECRET on the backend handle, trust
gate on operate (only signed_trusted may run), and core-side confirmation for
mutating actions.

Providers: mock:// (in-memory, seeded from a build --mock-demo file) and
local:// (JSON files under a sandboxed --data-dir). gdrive://, api://, https://
are reserved for real adapters (v1.2) and fail closed.

Dependencies: Python 3.8+ and ``cryptography`` (Ed25519). No network needed.
"""
from __future__ import annotations

import argparse
import base64
import hashlib
import json
import os
import re
import sys
import unicodedata
from typing import Any, Dict, List, Optional, Tuple

try:
    from cryptography.hazmat.primitives.asymmetric.ed25519 import (
        Ed25519PrivateKey,
        Ed25519PublicKey,
    )
    _HAVE_CRYPTO = True
except Exception:  # pragma: no cover
    _HAVE_CRYPTO = False

# ---------------------------------------------------------------------------
# Constants (mirrors Go main.go)
# ---------------------------------------------------------------------------
SEP = "："
LIST_SEP = "｜"
EXT_PREFIX = "ㄝ"

CORE_HEADER_ORDER = ["ㄊㄡ", "ㄏㄠ", "ㄉㄞ", "ㄏㄡ", "ㄈㄢ"]
ACTION_FIELD_ORDER = ["ㄘ", "ㄓ", "ㄍㄞ", "ㄖㄣ", "ㄕㄡ", "ㄉㄜ", "ㄑㄩㄢ", "ㄊㄞ"]
BLOCK_FIELD_ORDER = ["ㄍㄜ", "ㄩㄢ", "ㄊㄠ", "ㄊㄞ"]
PREF_ORDER = ["ㄋㄛ", "ㄘㄤ", "ㄙㄜ", "ㄗ", "ㄉㄥ", "ㄎㄞ"]
SET_KEYS = {"ㄋㄛ", "ㄘㄤ", "ㄑㄩㄢ"}
SECTION_NAMES = {"ㄋㄥ": "neng", "ㄎㄜ": "ke", "ㄕㄜ": "she", "ㄓㄤ": "zhang"}

VERB_READ, VERB_SEARCH = "ㄎㄢ", "ㄙㄡ"

BACKEND_SCHEMES = ("mock://", "gdrive://", "api://", "https://", "local://")


class Wa3Error(Exception):
    def __init__(self, code: str, msg: str = ""):
        super().__init__(f"{code}: {msg}" if msg else code)
        self.code = code
        self.msg = msg


def fail(code: str, msg: str = "") -> Wa3Error:
    return Wa3Error(code, msg)


# ---------------------------------------------------------------------------
# Normalize / parse  (must match Go normalize + parseDoc)
# ---------------------------------------------------------------------------
def _has_zw_or_bidi(s: str) -> bool:
    for ch in s:
        o = ord(ch)
        if 0x200B <= o <= 0x200D or o == 0xFEFF or 0x202A <= o <= 0x202E or 0x2066 <= o <= 0x2069:
            return True
    return False


def _is_c0c1_control(o: int) -> bool:
    return (0x00 <= o <= 0x1F and o != 0x0A) or o == 0x7F or (0x80 <= o <= 0x9F)


def normalize(raw: bytes) -> str:
    if raw.startswith(b"\xef\xbb\xbf"):
        raise fail("E-ENC-BOM", "byte order mark")
    if b"\x00" in raw:
        raise fail("E-ENC-NULL", "null byte")
    try:
        text = raw.decode("utf-8")
    except UnicodeDecodeError:
        raise fail("E-ENC-UTF8", "not valid utf-8")
    text = text.replace("\r\n", "\n").replace("\r", "\n")
    text = unicodedata.normalize("NFC", text)
    return "".join(c for c in text if c == "\n" or not _is_c0c1_control(ord(c)))


def _clean_value(v: str) -> str:
    return "".join(c for c in v if not _has_zw_or_bidi(c)).rstrip(" \t")


def _split_line(line: str) -> Tuple[int, Dict[str, str]]:
    stripped = line.lstrip(" ")
    indent = len(line) - len(stripped)
    if SEP not in stripped:
        raise fail("E-STRUCT-PARSE", "no separator: " + line)
    key, _, rawval = stripped.partition(SEP)
    key = key.strip()
    if _has_zw_or_bidi(key):
        raise fail("E-TOKEN-INVALID", repr(key))
    return indent, {"key": key, "value": _clean_value(rawval), "raw": rawval}


def parse_doc(text: str) -> Dict[str, Any]:
    doc: Dict[str, Any] = {"header": [], "neng": [], "ke": [], "she": [], "zhang": []}
    section = "header"
    cur: Optional[Dict[str, Any]] = None
    for line in text.split("\n"):
        if line.strip() == "":
            continue
        bare = line.strip()
        if SEP not in bare and bare in SECTION_NAMES:
            section = SECTION_NAMES[bare]
            cur = None
            continue
        indent, p = _split_line(line)
        if section == "neng":
            if indent == 0 and p["key"] in ("ㄕ", "ㄓㄠ"):
                cur = {"kind": p["key"], "id": p["value"], "fields": []}
                doc["neng"].append(cur)
            elif cur is not None and indent > 0:
                cur["fields"].append(p)
            else:
                raise fail("E-STRUCT-PARSE", "stray line in ㄋㄥ: " + line)
        elif section == "ke":
            if indent == 0 and p["key"] == "ㄑㄩ":
                cur = {"kind": p["key"], "id": p["value"], "fields": []}
                doc["ke"].append(cur)
            elif cur is not None and indent > 0:
                cur["fields"].append(p)
            else:
                raise fail("E-STRUCT-PARSE", "stray line in ㄎㄜ: " + line)
        elif section == "she":
            doc["she"].append(p)
        elif section == "zhang":
            doc["zhang"].append(p)
        else:
            doc["header"].append(p)
    return doc


def header_value(doc: Dict[str, Any], key: str) -> str:
    for p in doc["header"]:
        if p["key"] == key:
            return p["value"]
    return ""


def zhang_value(doc: Dict[str, Any], key: str) -> str:
    for p in doc["zhang"]:
        if p["key"] == key:
            return p["value"]
    return ""


# ---------------------------------------------------------------------------
# Canonical form  (must match Go canonicalDoc + emit* exactly)
# ---------------------------------------------------------------------------
def _emit_value(key: str, value: str) -> str:
    if key not in SET_KEYS or LIST_SEP not in value:
        return value
    return LIST_SEP.join(sorted(value.split(LIST_SEP)))


def _is_ext(key: str) -> bool:
    return key.startswith(EXT_PREFIX)


def _emit_scalar(pairs: List[Dict[str, str]], order: List[str]) -> List[str]:
    seen = set()
    for p in pairs:
        if p["key"] in seen:
            raise fail("E-STRUCT-DUPKEY", p["key"])
        seen.add(p["key"])
    idx = {k: i for i, k in enumerate(order)}
    core, ext = [], []
    for p in pairs:
        if _is_ext(p["key"]):
            ext.append(p)
        elif p["key"] not in idx:
            raise fail("E-TOKEN-UNKNOWN-CORE", p["key"])
        else:
            core.append(p)
    core.sort(key=lambda p: idx[p["key"]])
    ext.sort(key=lambda p: p["key"])
    return [p["key"] + SEP + _emit_value(p["key"], p["value"]) for p in core + ext]


def _emit_entry_fields(fields: List[Dict[str, str]], order: List[str]) -> List[str]:
    idx = {k: i for i, k in enumerate(order)}
    core, ext = [], []
    for i, f in enumerate(fields):
        if _is_ext(f["key"]):
            ext.append(f)
        elif f["key"] not in idx:
            raise fail("E-TOKEN-UNKNOWN-CORE", f["key"])
        else:
            core.append((idx[f["key"]], i, f))
    core.sort(key=lambda t: (t[0], t[1]))
    ext.sort(key=lambda f: f["key"])
    out = ["  " + f["key"] + SEP + _emit_value(f["key"], f["value"]) for _, _, f in core]
    out += ["  " + f["key"] + SEP + _emit_value(f["key"], f["value"]) for f in ext]
    return out


def _emit_entity_fields(fields: List[Dict[str, str]]) -> List[str]:
    seen, core, ext = set(), [], []
    for f in fields:
        if f["key"] in seen:
            raise fail("E-STRUCT-DUPKEY", f["key"])
        seen.add(f["key"])
        (ext if _is_ext(f["key"]) else core).append(f)
    ext.sort(key=lambda f: f["key"])
    return ["  " + f["key"] + SEP + _emit_value(f["key"], f["value"]) for f in core + ext]


def canonical_doc(doc: Dict[str, Any]) -> bytes:
    lines: List[str] = list(_emit_scalar(doc["header"], CORE_HEADER_ORDER))
    entities = [e for e in doc["neng"] if e["kind"] == "ㄕ"]
    actions = [e for e in doc["neng"] if e["kind"] == "ㄓㄠ"]
    if doc["neng"]:
        lines.append("ㄋㄥ")
        for e in sorted(entities, key=lambda e: e["id"]):
            lines.append("ㄕ" + SEP + e["id"])
            lines += _emit_entity_fields(e["fields"])
        for a in sorted(actions, key=lambda e: e["id"]):
            lines.append("ㄓㄠ" + SEP + a["id"])
            lines += _emit_entry_fields(a["fields"], ACTION_FIELD_ORDER)
    if doc["ke"]:
        lines.append("ㄎㄜ")
        for b in sorted(doc["ke"], key=lambda e: e["id"]):
            lines.append("ㄑㄩ" + SEP + b["id"])
            lines += _emit_entry_fields(b["fields"], BLOCK_FIELD_ORDER)
    if doc["she"]:
        lines.append("ㄕㄜ")
        lines += _emit_scalar(doc["she"], PREF_ORDER)
    return ("\n".join(lines) + "\n").encode("utf-8")


def canonicalize_bytes(raw: bytes) -> bytes:
    return canonical_doc(parse_doc(normalize(raw)))


def sha256_hex(b: bytes) -> str:
    return hashlib.sha256(b).hexdigest()


# ---------------------------------------------------------------------------
# Secret gate + backend validation  (mirror Go)
# ---------------------------------------------------------------------------
_JWT_RE = re.compile(r"eyJ[A-Za-z0-9_-]*\.[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+")
_QUERY_RE = re.compile(r"(^|[?&])(token|access_token|api_key)=")
_B64_RE = re.compile(r"^[A-Za-z0-9+/]{48,}={0,2}$")


def scan_secret_string(s: str, path: str, opaque: bool = False) -> None:
    for n in ("Bearer ", "ghp_", "sk-", "AKIA"):
        if n in s:
            raise fail("E-VALUE-SECRET", f"{path} contains {n}")
    if _QUERY_RE.search(s):
        raise fail("E-VALUE-SECRET", f"{path} contains secret query parameter")
    if _JWT_RE.search(s):
        raise fail("E-VALUE-SECRET", f"{path} contains JWT-shaped value")
    if _B64_RE.match(s) and not opaque:
        raise fail("E-VALUE-SECRET", f"{path} contains long base64-shaped value")


def validate_backend_handle(s: str) -> None:
    if not any(s.startswith(p) for p in BACKEND_SCHEMES):
        raise fail("E-BUILDER-SCHEMA", "backend must be mock://, gdrive://, api://, https://, or local://")


# ---------------------------------------------------------------------------
# Ed25519 keygen / sign / verify / trust
# ---------------------------------------------------------------------------
def _require_crypto() -> None:
    if not _HAVE_CRYPTO:
        raise fail("E-CRYPTO", "the 'cryptography' package is required for sign/trust/operate")


def keygen(publisher: str) -> Dict[str, str]:
    _require_crypto()
    priv = Ed25519PrivateKey.generate()
    pub = priv.public_key()
    seed = priv.private_bytes_raw()
    pub_raw = pub.public_bytes_raw()
    return {
        "publisher": publisher,
        "seed_b64": base64.b64encode(seed).decode(),
        "public_key": "ed25519:" + base64.b64encode(pub_raw).decode(),
        "public_key_hex": pub_raw.hex(),
    }


def _load_key(path: str) -> Dict[str, str]:
    with open(path) as f:
        return json.load(f)


def sign_contract(raw: bytes, key: Dict[str, str]) -> bytes:
    _require_crypto()
    priv = Ed25519PrivateKey.from_private_bytes(base64.b64decode(key["seed_b64"]))
    pub_raw = priv.public_key().public_bytes_raw()
    sig = priv.sign(canonicalize_bytes(raw))
    pub_value = "ed25519:" + base64.b64encode(pub_raw).decode()
    sig_value = base64.b64encode(sig).decode()
    out = []
    for line in raw.decode("utf-8").split("\n"):
        if line.startswith("ㄎㄟ" + SEP):
            out.append("ㄎㄟ" + SEP + pub_value)
        elif line.startswith("ㄓㄥ" + SEP):
            out.append("ㄓㄥ" + SEP + sig_value)
        else:
            out.append(line)
    return "\n".join(out).encode("utf-8")


def classify_trust(raw: bytes, trusted_hex: str = "") -> Dict[str, str]:
    _require_crypto()
    doc = parse_doc(normalize(raw))
    if not doc["zhang"]:
        return {"state": "unsigned_draft", "code": "E-TRUST-NOSIG", "badge": "unsigned"}
    pub_value = zhang_value(doc, "ㄎㄟ")
    sig_value = zhang_value(doc, "ㄓㄥ")
    if not sig_value.strip():
        return {"state": "unsigned_draft", "code": "E-TRUST-UNSIGNED", "badge": "draft"}
    if not pub_value.startswith("ed25519:"):
        return {"state": "sig_mismatch", "code": "E-TRUST-BADKEY", "badge": "invalid"}
    try:
        pub_raw = base64.b64decode(pub_value[len("ed25519:"):])
        sig = base64.b64decode(sig_value)
        Ed25519PublicKey.from_public_bytes(pub_raw).verify(sig, canonicalize_bytes(raw))
    except Exception:
        return {"state": "sig_mismatch", "code": "E-TRUST-SIG-MISMATCH", "badge": "invalid",
                "public_key": pub_raw.hex() if 'pub_raw' in dir() else ""}
    pub_hex = pub_raw.hex()
    if trusted_hex and trusted_hex.lower() == pub_hex.lower():
        return {"state": "signed_trusted", "code": "OK", "badge": "trusted", "public_key": pub_hex}
    return {"state": "signed_untrusted_key", "code": "E-TRUST-UNTRUSTED-KEY", "badge": "untrusted",
            "public_key": pub_hex}


# ---------------------------------------------------------------------------
# init  (Block 1)
# ---------------------------------------------------------------------------
def cmd_init(args) -> int:
    with open(args.answers) as f:
        doc = json.load(f)
    backend = args.backend
    if backend is None:
        sys.stderr.write(
            "Where should this app's data live?\n"
            "Enter a backend handle (e.g. mock://board, local://board, gdrive://FILE_ID, api://provider/resource).\n"
            "Do not paste tokens, API keys, or cookies — credentials belong in the Runtime credential store.\n"
            "backend> ")
        sys.stderr.flush()
        backend = sys.stdin.readline()
    backend = backend.strip()
    if not backend:
        raise fail("E-BUILDER-HUMAN-ONLY", "backend is a required human-only decision; none provided")
    scan_secret_string(backend, "app.backend")
    validate_backend_handle(backend)
    doc.setdefault("app", {})["backend"] = backend
    out = args.out or args.answers
    with open(out, "w") as f:
        json.dump(doc, f, ensure_ascii=False, indent=2)
        f.write("\n")
    sys.stderr.write(f"backend recorded (owner=human_only, provenance=user_confirmed): {backend} -> {out}\n")
    return 0


# ---------------------------------------------------------------------------
# build  (replicate Go emitContract)  (Block: front+back source contract)
# ---------------------------------------------------------------------------
def _yesno(v: bool) -> str:
    return "yes" if v else "no"


def _load_template(root: str, template_id: str) -> Dict[str, Any]:
    path = os.path.join(root, "builder", "templates", template_id + ".json")
    with open(path) as f:
        return json.load(f)


def build_contract(answers: Dict[str, Any], tmpl: Dict[str, Any]) -> Tuple[str, List[Dict[str, Any]], Dict[str, str]]:
    decisions = {d["action_id"]: d for d in answers.get("human_decisions", [])}
    kept: List[Dict[str, Any]] = []
    renames: Dict[str, str] = {}
    seen = set()
    for a in tmpl["actions"]:
        a = dict(a)
        d = decisions.get(a["id"], {})
        decision = d.get("decision", "keep")
        if decision == "remove":
            continue
        if d.get("confirm_disabled_by_user"):
            if a.get("risk_class") != "low_mutate":
                raise fail("E-BUILDER-RISK", "confirm can only be disabled for low_mutate: " + a["id"])
            a["confirm"] = False
        if decision == "rename":
            if not d.get("new_id"):
                raise fail("E-BUILDER-SCHEMA", "rename requires new_id: " + a["id"])
            renames[a["id"]] = d["new_id"]
            a["id"] = d["new_id"]
        if a["id"] in seen:
            raise fail("E-BUILDER-SCHEMA", "duplicate action id after decisions: " + a["id"])
        seen.add(a["id"])
        kept.append(a)
    return emit_contract(answers, tmpl, kept, renames), kept, renames


def _renamed(i: str, renames: Dict[str, str]) -> str:
    return renames.get(i, i)


def emit_contract(answers, tmpl, actions, renames) -> str:
    app = answers["app"]
    L = [
        "ㄊㄡ" + SEP + "WA3 v0.3",
        "ㄏㄠ" + SEP + app["id"],
        "ㄉㄞ" + SEP + app["version"],
        "ㄏㄡ" + SEP + app["backend"],
        "ㄈㄢ" + SEP + app["scope"],
        "", "ㄋㄥ",
    ]
    for e in tmpl["entities"]:
        L.append("ㄕ" + SEP + e["id"])
        for fld in e["fields"]:
            L.append("  " + fld["name"] + SEP + fld["type"])
    kept_ids = {a["id"] for a in actions}
    for a in actions:
        L.append("ㄓㄠ" + SEP + a["id"])
        L.append("  ㄘ" + SEP + a["verb"])
        L.append("  ㄓ" + SEP + a["target"])
        L.append("  ㄍㄞ" + SEP + _yesno(a["mutates"]))
        if a["mutates"] or a["confirm"]:
            L.append("  ㄖㄣ" + SEP + _yesno(a["confirm"]))
        for inp in a.get("inputs", []):
            L.append("  ㄕㄡ" + SEP + inp["name"] + LIST_SEP + inp["type"])
        if a.get("outputs"):
            L.append("  ㄉㄜ" + SEP + LIST_SEP.join(a["outputs"]))
    L += ["", "ㄎㄜ"]
    for b in tmpl["blocks"]:
        L.append("ㄑㄩ" + SEP + b["id"])
        L.append("  ㄍㄜ" + SEP + b["type"])
        L.append("  ㄩㄢ" + SEP + _renamed(b["source"], renames))
        refs = [_renamed(i, renames) for i in b.get("actions", []) if _renamed(i, renames) in kept_ids]
        if refs:
            L.append("  ㄊㄠ" + SEP + LIST_SEP.join(refs))
        if b.get("fallback"):
            L.append("  ㄊㄞ" + SEP + LIST_SEP.join(b["fallback"]))
    L += ["", "ㄕㄜ"]
    p = tmpl["preferences"]
    if p.get("readonly"):
        L.append("ㄋㄛ" + SEP + LIST_SEP.join(p["readonly"]))
    hidden = [_renamed(i, renames) for i in p.get("hidden_actions", []) if _renamed(i, renames) in kept_ids]
    if hidden:
        L.append("ㄘㄤ" + SEP + LIST_SEP.join(hidden))
    if p.get("theme_colors"):
        L.append("ㄙㄜ" + SEP + LIST_SEP.join(p["theme_colors"]))
    if p.get("font_sizes"):
        L.append("ㄗ" + SEP + LIST_SEP.join(p["font_sizes"]))
    if p.get("density"):
        L.append("ㄉㄥ" + SEP + p["density"])
    if p.get("visible_blocks"):
        L.append("ㄎㄞ" + SEP + LIST_SEP.join(p["visible_blocks"]))
    L += ["", "ㄓㄤ"]
    L.append("ㄓㄜ" + SEP + answers.get("publisher", {}).get("id", ""))
    L.append("ㄎㄟ" + SEP + "")
    L.append("ㄔㄣ" + SEP + "2026-06-27T00:00:00Z")
    L.append("ㄓㄥ" + SEP + "")
    return "\n".join(L) + "\n"


def cmd_build(args) -> int:
    with open(args.answers) as f:
        answers = json.load(f)
    root = _find_root(args.answers)
    tmpl = answers.get("custom_template") or _load_template(root, answers["template_id"])
    # backend validation gates (same as init, in case build is called directly)
    scan_secret_string(answers["app"]["backend"], "app.backend")
    validate_backend_handle(answers["app"]["backend"])
    contract, kept, _ = build_contract(answers, tmpl)
    raw = contract.encode("utf-8")
    if args.sign:
        key = _load_key(args.key) if args.key else keygen(answers.get("publisher", {}).get("id", "com.local"))
        if not args.key:
            sys.stderr.write("generated ephemeral key; public_key_hex=" + key["public_key_hex"] + "\n")
        raw = sign_contract(raw, key)
        if args.pin_out:
            with open(args.pin_out, "w") as f:
                json.dump({"public_key_hex": key["public_key_hex"]}, f)
    with open(args.out, "wb") as f:
        f.write(raw)
    if args.mock_demo:
        demo = {
            "template_id": tmpl["template_id"],
            "app_id": answers["app"]["id"],
            "backend": answers["app"]["backend"],
            "actions": [{"id": a["id"], "risk_class": a.get("risk_class"),
                         "confirm_required": a["confirm"], "provider": "mock"} for a in kept],
            "sample_data": _mock_sample(tmpl["template_id"]),
        }
        with open(args.mock_demo, "w") as f:
            json.dump(demo, f, ensure_ascii=False, indent=2)
    sys.stderr.write("canonical_sha256=" + sha256_hex(canonicalize_bytes(raw)) + "\n")
    return 0


def _mock_sample(template_id: str) -> Any:
    if template_id == "board":
        return [
            {"id": "msg-1", "author": "Ada", "text": "Hello WA3"},
            {"id": "msg-2", "author": "Lin", "text": "Fallback channel works"},
        ]
    return [{"id": "row-1", "title": "Demo"}]


# ---------------------------------------------------------------------------
# Providers  (Block 2)
# ---------------------------------------------------------------------------
def safe_name(s: str) -> str:
    s = s.strip().lower()
    out = "".join(c for c in s if c.isascii() and (c.isalnum() or c in "_-"))
    return out or "default"


def resource_key(target: str) -> str:
    t = target.lstrip("/")
    if "/" in t:
        t = t.split("/", 1)[0]
    t = t.replace("{", "").replace("}", "")
    return safe_name(t)


def _matches(row: Dict[str, str], q: str) -> bool:
    if not q:
        return True
    q = q.lower()
    return any(q in str(v).lower() for v in row.values())


class MockProvider:
    def __init__(self, handle: str, seed_path: Optional[str]):
        self.namespace = safe_name(handle[len("mock://"):])
        self.data: Dict[str, List[Dict[str, str]]] = {}
        self.seed: List[Dict[str, str]] = []
        if seed_path:
            with open(seed_path) as f:
                demo = json.load(f)
            self.seed = _coerce_rows(demo.get("sample_data"))

    def _ensure(self, r: str):
        if r not in self.data:
            self.data[r] = [dict(x) for x in self.seed]

    def read(self, r, _inp):
        self._ensure(r)
        return self.data[r]

    def search(self, r, inp):
        self._ensure(r)
        return [x for x in self.data[r] if _matches(x, inp.get("q", ""))]

    def write(self, r, inp):
        self._ensure(r)
        row = dict(inp)
        row.setdefault("id", f"{r}-{len(self.data[r]) + 1}")
        self.data[r].append(row)
        return row


class LocalProvider:
    def __init__(self, handle: str, data_dir: str):
        ns = handle[len("local://"):]
        if not ns or ns != safe_name(ns):
            raise fail("E-PROVIDER-PATH", "local:// namespace must be [a-z0-9_-]: " + ns)
        self.root = os.path.abspath(os.path.join(data_dir or "./wa3-data", ns))
        os.makedirs(self.root, exist_ok=True)

    def _path(self, r: str) -> str:
        r = safe_name(r)
        p = os.path.abspath(os.path.join(self.root, r + ".json"))
        if os.path.commonpath([self.root, p]) != self.root:
            raise fail("E-PROVIDER-PATH", "resource escapes data sandbox: " + r)
        return p

    def _load(self, r: str):
        p = self._path(r)
        if not os.path.exists(p):
            return [], p
        with open(p) as f:
            return json.load(f), p

    def read(self, r, _inp):
        return self._load(r)[0]

    def search(self, r, inp):
        return [x for x in self._load(r)[0] if _matches(x, inp.get("q", ""))]

    def write(self, r, inp):
        rows, p = self._load(r)
        row = dict(inp)
        row.setdefault("id", f"{r}-{len(rows) + 1}")
        rows.append(row)
        with open(p, "w") as f:
            json.dump(rows, f, ensure_ascii=False, indent=2)
            f.write("\n")
        return row


def _coerce_rows(sample: Any) -> List[Dict[str, str]]:
    if isinstance(sample, list):
        return [{k: str(v) for k, v in m.items() if isinstance(v, (str, int, float))} for m in sample if isinstance(m, dict)]
    if isinstance(sample, dict):
        for k in sorted(sample):
            if isinstance(sample[k], list) and sample[k]:
                return _coerce_rows(sample[k])
        return _coerce_rows([sample])
    return []


def new_provider(handle: str, data_dir: str, seed_path: Optional[str]):
    if handle.startswith("mock://"):
        return MockProvider(handle, seed_path)
    if handle.startswith("local://"):
        return LocalProvider(handle, data_dir)
    scheme = handle.split("://", 1)[0] + "://" if "://" in handle else handle
    raise fail("E-PROVIDER-SCHEME", f"no provider wired for {scheme} backend (fallback supports mock:// and local:// only)")


# ---------------------------------------------------------------------------
# operate  (Block 3)
# ---------------------------------------------------------------------------
def _find_action(doc, action_id):
    for e in doc["neng"]:
        if e["kind"] == "ㄓㄠ" and e["id"] == action_id:
            return e
    return None


def _field(e, key):
    for p in e["fields"]:
        if p["key"] == key:
            return p["value"].strip()
    return ""


def _declared_inputs(e):
    out = set()
    for p in e["fields"]:
        if p["key"] == "ㄕㄡ":
            name = p["value"].split(LIST_SEP, 1)[0]
            out.add(name.strip())
    return out


def operate(raw: bytes, action_id: str, inputs: Dict[str, str], confirm: bool,
            trusted_hex: str, data_dir: str, seed_path: Optional[str]) -> Dict[str, Any]:
    status = classify_trust(raw, trusted_hex)
    if status["state"] != "signed_trusted":
        hint = " (pin the publisher key with --trusted-pub <hex>)" if status["state"] == "signed_untrusted_key" else ""
        raise fail("E-OPERATE-UNTRUSTED", f"refused: trust state {status['state']} [{status['code']}]{hint}")
    doc = parse_doc(normalize(raw))
    act = _find_action(doc, action_id)
    if act is None:
        raise fail("E-OPERATE-NO-ACTION", "action not found in verified contract: " + action_id)
    verb = _field(act, "ㄘ")
    target = _field(act, "ㄓ")
    mutates = _field(act, "ㄍㄞ") == "yes"
    declared_confirm = _field(act, "ㄖㄣ") == "yes"
    declared = _declared_inputs(act)
    for k, v in inputs.items():
        if k not in declared:
            raise fail("E-OPERATE-INPUT", f"undeclared input for {action_id}: {k}")
        scan_secret_string(v, "input." + k)
    if (mutates or declared_confirm) and not confirm:
        raise fail("E-OPERATE-CONFIRM-REQUIRED",
                   f"action {action_id} mutates the backend; pass --confirm to authorize")
    backend = header_value(doc, "ㄏㄡ").strip()
    if not backend:
        raise fail("E-OPERATE-PROVIDER", "verified contract has no backend (ㄏㄡ)")
    prov = new_provider(backend, data_dir, seed_path)
    resource = resource_key(target)
    if verb == VERB_SEARCH:
        result = prov.search(resource, inputs)
    elif mutates:
        result = prov.write(resource, inputs)
    else:
        result = prov.read(resource, inputs)
    return {"action": action_id, "verb": verb, "resource": resource, "backend": backend,
            "mutated": mutates, "status": "ok", "result": result}


def cmd_operate(args) -> int:
    with open(args.file, "rb") as f:
        raw = f.read()
    inputs = {}
    for kv in args.input or []:
        k, _, v = kv.partition("=")
        if not k:
            raise fail("E-OPERATE-INPUT", "--input must be key=value: " + kv)
        inputs[k] = v
    out = operate(raw, args.action, inputs, args.confirm, args.trusted_pub or "",
                  args.data_dir, args.mock_demo)
    sys.stdout.write(json.dumps(out, ensure_ascii=False, indent=2) + "\n")
    return 0


# ---------------------------------------------------------------------------
# app / serve — the runnable front+back app channel
# ---------------------------------------------------------------------------
def _app_model(doc: Dict[str, Any]) -> Dict[str, Any]:
    """Reduce a verified contract to a render model for the frontend."""
    actions = []
    for e in doc["neng"]:
        if e["kind"] != "ㄓㄠ":
            continue
        actions.append({
            "id": e["id"],
            "verb": _field(e, "ㄘ"),
            "target": _field(e, "ㄓ"),
            "mutates": _field(e, "ㄍㄞ") == "yes",
            "confirm": _field(e, "ㄖㄣ") == "yes",
            "inputs": sorted(_declared_inputs(e)),
            "resource": resource_key(_field(e, "ㄓ")),
        })
    return {"app_id": header_value(doc, "ㄏㄠ"), "backend": header_value(doc, "ㄏㄡ"), "actions": actions}


def render_frontend(model: Dict[str, Any]) -> str:
    return """<!doctype html><html lang="en"><head><meta charset="utf-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<title>__APP__ — WA3 (Python fallback)</title>
<style>
 body{font:15px/1.5 system-ui,sans-serif;max-width:640px;margin:2rem auto;padding:0 1rem;color:#1c2530}
 h1{font-size:1.25rem} .pill{font-size:.7rem;background:#e7f0ff;color:#1a52c4;border-radius:99px;padding:2px 8px}
 .card{border:1px solid #e2e6ec;border-radius:10px;padding:.8rem 1rem;margin:.6rem 0}
 .msg{border-bottom:1px solid #eef1f4;padding:.4rem 0} .msg:last-child{border:0}
 input,button{font:inherit;padding:.45rem .6rem;border-radius:8px;border:1px solid #cdd4dd}
 button{background:#1a52c4;color:#fff;border:0;cursor:pointer} button.ghost{background:#eef1f6;color:#1c2530}
 .row{display:flex;gap:.5rem;flex-wrap:wrap;align-items:center} .muted{color:#6b7785;font-size:.85rem}
 .warn{color:#a33;font-size:.85rem}
</style></head><body>
<h1>__APP__ <span class="pill">verified · python fallback</span></h1>
<p class="muted">backend <code>__BACKEND__</code> — served by the wa3lite operate layer. Mutating actions require core-side confirmation.</p>
<div id="board" class="card"><em class="muted">loading…</em></div>
<div id="controls"></div>
<p id="err" class="warn"></p>
<script>
const MODEL = __MODEL__;
const board = document.getElementById('board'), controls = document.getElementById('controls'), err = document.getElementById('err');
function api(action, inputs, confirm){
  const u = new URL(location.origin + '/api/' + action);
  for(const k in inputs) u.searchParams.set('in.'+k, inputs[k]);
  if(confirm) u.searchParams.set('confirm','1');
  return fetch(u, {method: confirm? 'POST':'GET'}).then(async r=>{const j=await r.json(); if(!r.ok) throw new Error(j.error||r.status); return j;});
}
function esc(s){return String(s).replace(/[&<>]/g,c=>({'&':'&amp;','<':'&lt;','>':'&gt;'}[c]));}
const readAct = MODEL.actions.find(a=>a.verb==='ㄎㄢ');
const searchAct = MODEL.actions.find(a=>a.verb==='ㄙㄡ');
const writeActs = MODEL.actions.filter(a=>a.mutates);
function renderRows(rows){
  board.innerHTML = rows.length? rows.map(m=>`<div class="msg"><b>${esc(m.author||m.title||m.id)}</b> ${esc(m.text||m.summary||'')}</div>`).join('') : '<em class="muted">no records</em>';
}
function refresh(){ if(readAct) api(readAct.id,{}).then(j=>renderRows(j.result)).catch(e=>err.textContent=e.message); }
writeActs.forEach(a=>{
  const c=document.createElement('div'); c.className='card';
  c.innerHTML=`<div class="muted">${esc(a.id)} ${a.confirm?'· needs confirm':''}</div>`;
  const row=document.createElement('div'); row.className='row'; row.style.marginTop='.5rem';
  const fields={};
  a.inputs.forEach(name=>{const i=document.createElement('input'); i.placeholder=name; row.appendChild(i); fields[name]=i;});
  const b=document.createElement('button'); b.textContent=a.id;
  b.onclick=()=>{const vals={}; for(const k in fields) vals[k]=fields[k].value;
    if(a.confirm && !window.confirm('Confirm mutating action: '+a.id+'?')) return;
    api(a.id, vals, true).then(()=>{for(const k in fields)fields[k].value='';refresh();}).catch(e=>err.textContent=e.message);};
  row.appendChild(b); c.appendChild(row); controls.appendChild(c);
});
if(searchAct){
  const c=document.createElement('div'); c.className='card';
  const row=document.createElement('div'); row.className='row';
  const i=document.createElement('input'); i.placeholder='search…';
  const b=document.createElement('button'); b.className='ghost'; b.textContent='search';
  const clr=document.createElement('button'); clr.className='ghost'; clr.textContent='clear';
  b.onclick=()=>api(searchAct.id,{q:i.value}).then(j=>renderRows(j.result)).catch(e=>err.textContent=e.message);
  clr.onclick=()=>{i.value='';refresh();};
  row.append(i,b,clr); c.appendChild(row); controls.appendChild(c);
}
refresh();
</script></body></html>""".replace("__APP__", model["app_id"]).replace("__BACKEND__", model["backend"]).replace("__MODEL__", json.dumps(model, ensure_ascii=False))


def cmd_app(args) -> int:
    with open(args.contract, "rb") as f:
        raw = f.read()
    status = classify_trust(raw, args.trusted_pub or "")
    if status["state"] != "signed_trusted":
        raise fail("E-OPERATE-UNTRUSTED", f"cannot build app from {status['state']} contract [{status['code']}]")
    model = _app_model(parse_doc(normalize(raw)))
    out_dir = args.out_dir or "./wa3-app"
    os.makedirs(out_dir, exist_ok=True)
    with open(os.path.join(out_dir, "index.html"), "w") as f:
        f.write(render_frontend(model))
    sys.stderr.write(f"front-end written to {out_dir}/index.html\nrun the backend with: wa3lite serve --contract {args.contract} "
                     f"--trusted-pub {status.get('public_key','<hex>')}\n")
    return 0


def cmd_serve(args) -> int:
    import http.server
    import urllib.parse
    with open(args.contract, "rb") as f:
        raw = f.read()
    status = classify_trust(raw, args.trusted_pub or "")
    if status["state"] != "signed_trusted":
        raise fail("E-OPERATE-UNTRUSTED", f"refuse to serve {status['state']} contract [{status['code']}]")
    model = _app_model(parse_doc(normalize(raw)))
    html = render_frontend(model).encode("utf-8")
    trusted = args.trusted_pub or ""
    data_dir, seed = args.data_dir, args.mock_demo

    class H(http.server.BaseHTTPRequestHandler):
        def _json(self, code, obj):
            b = json.dumps(obj, ensure_ascii=False).encode("utf-8")
            self.send_response(code)
            self.send_header("Content-Type", "application/json; charset=utf-8")
            self.send_header("Content-Length", str(len(b)))
            self.end_headers()
            self.wfile.write(b)

        def _do(self):
            u = urllib.parse.urlparse(self.path)
            if u.path in ("/", "/index.html"):
                self.send_response(200)
                self.send_header("Content-Type", "text/html; charset=utf-8")
                self.send_header("Content-Length", str(len(html)))
                self.end_headers()
                self.wfile.write(html)
                return
            if u.path.startswith("/api/"):
                action = u.path[len("/api/"):]
                q = urllib.parse.parse_qs(u.query)
                inputs = {k[3:]: v[0] for k, v in q.items() if k.startswith("in.")}
                confirm = "confirm" in q
                try:
                    out = operate(raw, action, inputs, confirm, trusted, data_dir, seed)
                    self._json(200, out)
                except Wa3Error as e:
                    self._json(403 if e.code.startswith(("E-OPERATE", "E-PROVIDER", "E-VALUE")) else 400,
                               {"error": str(e), "code": e.code})
                return
            self._json(404, {"error": "not found"})

        do_GET = _do
        do_POST = _do

        def log_message(self, *a):
            pass

    srv = http.server.HTTPServer(("127.0.0.1", args.port), H)
    sys.stderr.write(f"serving {model['app_id']} on http://127.0.0.1:{args.port}  (backend {model['backend']})\n")
    try:
        srv.serve_forever()
    except KeyboardInterrupt:
        pass
    return 0


# ---------------------------------------------------------------------------
# canonical / trust CLI + dispatch
# ---------------------------------------------------------------------------
def cmd_canonical(args) -> int:
    with open(args.file, "rb") as f:
        out = canonicalize_bytes(f.read())
    sys.stdout.buffer.write(out)
    sys.stderr.write("sha256=" + sha256_hex(out) + "\n")
    return 0


def cmd_keygen(args) -> int:
    key = keygen(args.publisher)
    with open(args.key_out, "w") as f:
        json.dump(key, f, indent=2)
    sys.stderr.write("public_key_hex=" + key["public_key_hex"] + "\n")
    return 0


def cmd_sign(args) -> int:
    with open(args.infile, "rb") as f:
        raw = f.read()
    signed = sign_contract(raw, _load_key(args.key_file))
    with open(args.out, "wb") as f:
        f.write(signed)
    return 0


def cmd_trust(args) -> int:
    with open(args.file, "rb") as f:
        status = classify_trust(f.read(), args.trusted_pub or "")
    sys.stdout.write(json.dumps(status, ensure_ascii=False) + "\n")
    return 0


def _find_root(start: str) -> str:
    d = os.path.abspath(os.path.dirname(start) or ".")
    while True:
        if os.path.isdir(os.path.join(d, "builder", "templates")):
            return d
        parent = os.path.dirname(d)
        if parent == d:
            return os.path.abspath(".")
        d = parent


def main(argv=None) -> int:
    p = argparse.ArgumentParser(prog="wa3lite", description="WA3 pure-Python fallback pipeline")
    sub = p.add_subparsers(dest="cmd", required=True)

    s = sub.add_parser("canonical"); s.add_argument("file"); s.set_defaults(fn=cmd_canonical)

    s = sub.add_parser("init")
    s.add_argument("--answers", required=True); s.add_argument("--backend"); s.add_argument("--out")
    s.set_defaults(fn=cmd_init)

    s = sub.add_parser("build")
    s.add_argument("--answers", required=True); s.add_argument("--out", required=True)
    s.add_argument("--sign", action="store_true"); s.add_argument("--key")
    s.add_argument("--pin-out"); s.add_argument("--mock-demo")
    s.set_defaults(fn=cmd_build)

    s = sub.add_parser("keygen")
    s.add_argument("--publisher", required=True); s.add_argument("--key-out", required=True)
    s.set_defaults(fn=cmd_keygen)

    s = sub.add_parser("sign")
    s.add_argument("--key-file", required=True, dest="key_file")
    s.add_argument("--in", required=True, dest="infile"); s.add_argument("--out", required=True)
    s.set_defaults(fn=cmd_sign)

    s = sub.add_parser("trust"); s.add_argument("file"); s.add_argument("--trusted-pub")
    s.set_defaults(fn=cmd_trust)

    s = sub.add_parser("operate")
    s.add_argument("file"); s.add_argument("--action", required=True)
    s.add_argument("--input", action="append"); s.add_argument("--confirm", action="store_true")
    s.add_argument("--trusted-pub"); s.add_argument("--data-dir", default="./wa3-data")
    s.add_argument("--mock-demo")
    s.set_defaults(fn=cmd_operate)

    s = sub.add_parser("app")
    s.add_argument("--contract", required=True); s.add_argument("--trusted-pub")
    s.add_argument("--out-dir")
    s.set_defaults(fn=cmd_app)

    s = sub.add_parser("serve")
    s.add_argument("--contract", required=True); s.add_argument("--trusted-pub")
    s.add_argument("--data-dir", default="./wa3-data"); s.add_argument("--mock-demo")
    s.add_argument("--port", type=int, default=8770)
    s.set_defaults(fn=cmd_serve)

    args = p.parse_args(argv)
    try:
        return args.fn(args)
    except Wa3Error as e:
        sys.stderr.write(str(e) + "\n")
        return 1


if __name__ == "__main__":
    sys.exit(main())
