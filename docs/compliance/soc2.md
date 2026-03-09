# SOC 2 Type II Export Specification

This document defines the schema and control mapping for generating SOC 2 audit artifacts from ATB bundles. The goal is to provide auditors with cryptographically verifiable evidence of system integrity, access control, and change management.

## 1. Export Command Usage

```bash
atb export --format soc2 --bundle <path> --output <dir>