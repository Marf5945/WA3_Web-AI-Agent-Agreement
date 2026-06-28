#!/usr/bin/env python3
"""WA3 reference runtime (Python) — fills the three gaps around the backend:

  1. init      : interactively (or via --backend) ASK where data is stored,
                 validate the handle, block secrets, write answers JSON.
  2. providers : a pluggable Provider interface; mock:// (in-memory) and
                 local:// (persists to a JSON file on disk) implemented.
  3. operate   : verify trust -> resolve action -> resolve ㄏㄡ backend handle to
                 a provider -> enforce confirmation for mutating actions
                 (core-side, not renderer) -> execute read/search/write.

This is a REFERENCE prototype meant to be runnable and verifiable. The
authoritative implementation is Go (conformance/tools/wa3). Cryptographic
Ed25519 verification is the Go/runtime job; here trust is structural
(unsigned drafts are rejected; signed contracts are accepted with a warning
that signatures are not cryptographically verified in this prototype).
"""
import sys, os, json, re, argparse, unicodedata

# ---- zhuyin keys (subset we need) -----------------------------------------
K_TITLE, K_APP, K_VER, K_BACKEND, K_SCOPE = 'ㄊㄡ', 'ㄏㄠ', 'ㄉㄞ', 'ㄏㄡ', 'ㄈㄢ'
SEC_ACT, SEC_BLK, SEC_PREF, SEC_SIG, ENT = 'ㄋㄥ', 'ㄎㄜ', 'ㄕㄜ', 'ㄓㄤ', 'ㄕ'
A_ID, A_VERB, A_PATH, A_MUT, A_CONF, A_IN = 'ㄓㄠ', 'ㄘ', 'ㄓ', 'ㄍㄞ', 'ㄖㄣ', 'ㄕㄡ'
SIG_VAL = 'ㄓㄥ'
VERB = {'ㄎㄢ': 'read', 'ㄔㄥ': 'submit', 'ㄢ': 'react', 'ㄗㄜ': 'select',
        'ㄙㄡ': 'search', 'ㄕㄢ': 'delete'}

# Backend handle policy (mirrors Go validateBackendHandle + §10.1 secret gate)
ALLOWED_SCHEMES = ('mock://', 'local://', 'gdrive://', 'api://', 'https://')
IMPLEMENTED_SCHEMES = ('mock://', 'local://')
SECRET_PATTERNS = [r'Bearer\s', r'ghp_[A-Za-z0-9]', r'sk-[A-Za-z0-9]', r'AKIA[0-9A-Z]{8}',
                   r'eyJ[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+\.']  # JWT-ish


class WA3Error(Exception):
    def __init__(self, code, msg):
        super().__init__(f'{code}: {msg}'); self.code = code; self.msg = msg


# ---- parsing ---------------------------------------------------------------
def parse_tdy(text):
    doc = {'meta': {}, 'backend': None, 'scope': None, 'actions': [], 'sig': {}}
    sec, cur = None, None
    for raw in text.splitlines():
        if not raw.strip():
            continue
        indented = raw[:1] in (' ', '\t')
        key = raw.strip().split('：')[0]
        val = raw.split('：', 1)[1].strip() if '：' in raw else ''
        if not indented:
            if key == K_TITLE: doc['meta']['title'] = val
            elif key == K_APP: doc['meta']['app'] = val
            elif key == K_VER: doc['meta']['ver'] = val
            elif key == K_BACKEND: doc['backend'] = val
            elif key == K_SCOPE: doc['scope'] = val
            elif key == SEC_ACT: sec = SEC_ACT
            elif key == SEC_BLK: sec = SEC_BLK
            elif key == SEC_PREF: sec = SEC_PREF
            elif key == SEC_SIG: sec = SEC_SIG
            elif key == A_ID:
                cur = {'id': val, 'inputs': []}; doc['actions'].append(cur); sec = SEC_ACT
            elif key == ENT:
                cur = None; sec = SEC_ACT
            elif sec == SEC_SIG and key:
                doc['sig'][key] = val
        else:
            if sec == SEC_ACT and cur is not None:
                if key == A_VERB: cur['verb'] = val
                elif key == A_PATH: cur['path'] = val
                elif key == A_MUT: cur['mutates'] = (val == 'yes')
                elif key == A_CONF: cur['confirm'] = (val == 'yes')
                elif key == A_IN: cur['inputs'].append(val.split('｜')[0])
    return doc


# ---- 1. backend handle validation + init ----------------------------------
def validate_backend(handle):
    if not handle:
        raise WA3Error('E-BACKEND-EMPTY', 'backend handle is required (where should data live?)')
    for pat in SECRET_PATTERNS:
        if re.search(pat, handle):
            raise WA3Error('E-VALUE-SECRET',
                           'backend looks like a secret/token. Put credentials in the '
                           'Runtime credential store and use a stable handle '
                           '(mock://, local://, gdrive://ID, api://provider/resource).')
    if not handle.startswith(ALLOWED_SCHEMES):
        raise WA3Error('E-BACKEND-SCHEME',
                       f'backend must start with one of {", ".join(ALLOWED_SCHEMES)}')
    return handle


def cmd_init(args):
    """Ask the user WHERE backend data should live, validate, write answers JSON.
    --backend makes it non-interactive (for scripts/CI)."""
    answers = {}
    if args.answers and os.path.exists(args.answers):
        answers = json.load(open(args.answers, encoding='utf-8'))
    app = answers.setdefault('app', {})

    handle = args.backend
    if handle is None:  # interactive
        print('Where should this app\'s data be stored?')
        print('  examples: mock://board (demo) | local://board (this machine) |')
        print('            gdrive://FILE_ID | api://provider/resource')
        print('  (do NOT paste tokens/keys — credentials go to the Runtime credential store)')
        while True:
            try:
                handle = input('backend> ').strip()
            except EOFError:
                raise WA3Error('E-BACKEND-EMPTY', 'no input for backend handle')
            try:
                validate_backend(handle); break
            except WA3Error as e:
                print(f'  ✗ {e.code}: {e.msg}')
    else:
        validate_backend(handle)

    app['backend'] = handle
    # backend is a HUMAN-ONLY decision (§10.1): provenance is user_confirmed.
    answers.setdefault('_provenance', {})['app.backend'] = 'user_confirmed'
    out = args.answers or 'answers.json'
    json.dump(answers, open(out, 'w', encoding='utf-8'), ensure_ascii=False, indent=2)
    impl = '' if handle.startswith(IMPLEMENTED_SCHEMES) else \
        '  (note: scheme not wired up yet — deferred to v1.2; only mock:// and local:// run today)'
    print(f'✓ backend recorded: {handle}{impl}')
    print(f'✓ written to {out} (app.backend, provenance=user_confirmed)')
    return 0


# ---- 2. provider interface -------------------------------------------------
class Provider:
    def read(self, resource): raise NotImplementedError
    def write(self, resource, record): raise NotImplementedError
    def search(self, resource, q): raise NotImplementedError


class MockProvider(Provider):
    """In-memory fake backend for mock://. Optionally seeded."""
    def __init__(self, seed=None):
        self.store = {}
        if seed:
            for r, items in seed.items():
                self.store[r] = list(items)

    def read(self, resource):
        return list(self.store.get(self._base(resource), []))

    def write(self, resource, record):
        self.store.setdefault(self._base(resource), []).append(record)
        return record

    def search(self, resource, q):
        q = (q or '').lower()
        return [r for r in self.read(resource)
                if any(q in str(v).lower() for v in r.values())]

    @staticmethod
    def _base(resource):  # /messages/{id}/mood -> /messages
        return '/' + resource.strip('/').split('/')[0] if resource.strip('/') else resource


class LocalProvider(Provider):
    """local://<name> -> persists to <data_dir>/<name>.json on disk (real storage)."""
    def __init__(self, handle, data_dir):
        self.name = handle[len('local://'):] or 'default'
        os.makedirs(data_dir, exist_ok=True)
        self.path = os.path.join(data_dir, f'{self.name}.json')

    def _load(self):
        if os.path.exists(self.path):
            return json.load(open(self.path, encoding='utf-8'))
        return {}

    def _save(self, data):
        json.dump(data, open(self.path, 'w', encoding='utf-8'), ensure_ascii=False, indent=2)

    def read(self, resource):
        return list(self._load().get(MockProvider._base(resource), []))

    def write(self, resource, record):
        data = self._load()
        data.setdefault(MockProvider._base(resource), []).append(record)
        self._save(data)
        return record

    def search(self, resource, q):
        q = (q or '').lower()
        return [r for r in self.read(resource)
                if any(q in str(v).lower() for v in r.values())]


def get_provider(handle, data_dir, seed=None):
    if handle.startswith('mock://'):
        return MockProvider(seed)
    if handle.startswith('local://'):
        return LocalProvider(handle, data_dir)
    if handle.startswith(('gdrive://', 'api://', 'https://')):
        raise WA3Error('E-PROVIDER-UNIMPLEMENTED',
                       f'{handle.split("://")[0]}:// provider integration is deferred to v1.2 '
                       '(only mock:// and local:// run today)')
    raise WA3Error('E-BACKEND-SCHEME', f'unknown backend scheme: {handle}')


# ---- 3. operate ------------------------------------------------------------
def trust_state(doc):
    sig = doc['sig'].get(SIG_VAL, '')
    if not sig:
        return 'unsigned_draft'
    if sig == 'SIGNATURE_BASE64':
        return 'placeholder_signed'   # template placeholder, not a real signature
    return 'signed_unverified'        # real-looking sig, but we don't have key material here


def operate(tdy_path, action_id, inp, confirm, data_dir, allow_unverified=False):
    doc = parse_tdy(open(tdy_path, encoding='utf-8').read())

    # --- trust gate (core-side) ---
    st = trust_state(doc)
    if st == 'unsigned_draft':
        raise WA3Error('E-TRUST-UNSIGNED', 'contract is an unsigned draft; refusing to operate')
    if st in ('placeholder_signed', 'signed_unverified') and not allow_unverified:
        raise WA3Error('E-TRUST-UNVERIFIED',
                       f'trust={st}: this reference runtime cannot cryptographically verify '
                       'Ed25519 signatures (that is the Go/runtime job per §8). '
                       'Pass --allow-unverified to proceed in the prototype.')

    # --- resolve action ---
    act = next((a for a in doc['actions'] if a['id'] == action_id), None)
    if act is None:
        raise WA3Error('E-ACTION-UNKNOWN',
                       f'action "{action_id}" not declared. Available: '
                       f'{[a["id"] for a in doc["actions"]]}')

    # --- confirmation gate for mutating actions (core enforces, not renderer) ---
    if act.get('mutates') and not confirm:
        raise WA3Error('E-CONFIRM-REQUIRED',
                       f'action "{action_id}" changes data (ㄍㄞ=yes); core requires confirmation. '
                       'Re-run with --confirm.')

    # --- resolve backend handle -> provider ---
    if not doc['backend']:
        raise WA3Error('E-BACKEND-EMPTY', 'contract has no ㄏㄡ backend handle; run init first')
    validate_backend(doc['backend'])
    seed = {'/messages': [{'id': 'm1', 'author': 'seed', 'text': 'hello WA3'}]} \
        if doc['backend'].startswith('mock://') else None
    provider = get_provider(doc['backend'], data_dir, seed)

    verb = VERB.get(act.get('verb', ''), '?')
    path = act.get('path', '/')
    if verb in ('read',):
        return {'action': action_id, 'verb': verb, 'result': provider.read(path)}
    if verb == 'search':
        return {'action': action_id, 'verb': verb, 'result': provider.search(path, inp.get('q'))}
    if verb in ('submit', 'react'):
        rec = dict(inp)
        rec.setdefault('id', f'm{os.urandom(2).hex()}')
        provider.write(path, rec)
        return {'action': action_id, 'verb': verb, 'wrote': rec, 'backend': doc['backend']}
    raise WA3Error('E-VERB-UNSUPPORTED', f'verb {verb} not supported by this prototype')


def cmd_operate(args):
    inp = {}
    for kv in args.input or []:
        k, _, v = kv.partition('=')
        inp[k] = v
    res = operate(args.file, args.action, inp, args.confirm, args.data_dir, args.allow_unverified)
    print(json.dumps(res, ensure_ascii=False, indent=2))
    return 0


# ---- CLI -------------------------------------------------------------------
def main():
    p = argparse.ArgumentParser(prog='wa3_runtime')
    sub = p.add_subparsers(dest='cmd', required=True)

    pi = sub.add_parser('init', help='ask where backend data is stored; write answers JSON')
    pi.add_argument('--backend', help='non-interactive handle (mock://, local://, ...)')
    pi.add_argument('--answers', help='answers JSON path to create/update', default='answers.json')

    po = sub.add_parser('operate', help='verify -> confirm -> call backend')
    po.add_argument('file'); po.add_argument('--action', required=True)
    po.add_argument('--input', action='append', help='k=v (repeatable)')
    po.add_argument('--confirm', action='store_true')
    po.add_argument('--allow-unverified', action='store_true')
    po.add_argument('--data-dir', default='wa3_data')

    args = p.parse_args()
    try:
        return cmd_init(args) if args.cmd == 'init' else cmd_operate(args)
    except WA3Error as e:
        print(f'✗ {e}', file=sys.stderr)
        return 1


if __name__ == '__main__':
    sys.exit(main())
