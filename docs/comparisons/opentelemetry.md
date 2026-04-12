# ATB vs OpenTelemetry

## What they share

Both ATB and OpenTelemetry capture structured event data from application workflows. Both support trace context propagation via W3C Trace Context headers. ATB's `trace_id` and `span_id` fields correlate directly with OTel spans, so the two systems can reference the same workflow execution without duplication.

## Where they differ

| | ATB | OpenTelemetry |
| --- | --- | --- |
| Primary purpose | Portable, tamper-evident audit trails for AI and high-risk workflow evidence | Distributed tracing, metrics, logs, and operational observability |
| Integrity guarantee | Hash-chained bundle records that can be verified locally | Telemetry pipelines focus on transport and analysis, not tamper-evident artefacts |
| Local-first operation | Designed to write and keep bundles locally by default | Often deployed through collectors and central backends, though self-hosting is possible |
| Compliance export | Deterministic evidence exports for audit, handoff, and review workflows | Exporters and backends can support reporting, but compliance evidence packaging is not the primary product shape |
| Profile-based verification | Built-in workflow profiles evaluate expected evidence and CAS completeness | No equivalent profile verifier for workflow evidence completeness |
| Storage model | Portable `.atb` bundle files | Telemetry streams sent to collectors, brokers, and trace stores |
| Privacy posture | Favourable for privacy-sensitive local retention and deliberate export | Commonly optimised for central collection, retention, and cross-service analysis |

## Using them together

The recommended pattern is to use OpenTelemetry for operational observability and ATB for compliance evidence on the same workflow. OTel gives you service-level latency and dependency insight, while ATB gives you a portable artefact for review, audit, or handoff. Correlate the two with the same `trace_id` so an operator can move from a trace in the APM system to the matching ATB bundle evidence.

```ts
function handlePrivilegedAction(request: Request) {
  const span = otel.startSpan("privileged_action");
  const traceId = span.spanContext().traceId;

  bundle.append("ai.request.received", {
    request_id: request.id,
    actor_id_hash: request.actorHash,
    purpose_tag: "privileged_tool_demo",
    trace_id: traceId,
  });

  const result = executeAction(request);
  span.end();
  return result;
}
```

## When to choose ATB

- Privacy-sensitive workflows where sending traces to a hosted observability backend is not acceptable
- Compliance requirements for a tamper-evident portable audit artefact that travels with the workflow
- Teams that need verifiable evidence for incident review or regulatory handoff without a persistent backend dependency

## When to choose OTel

- Distributed tracing across many services with latency profiling
- Integration with existing APM infrastructure (Datadog, Honeycomb, Grafana Tempo, etc.)
- No compliance requirement for tamper-evident local evidence
