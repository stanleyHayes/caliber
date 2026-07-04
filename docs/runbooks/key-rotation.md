# Field-encryption key rotation (CAL-117)

How to rotate — or first enable — the AES-256 key that encrypts candidate PII at
rest (`CALIBER_FIELD_ENCRYPTION_KEY`). This complements
[secret-rotation.md](secret-rotation.md) (which covers the other secrets) and the
at-rest design in [data-protection.md](../data-protection.md).

## What is encrypted

The field cipher (`internal/adapters/outbound/fieldcrypto`, AES-256-GCM,
`enc:v1:<base64>`) protects these columns, transparently to callers:

| Table | Columns |
|---|---|
| `candidates` | `location`, `preferences` (intake: salary floor, deal-breakers) |
| `talent_profiles` | `summary`, `profile` (evidenced competencies — verbatim CV quotes) |
| `talent_interviews` + `interview_turns` | `question`, `answer`, `report_card` (transcript evidence) |
| `matches` | `rationale`, `breakdown`, `watch_outs` |

## Key facts

- **The marker does not name the key.** A stored value is `enc:v1:…` regardless of
  which key sealed it. On read, the cipher tries the **primary** key first, then
  each **previous** key (GCM authentication makes a wrong key fail cleanly).
- **Writes always use the primary key** (`CALIBER_FIELD_ENCRYPTION_KEY`).
- **`CALIBER_FIELD_ENCRYPTION_KEY_PREVIOUS`** (comma-separated base64 keys) is
  decrypt-only — the retiring key(s) that reads should still accept during a
  rotation window.
- **`reencrypt`** (`cmd/reencrypt`) rewrites every PII row through the cipher,
  re-sealing it with the current primary key. It is idempotent and safe to re-run.
  It refuses to run without a primary key.

## Generate a key

```bash
openssl rand -base64 32
```

## Rotation procedure (zero-downtime)

Rotating from OLD → NEW while the app stays up:

1. **Add the new key as primary, keep the old as previous.** In the secret store,
   set:
   - `CALIBER_FIELD_ENCRYPTION_KEY` = NEW key
   - `CALIBER_FIELD_ENCRYPTION_KEY_PREVIOUS` = OLD key

   Redeploy the API and worker. New writes now use NEW; reads still open OLD-key
   rows via the fallback. **No data is unreadable at any point.**

2. **Re-encrypt stored rows** — run the migration command with the same two keys
   set (so it can read OLD rows and write NEW):

   ```bash
   CALIBER_DATABASE_URL=… \
   CALIBER_FIELD_ENCRYPTION_KEY=<NEW> \
   CALIBER_FIELD_ENCRYPTION_KEY_PREVIOUS=<OLD> \
   go run ./cmd/reencrypt        # or the built binary
   ```

   It logs a count per entity type when done. Re-run if it is interrupted — it is
   idempotent (rows already on the new key re-seal to the same key).

3. **Retire the old key.** Once `reencrypt` has completed successfully, remove
   `CALIBER_FIELD_ENCRYPTION_KEY_PREVIOUS` from the secret store and redeploy. The
   old key is now fully out of rotation; verify a candidate profile still loads.

## First-time enablement (plaintext → encrypted)

A store that ran with **no** key holds plaintext rows. To turn encryption on:

1. Set `CALIBER_FIELD_ENCRYPTION_KEY` = NEW key (leave `_PREVIOUS` empty) and
   redeploy — new writes encrypt; existing plaintext reads pass through.
2. Run `reencrypt` (with only the primary key) to encrypt the existing rows.

> **Edge case:** a *plaintext* value written pre-encryption that literally starts
> with `enc:v1:` is ambiguous once a key is set. This only occurs on a dirty dev
> store (production is keyed from the first write). If `reencrypt` reports a
> decrypt failure on such a row, fix that row manually before retrying.

## If the key is lost

There is no recovery: AES-256-GCM ciphertext cannot be read without the key. Keep
the key in the managed secret store with the same backup/rotation discipline as
`CALIBER_JWT_SECRET`. If the **only** copy is lost, the encrypted PII is
unrecoverable and the affected rows must be re-collected.
