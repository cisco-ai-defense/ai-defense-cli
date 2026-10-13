# 008 — Design: `model-scan` multipart upload

## Context

`cisco-aidefense-sdk` 2.2.0 switched model file uploads to S3 multipart
(SDK PR #120). The CLI still creates the scan object without
`use_multipart_upload` and PUTs the whole file to one presigned URL, under the
30 s default request timeout. Large models therefore upload slowly, cannot
resume a dropped connection, and in practice time out.

The scan-manager service (`ai_defense/scan_manager/v1/scan_manager.proto`)
already exposes the multipart contract on the same `/v1/scans/...` surface the
CLI uses for `register`, `objects`, and `run`.

The reference implementation is `aidefense/modelscan/model_scan_base.py`
(`_upload_file_multipart`). Constants, retry rules, URL rules, and error text
match it unless noted under [Differences from the SDK](#differences-from-the-sdk).

## API contract (scan-manager)

| Step | Method + path (relative to `/api/ai-defense/v1`) | Body |
|------|---------------------------------------------------|------|
| Create object | `POST scans/{scan_id}/objects` | `{file_name, size, use_multipart_upload: true}` → `{object_id, multipart_upload: {upload_id, part_size_bytes}}` |
| Part URLs | `POST scans/{scan_id}/objects/{object_id}/multipart/part-urls` | `{upload_id, part_numbers: [1..32 items, 1–10000]}` → `{parts: [{part_number, upload_url}]}` |
| Upload part | `PUT <upload_url>` (S3, no API key) | raw bytes → `ETag` response header |
| Complete | `POST …/multipart/complete` | `{upload_id, parts: [{part_number, etag}]}` |
| Abort | `POST …/multipart/abort` | `{upload_id}` |

`part_size_bytes` is a proto `uint64`, so grpc-gateway may encode it as a JSON
string; it decodes through the existing `FlexInt64`. Responses are accepted in
both snake_case and camelCase, like the other model-scan responses.

When `use_multipart_upload` is set, scan-manager returns only multipart
details (no `upload_url`). It rejects multipart for a zero-byte file, caps
part-URL batches at 32 and uploads at 10,000 parts, and takes the part size
from `MULTIPART_SIZE_MB` (default 64, 5–5120). Part URLs are presigned without
a `Content-Type`, so the CLI sends none.

## Package layout

| File | Change |
|------|--------|
| `api/client.go` | `DoRawPUTWithHeaders(ctx, url, body, len) (http.Header, error)`; `DoRawPUT` delegates to it. `DoRawPUTNoRedirect` for parts. Upload transport keeps 32 idle connections per host. Upload error bodies reduced to a redacted S3 `Code: Message`. |
| `api/errors.go` | New `UploadError` (part number, `Transient`, object-store cause) so `clierror` can tell object-store failures from AI Defense API failures. |
| `internal/clierror/clierror.go` | Classifies `UploadError` before `ApiError` (REQ-03.4a). |
| `api/modelscan/routes.go` | `MultipartPartURLs`, `MultipartComplete`, `MultipartAbort`. |
| `api/modelscan/models.go` | `UseMultipartUpload` on `CreateScanObjectRequest`; `MultipartUpload` on the response; part-URL / complete / abort request and response types. |
| `api/modelscan/multipart.go` | New. `UploadOptions`, `UploadFileWithOptions`, the multipart engine, retry classification, part-URL host check. |
| `api/modelscan/client.go` | `ScanFileWithOptions(path, opts)`; `ScanFile` keeps its signature and uses defaults. `NewClient` raises an unset upload timeout to 300 s (REQ-05). Rejects empty files before registering. |
| `api/modelscan/events.go` | New. `ScanEvents.PollingStarted` callback; `Client.SetEvents`. |
| `internal/cli/modelscan/trigger.go` | `--no-multipart`, `--upload-concurrency`; calls `ScanFileWithOptions` with the upload bar as `Progress`; starts the spinner on `PollingStarted`. |
| `internal/cli/modelscan/progress.go` | New. `uploadProgress` (SDK 2.2.0 upload bar, REQ-06.2) and `statusSpinner` (SDK 2.2.0 polling spinner, REQ-06.5). |

## Upload engine

```text
UploadFileWithOptions(scanID, path, opts)
├─ size == 0 → error "file must not be empty" (no API call)
├─ opts.DisableMultipart → CreateScanObject → single PUT to upload_url
├─ CreateScanObject(use_multipart_upload=true)
│   └─ no multipart_upload → error (no single-PUT fallback)
├─ partCount = ceil(size / partSize); 1..10000 else abort + error
├─ producer goroutine, per batch of ≤32 part numbers:
│     acquire one slot per part (64 slots = two batches outstanding)
│     POST part-urls → verify exact set → queue jobs
├─ N workers (opts.Concurrency):
│     host check (*.amazonaws.com[.cn], https) → SectionReader → DoRawPUTNoRedirect
│     retryable? → backoff 0.5s·2^attempt, refresh URL for this part
│                  (one refresh at a time), retry (≤3)
│     body counted as read → progress; failed attempt → subtract its bytes;
│     success → top up to the part length; record ETag; release slot
├─ first error → cancel context; workers drain; abort (best effort, silent)
└─ all ok → POST complete with parts in ascending order
```

- The slot semaphore reproduces the SDK's pipelining: two batches are fetched
  up front, and the next batch is fetched once 32 slots are free (at most 32
  parts outstanding), so a slow part never stalls the next batch.
- One `*os.File` is opened and shared; `io.NewSectionReader` gives each attempt
  an independent, re-readable reader (`ReadAt` is safe for concurrent use).
- `ETag` values are recorded verbatim (including quotes), as S3 expects.
- Single-part URL refreshes on retry hold `refreshMu`, like the SDK's
  `refresh_lock` (REQ-03.1a): if a whole batch of URLs expires together, the
  refreshes go out one at a time instead of up to 32 at once. The batch
  fetcher does not take the lock (as in the SDK). After taking the lock a
  worker checks the context, so it does not refresh once the upload is
  failing. A failed refresh is an AI Defense API error and is classified as
  one (REQ-03.4b).
- Progress callbacks are serialised with a mutex and fire per body read
  (REQ-06.3), so the CLI throttles redraws.
- The HTTP transport may still read a part body after `Do` returns, so the
  per-attempt `partCounter` is frozen (`stop`) before its bytes are rolled
  back or topped up; late reads are ignored.

### Retry classification

| Error | Retry? |
|-------|--------|
| `*api.TransportError` (timeout, reset, DNS) | yes |
| `*api.ApiError` 429 or ≥ 500 | yes |
| `*api.ApiError` 400/403 with `ExpiredToken` / `RequestExpired` in body | yes (fresh URL) |
| 3xx (redirect not followed), non-S3 part URL, `*api.ValidationError`, missing/oversized ETag, other 4xx | no |

### Errors and exit codes

A failed part returns `*api.UploadError{Part, Transient, Cause}`, whose
`Error()` is only `multipart upload part N failed` (as in the SDK, which
raises `from None`). A failed `--no-multipart` PUT returns the same type with
`Part: 0` (`file upload failed`). `Transient` is true when the last failure
met the retry conditions below.

The cause is an S3 `*api.ApiError` or `*api.TransportError`, the same types
the AI Defense API returns. `clierror.Classify` therefore checks
`UploadError` first (REQ-03.4a); otherwise an S3 403 would become exit 3 with
the API-key hint:

| Upload failure | Exit | Hint |
|----------------|------|------|
| `Transient` (network, 429, 5xx, expired URL, after 3 attempts) | 4 | Check connectivity / proxy to S3, run again |
| Anything else (other 4xx, 3xx, bad ETag, non-S3 part host) | 1 | The object store rejected the upload; not an API-key problem |

The message is the wrapped text plus `UploadError.Detail()`, for example
`failed to upload file: multipart upload part 3 failed (object store returned
status 403: AccessDenied: Access Denied)`. `Detail` uses only the sanitized
S3 code and message (REQ-03.6) or the transport error without its URL.
Errors from AI Defense API calls made during the upload (create object, part
URLs, complete) are not wrapped and keep the existing classification.

Empty files and a missing `multipart_upload` are plain errors (exit 1), the
same as an unsupported file type. The `--upload-concurrency` check returns
`clierror.Usage` (2) and runs only for file scans in multipart mode
(REQ-04.2–04.3); `UploadOptions.concurrency()` skips validation when
`DisableMultipart` is set. Both are listed in the exit-code registry.

A failed `abort` is ignored, like the scan delete in `CleanupScanData`; the
original upload error is returned. Part retries write nothing.

## Console output

`trigger` keeps the existing `Scanning file (uploading and processing)...` /
`Scanning repository (processing)...` line and adds the SDK 2.2.0 upload bar
(file scans) and polling spinner (file and repository scans) on stderr
(REQ-06):

```text
Scanning file (uploading and processing)...
Uploading [##############################] 100.0% (467046.4/467046.4 MiB)
| Upload complete. Waiting for scan status...
```

- The bar is redrawn with a leading `\r`, at most every 100 ms plus a final
  draw at 100%, which ends with a newline. The SDK redraws when each part
  completes; the CLI counts bytes as they are sent (REQ-06.3), so it
  throttles instead.
- If the upload fails, `trigger` calls `close`, which ends the unfinished line
  so the error starts on its own line.
- The spinner starts on `ScanEvents.PollingStarted` (fired once, after the
  scan is triggered and before the first poll), draws its first frame
  immediately, then cycles `| / - \` every 100 ms on its own goroutine.
  `trigger` calls `close` before printing the result or error; it stops the
  goroutine and clears the line with `\r`, 64 spaces, `\r`. Repository scans
  upload nothing, so their spinner reads `| Waiting for scan status...`; the
  SDK shows the file text for both.
- Nothing else is written: no retry, abort, scan ID, or status lines. The
  upload engine's output hooks receive only byte counts (`Progress`) and the
  polling start signal (`PollingStarted`). The scan ID appears only in the
  poll-timeout hint (REQ-09).

## Poll budget and timeout guidance

`ScanFileWithOptions` and `ScanRepo` both poll with `pollPlan()`, which
mirrors `DEFAULT_SCAN_RESULT_TIMEOUT_SECONDS` in `cisco-aidefense-sdk`
(PR #127):

```text
either env var set:  interval = WAIT_TIME (default 10 s)
                     polls    = RETRY_COUNT (default 60)
otherwise:           interval = 10 s
                     polls    = 15,000 s / 10 s = 1,500
```

scan-manager's `defaultMaxPollDuration` lets a scan run up to 4 h after it
is triggered, and downloading hundreds of GiB alone takes over an hour. A
15,000 s (4 h 10 min) budget outlasts that window, so the CLI reports the
server's final status (including a server-side timeout) rather than a
client-side timeout. Small scans are unaffected: polling stops at the first
terminal status.

The SDK treats a variable as set when it is present in the environment; the
CLI treats it as set when it is non-empty, matching `envIntOrDefault`.

`api.PollTimeoutError` carries `StatusCommand` and `BudgetEnvVars`, set only
by the `model-scan` client. When `StatusCommand` is set, `clierror.Classify`
builds the hint from them: the scan is still running and was not cancelled,
check it with the status command, and raise the listed variables next time.
It never suggests re-running, which would register a new scan and upload the
file again. Without `StatusCommand` (`mcp-scan run`), the existing hint is
unchanged.

## Security

- Part URLs must be HTTPS on `*.amazonaws.com` / `*.amazonaws.com.cn` and are
  not followed through redirects (SDK parity). They also pass
  `validateUploadURL`. scan-manager's S3 client uses standard AWS endpoints in
  every deployment, so the allowlist does not exclude any supported
  environment. The API key header is never sent to S3.
- Upload concurrency is capped at 32 to bound sockets and memory; at most two
  batches of URLs are outstanding.
- Presigned URL signatures never reach stderr (REQ-03.6). `TransportError`
  already prints only the host, but its cause was a `*url.Error` whose message
  embeds the full URL; upload requests now unwrap it (`withoutURL`) so only the
  underlying network error remains.
- Object-store error bodies are reduced to `Code: Message` (`uploadErrorMessage`).
  A raw `SignatureDoesNotMatch` document echoes the access key ID and string to
  sign; previously the single PUT printed it verbatim. At most 64 KiB is read.
- Bodies that are not S3 XML errors are never echoed (REQ-03.6a): a proxy or
  WAF page can reflect the request URI, including `X-Amz-Credential`,
  `X-Amz-Security-Token`, and `X-Amz-Signature`. The CLI prints
  `unexpected response from object store (N bytes, not shown)`; the HTTP
  status is still reported.
- In the S3 `Code` and `Message` that are shown, presigned-URL parameter
  values are replaced with `REDACTED` (REQ-03.6b), in case an S3-shaped body
  quotes the URL. Each field is capped at 512 bytes after redaction.
- `encoding/xml` does not resolve external entities, so parsing the S3 body
  is not an XXE vector.

## Differences from the SDK

| Area | SDK 2.2.0 | CLI | Reason |
|------|-----------|-----|--------|
| Upload timeout | `requests` `timeout=30` (connect/read idle; no total limit); single PUT has none | `http.Client.Timeout` 300 s per request (each part, or the whole single PUT) | Kept as is; see REQ-05. |
| Console output | Upload bar redrawn per completed part; polling spinner; abort-failure log warning | Existing `Scanning file/repository...` line, the same upload bar (redrawn at most every 100 ms as bytes are sent) and the same spinner; no abort warning | Keep existing CLI lines; byte-level progress (REQ-06.3) needs throttling. |
| Single-PUT size limit | Rejects > 5 GB when multipart is off | No client-side limit (S3 rejects) | Not in scope. |
| Concurrency validation | Inside the multipart path, after the scan is registered | Same modes (file + multipart only), but before any API call | Fail fast without an orphan scan. |
| Upload failure message | `Multipart upload part N failed` only | Same text plus the sanitized object-store status and S3 code; exit 4 if transient, else 1 | A CLI user needs to tell a network problem from a rejected upload. |
| Repository spinner text | `Upload complete. Waiting for scan status...` | `Waiting for scan status...` | Nothing is uploaded for a repository scan. |
| Single-PUT URL | No host check | Existing SSRF checks + redirect re-validation | Pre-existing CLI control. |

## Compatibility

- `ScanFile(path)` keeps its signature (used by tests and callers);
  `ScanFileWithOptions` is additive.
- Default behaviour changes from single PUT to multipart. `--no-multipart`
  restores the old path. Against a server without multipart support, the
  default now fails (REQ-01.3); use `--no-multipart` there.
- Empty files, which the single PUT used to accept, are now rejected.

## Out of scope

- `model-scan list` filters (`severity`, `status`, `scan_date`).
- Resuming an interrupted multipart upload across CLI invocations.

## Test plan

| Requirement | Test |
|-------------|------|
| REQ-01.1–01.2 | httptest server: create (multipart), part-urls, S3 PUTs return ETags; assert complete body has all parts in order and the uploaded bytes reassemble to the file. |
| REQ-01.3 | Create returns only `upload_url` → error, no single PUT. |
| REQ-01.4 | Empty file → error on both paths and via `ScanFileWithOptions`; zero API requests. |
| REQ-01.5 | Part size that yields > 10,000 parts → abort called, error returned. |
| REQ-02.2 | With one worker, part 1 is held until the second batch is fetched; outstanding parts never exceed 64. 70-part upload: every part-urls request has ≤ 32 parts. |
| REQ-02.3 | A part-urls response missing a part fails the upload. |
| REQ-02.4 | Upload transport `MaxIdleConnsPerHost` is 32. |
| REQ-03.1 | First PUT of a part returns 503 / 429 / 403 `RequestExpired` → URL refreshed, retried, succeeds. |
| REQ-03.1a | 32 parts, 32 workers, every first attempt returns 403 `ExpiredToken`: 1 batch + 32 refreshes, at most one `part-urls` request in flight, upload completes. (Fails with 26 concurrent refreshes without the lock.) |
| REQ-03.2–03.3 | Non-retryable 400, missing ETag, and 307 → message is exactly `multipart upload part 1 failed`, non-transient `UploadError` for part 1, cause kept, abort called, no retry. Exhausted 500s → transient `UploadError`. |
| REQ-03.4a–03.4b | `Classify`: rejected part → exit 1 with the part number and S3 detail; exhausted 503 and transport timeout → exit 4; single-PUT failure → `file upload failed`; never "API error" or the API-key hint. An AI Defense API 403 is still exit 3. Single PUT 403/503 → `UploadError` part 0, transient false/true. |
| REQ-03.2a | Abort returns 500 → abort attempted, original `multipart upload part 1 failed` error returned. |
| REQ-03.5 | Non-S3 part host → no PUT, abort; host-suffix table test; no-redirect PUT makes one request. |
| REQ-03.6 | Transport error with a signed URL omits the signature; S3 XML error reduces to `Code: Message`; access key ID never appears. |
| REQ-03.6a–03.6b | Reflected presigned URL in HTML, plain-text, JSON, S3-shaped XML, and SigV2 bodies: no credential, token, or signature placeholder appears; S3 code and message survive with `REDACTED` values. End to end, a 403 HTML page echoing the request URI does not leak the signature. |
| REQ-04.2 | Cobra command tests: invalid values for a multipart file scan → exit 2, no HTTP calls. |
| REQ-04.3 | `--upload-concurrency` 0 / 33 / -4 with `--repo-url` and with `--no-multipart` → scan runs. SDK level: `DisableMultipart` with -1 / 0 / 33 uploads and scans. (Fails without the gate.) |
| REQ-05.1 | `NewClient` with unset upload timeout yields ≥ 300 s upload client timeout. |
| REQ-06.2 | Bar with injected writer and clock: exact SDK line text (0%, partial, 100%, zero total), 100 ms throttle, newline at 100%, updates after completion ignored; `close` ends an unfinished bar and is a no-op otherwise. |
| REQ-06.3 | Fake S3 fails part 2 once: progress drops by that attempt's bytes, never exceeds the size, and ends at the size; single PUT also reports progress. |
| REQ-06.5 | Spinner draws the SDK text immediately and cycles frames; a second `start` is a no-op; `close` stops it and writes the 64-space clear exactly once; `close` without `start` writes nothing. `ScanFileWithOptions` calls `PollingStarted` once, after the trigger. |
| REQ-06.1, 06.4, 06.6–06.7 | `trigger.go` keeps `main`'s stderr lines; the upload API's output hooks carry only byte counts and the polling start. |
| REQ-08.1–08.2 | `pollPlan` table test: no env gives 10 s × 1,500 (15,000 s); retry count only, wait only, both, and an invalid value fall back to the 60 × 10 defaults for the unset part; the default budget exceeds 4 h. |
| REQ-09.1–09.3, 09.5 | `Classify` of a model-scan `PollTimeoutError`: exit 4, hint contains the status command and the model-scan env vars, and never "re-run". Without `StatusCommand` the existing hint is kept. |
| REQ-09.4 | Poll timeout in `ScanFileWithOptions` makes no `DELETE` request. |
