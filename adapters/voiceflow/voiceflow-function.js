// Example Voiceflow function wrapper for a W3A builder backend.
// Keep secrets in the backend/runtime credential store, not in this file.

export default async function main(args) {
  const endpoint = args?.w3aBuilderEndpoint;
  if (!endpoint) {
    return {
      ok: false,
      error_code: "E-CONFIG-MISSING",
      repair_hint: "Configure a W3A builder backend endpoint."
    };
  }

  const response = await fetch(endpoint, {
    method: "POST",
    headers: {"content-type": "application/json"},
    body: JSON.stringify({answers: args?.answers ?? {}})
  });

  const body = await response.json();
  return {
    ok: response.ok && body.ok === true,
    state: body.state ?? "",
    error_code: body.error_code ?? "",
    repair_hint: body.repair_hint ?? ""
  };
}
