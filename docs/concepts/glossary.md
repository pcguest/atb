# ATB glossary

These terms are canonical across current ATB documentation.

## Tenon

The umbrella product identity containing ATB and Mortise. Tenon is not an ATB
runtime component, hosted dependency, or verifier.

## ATB

The open-source, local-first evidence core for independently verifiable records
of AI-agent behaviour.

## Mortise

The optional commercial custody and organisational layer for ATB evidence.

## Bundle

A portable `.atb` evidence record containing canonicalised, hash-chained
records.

## Event

Semantic activity submitted to ATB through an SDK, capture, interception,
import, MCP, OTel, or explicit append path.

## Record

The canonicalised event and its stored chain hash as represented in a bundle.

## Capture

The process through which external activity becomes ATB evidence. Capture is
bounded by the integration used; ATB does not prove that all relevant activity
was captured.

## Verification

Independent evaluation of bundle integrity and configured evidence properties,
including supported signatures, timestamp evidence, and profiles.

## Profile

A declarative ATB evidence-obligation model evaluated against recorded
evidence.

## CAS

Completeness Assurance Score. A local estimate of profile-scoped evidence
coverage within recorded bundle events. CAS is not universal capture
completeness, an external audit opinion, or certification.

## Incident finding

A deterministic observation derived from recorded evidence and linked to the
relevant record sequences and hashes.

## Evidence pack / compliance pack

A portable collection of the bundle and derived verification, profile,
incident, mapping, manifest, and checksum artefacts. `compliance pack` is a CLI
command name for a mapping-oriented evidence pack; it is not legal
certification.

## Custody

External preservation and control of evidence after creation. Local bundle
integrity is distinct from external custody.

## Tamper-evident

A property that makes post-recording mutation, insertion, deletion, or reorder
detectable when the chain is verified. It does not mean the local file cannot
be replaced or rolled back.
