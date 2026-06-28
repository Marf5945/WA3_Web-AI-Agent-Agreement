# W3A for LangGraph

LangGraph is not a folder-skill install target in the same sense as Codex or
OpenHands. Treat W3A as an assistant/graph integration: the graph collects guided
answers, runs the deterministic W3A builder, and returns draft/trust artifacts.

## Files

- `langgraph.json` - example graph manifest.
- `w3a_builder_graph.py` - minimal graph-shaped example for wrapping the W3A CLI.

## Integration Boundary

- The graph may ask questions and collect answers.
- The graph must not decide `risk_class`, trust state, or canonical validity.
- The graph calls the W3A CLI or equivalent deterministic implementation for:
  - build
  - secret scan
  - risk gate
  - canonicalization
  - trust enum

## Prompt To Install

```text
Load adapters/langgraph/README.md and use langgraph.json as the assistant
integration sketch. Do not treat this as a native skill install. Wire the graph
to call conformance/tools/w3a for build and trust checks.
```
