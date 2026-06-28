package main

// Provider layer for the operate execution path (Block 2).
//
// A Provider resolves a verified contract's stable backend handle (ㄏㄡ) into
// concrete read/write/search storage operations. It is dispatched by handle
// scheme and is intentionally pluggable: v1 wires mock:// and local:// only,
// while gdrive://, api://, and https:// are reserved for real adapters (v1.2)
// and fail closed here.
//
// Trust invariant: the backend handle is a *stable namespace*, never a
// credential. The builder secret gate (E-VALUE-SECRET) guarantees no token can
// reach ㄏㄡ, and providers never read, cache, accept, or emit credentials.

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// record is a single stored row. Values are plain strings so that nothing in
// the storage layer can carry structured secrets or executable payloads.
type record map[string]string

// Provider is the small, pluggable storage interface. Implementations must be
// credential-free and must not resolve or fetch anything outside their declared
// namespace.
type Provider interface {
	// Read returns every row in a resource collection.
	Read(resource string, input record) (any, error)
	// Write appends a row to a resource collection and returns the stored row.
	Write(resource string, input record) (any, error)
	// Search returns rows in a resource collection whose values match input["q"].
	Search(resource string, input record) (any, error)
}

// newProvider selects a Provider from a verified backend handle. Only the two
// v1 schemes are wired; everything else fails closed with E-PROVIDER-SCHEME.
func newProvider(handle, dataDir, seedPath string) (Provider, error) {
	switch {
	case strings.HasPrefix(handle, "mock://"):
		return newMockProvider(handle, seedPath)
	case strings.HasPrefix(handle, "local://"):
		return newLocalProvider(handle, dataDir)
	default:
		return nil, fail(eProviderScheme,
			"no provider wired for "+schemeOf(handle)+" backend (v1 supports mock:// and local:// only)")
	}
}

// schemeOf returns the scheme portion of a handle for diagnostics. The handle
// itself is a non-secret stable identifier, so it is safe to surface.
func schemeOf(handle string) string {
	if i := strings.Index(handle, "://"); i >= 0 {
		return handle[:i+3]
	}
	return handle
}

// resourceKey derives a flat collection name from an action target path such as
// "/messages", "/messages/search", or "/messages/{id}/reaction". It takes the
// first path segment and strips templating, then sanitises it. This keeps
// provider storage independent of URL shape while staying deterministic.
func resourceKey(target string) string {
	t := strings.TrimPrefix(target, "/")
	if i := strings.IndexByte(t, '/'); i >= 0 {
		t = t[:i]
	}
	t = strings.NewReplacer("{", "", "}", "").Replace(t)
	return safeName(t)
}

// safeName restricts a name to [a-z0-9_-] so it can never traverse paths or
// introduce control characters. Empty input collapses to "default".
func safeName(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	var b strings.Builder
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9', r == '_', r == '-':
			b.WriteRune(r)
		}
	}
	if b.Len() == 0 {
		return "default"
	}
	return b.String()
}

// matchesQuery reports whether any value in row contains q (case-insensitive).
// An empty query matches everything.
func matchesQuery(row record, q string) bool {
	if q == "" {
		return true
	}
	q = strings.ToLower(q)
	for _, v := range row {
		if strings.Contains(strings.ToLower(v), q) {
			return true
		}
	}
	return false
}

// ---- MockProvider: in-memory, seeded from --mock-demo sample_data ----------

type mockProvider struct {
	namespace string
	data      map[string][]record
	seed      []record
}

// newMockProvider builds an in-memory provider for mock://<namespace>. If a
// seed file (the build --mock-demo artifact) is supplied, its sample_data array
// becomes the initial contents of any resource on first access.
func newMockProvider(handle, seedPath string) (*mockProvider, error) {
	m := &mockProvider{
		namespace: safeName(strings.TrimPrefix(handle, "mock://")),
		data:      map[string][]record{},
	}
	if seedPath == "" {
		return m, nil
	}
	raw, err := os.ReadFile(seedPath)
	if err != nil {
		return nil, fail(eProviderSeed, "cannot read mock seed: "+err.Error())
	}
	var demo struct {
		SampleData json.RawMessage `json:"sample_data"`
	}
	if err := json.Unmarshal(raw, &demo); err != nil {
		return nil, fail(eProviderSeed, "invalid mock seed json: "+err.Error())
	}
	m.seed = coerceRows(demo.SampleData)
	return m, nil
}

// ensure lazily initialises a resource from the seed rows so the same seed can
// stand in for whichever collection an action targets.
func (m *mockProvider) ensure(resource string) {
	if _, ok := m.data[resource]; !ok {
		rows := make([]record, len(m.seed))
		for i, r := range m.seed {
			cp := record{}
			for k, v := range r {
				cp[k] = v
			}
			rows[i] = cp
		}
		m.data[resource] = rows
	}
}

func (m *mockProvider) Read(resource string, _ record) (any, error) {
	m.ensure(resource)
	return m.data[resource], nil
}

func (m *mockProvider) Search(resource string, input record) (any, error) {
	m.ensure(resource)
	out := []record{}
	for _, row := range m.data[resource] {
		if matchesQuery(row, input["q"]) {
			out = append(out, row)
		}
	}
	return out, nil
}

func (m *mockProvider) Write(resource string, input record) (any, error) {
	m.ensure(resource)
	row := record{}
	for k, v := range input {
		row[k] = v
	}
	if row["id"] == "" {
		row["id"] = resource + "-" + strconv.Itoa(len(m.data[resource])+1)
	}
	m.data[resource] = append(m.data[resource], row)
	return row, nil
}

// ---- LocalProvider: JSON files under a sandboxed --data-dir ----------------

type localProvider struct {
	root string
}

// newLocalProvider builds a filesystem provider for local://<namespace>. All
// data lives under <dataDir>/<namespace>/; the namespace and every resource are
// validated so nothing can escape the sandbox root.
func newLocalProvider(handle, dataDir string) (*localProvider, error) {
	ns := strings.TrimPrefix(handle, "local://")
	if ns == "" {
		return nil, fail(eProviderPath, "local:// handle needs a namespace, e.g. local://board")
	}
	if ns != safeName(ns) {
		return nil, fail(eProviderPath, "local:// namespace must be [a-z0-9_-]: "+ns)
	}
	if dataDir == "" {
		dataDir = "./wa3-data"
	}
	root := filepath.Join(dataDir, ns)
	if err := os.MkdirAll(root, 0o755); err != nil {
		return nil, fail(eProviderPath, "cannot create data dir: "+err.Error())
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, fail(eProviderPath, err.Error())
	}
	return &localProvider{root: abs}, nil
}

// pathFor resolves a resource collection to a JSON file inside the sandbox and
// rejects any path that would escape root (defense in depth on top of safeName).
func (l *localProvider) pathFor(resource string) (string, error) {
	resource = safeName(resource)
	p := filepath.Join(l.root, resource+".json")
	rel, err := filepath.Rel(l.root, p)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fail(eProviderPath, "resource escapes data sandbox: "+resource)
	}
	return p, nil
}

func (l *localProvider) load(resource string) ([]record, string, error) {
	p, err := l.pathFor(resource)
	if err != nil {
		return nil, "", err
	}
	raw, err := os.ReadFile(p)
	if os.IsNotExist(err) {
		return []record{}, p, nil
	}
	if err != nil {
		return nil, "", fail(eProviderPath, err.Error())
	}
	var rows []record
	if err := json.Unmarshal(raw, &rows); err != nil {
		return nil, "", fail(eProviderPath, "corrupt store "+resource+": "+err.Error())
	}
	return rows, p, nil
}

func (l *localProvider) save(p string, rows []record) error {
	out, err := json.MarshalIndent(rows, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(p, append(out, '\n'), 0o644)
}

func (l *localProvider) Read(resource string, _ record) (any, error) {
	rows, _, err := l.load(resource)
	return rows, err
}

func (l *localProvider) Search(resource string, input record) (any, error) {
	rows, _, err := l.load(resource)
	if err != nil {
		return nil, err
	}
	out := []record{}
	for _, row := range rows {
		if matchesQuery(row, input["q"]) {
			out = append(out, row)
		}
	}
	return out, nil
}

func (l *localProvider) Write(resource string, input record) (any, error) {
	rows, p, err := l.load(resource)
	if err != nil {
		return nil, err
	}
	row := record{}
	for k, v := range input {
		row[k] = v
	}
	if row["id"] == "" {
		row["id"] = resource + "-" + strconv.Itoa(len(rows)+1)
	}
	rows = append(rows, row)
	if err := l.save(p, rows); err != nil {
		return nil, err
	}
	return row, nil
}

// ---- seed coercion ---------------------------------------------------------

// coerceRows flattens mock seed sample_data into a list of string rows. Arrays
// of objects map directly; a single object becomes one row; nested objects are
// best-effort flattened so any template's sample_data can seed a mock store.
func coerceRows(raw json.RawMessage) []record {
	if len(raw) == 0 {
		return nil
	}
	var arr []map[string]any
	if err := json.Unmarshal(raw, &arr); err == nil {
		return rowsFromMaps(arr)
	}
	var obj map[string]any
	if err := json.Unmarshal(raw, &obj); err == nil {
		// Prefer the first array-valued field if present (e.g. solutions,
		// product_units); otherwise treat the object itself as one row.
		keys := make([]string, 0, len(obj))
		for k := range obj {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			if sub, ok := obj[k].([]any); ok {
				ms := []map[string]any{}
				for _, e := range sub {
					if m, ok := e.(map[string]any); ok {
						ms = append(ms, m)
					}
				}
				if len(ms) > 0 {
					return rowsFromMaps(ms)
				}
			}
		}
		return rowsFromMaps([]map[string]any{obj})
	}
	return nil
}

func rowsFromMaps(ms []map[string]any) []record {
	out := make([]record, 0, len(ms))
	for _, m := range ms {
		r := record{}
		for k, v := range m {
			if s, ok := v.(string); ok {
				r[k] = s
			}
		}
		if len(r) > 0 {
			out = append(out, r)
		}
	}
	return out
}
