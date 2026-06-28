# W3A for Mistral Agents / Le Chat

Mistral Agents / Le Chat integrations should treat W3A as a tool-backed
workflow, not a native folder skill install. The assistant can collect guided
answers and call a deterministic W3A builder tool or MCP connector.

## Files

- `mistral-agent.json` - example agent configuration sketch.
- `mcp-connector-example.json` - example MCP connector declaration for a W3A
  builder service.

## Security Boundary

- The model may suggest answers and wording.
- Deterministic code must own schema validation, provenance promotion,
  secret/risk gates, canonicalization, lint, and trust enum classification.
- Do not place credentials or private files in prompts, generated `.w3a`, or
  adapter configuration.

## Prompt To Install

```text
Use adapters/mistral/ as the Mistral integration sketch. Configure an agent tool
or MCP connector that calls a trusted W3A builder service. The model may suggest
answers, but the service must enforce all gates before writing a .w3a file.
```
