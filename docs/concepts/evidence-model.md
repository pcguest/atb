# Evidence model

ATB records semantic agent and tool activity as events, canonicalises each event
with RFC 8785, and links the resulting records with SHA-256. The portable
`.atb` bundle is the evidence object. Verification and investigation do not
depend on the application that created it.

```text
agent / application
        |
        v
capture boundary (SDK / intercept / import)
        |
        v
semantic event
        |
        v
canonical, hash-chained record
        |
        v
portable .atb bundle
        |
        +--> verify
        +--> incident
        +--> evidence pack
        `--> local viewer
```

An **event** is semantic activity submitted to ATB. A **record** is its stored,
canonicalised representation plus chain hash. A **bundle** is the ordered NDJSON
sequence of those records, beginning with a manifest.

The chain proves the integrity and recorded ordering of the presented records.
It cannot prove that activity omitted before capture never happened, that a
caller-provided actor was truthful, or that the recorded action was correct.
Profiles and CAS evaluate expected evidence within the bundle; they do not turn
that boundary into universal capture completeness.

See the [bundle specification](../specification/bundle-v1.md) for the wire
contract and the [trust model](./trust-model.md) for claims and non-claims.
