package bundle

// BundleProvenanceRetrospective is the provenance tag written into bundle
// metadata for bundles constructed from imported historical logs.
// Retrospective bundles cannot carry RFC 3161 anchoring at original event
// time and must be treated as post-import integrity claims only.
const BundleProvenanceRetrospective = "retrospective-import"
