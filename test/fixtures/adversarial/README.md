# Adversarial Bundle Fixtures

| Scenario | Modified field or mutation | Expected verification outcome |
| --- | --- | --- |
| `TestTamperEventType` | `event.type` on record index 2 | verification error |
| `TestTamperEventData` | `event.data` on record index 2 | verification error |
| `TestTamperPrevHash` | `event.prev_hash` on record index 2 | verification error |
| `TestTamperHashField` | `hash` on record index 2 | verification error |
| `TestDeleteMiddleRecord` | delete record index 2 from a 4-record bundle | verification error |
| `TestTruncateBundle` | remove the last record from a 3-record bundle | verification error |
| `TestDuplicateRecord` | insert a copy of record index 2 into the sequence | verification error |
| `TestSwapRecords` | swap record structs at indexes 2 and 3 | verification error |
| `TestTamperSequenceNumber` | `event.seq` on record index 2 set to `99` | verification error |
| `TestAppendAfterSign` | append a record after signing the bundle | verification error |

`TestTruncateBundle` is currently skipped in `internal/bundle` because tail truncation of an unsigned bundle preserves a valid hash-chain prefix. Detecting that case requires a terminal commitment outside plain `Bundle.Verify()`.
