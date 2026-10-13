# 008 — `model-scan` multipart upload

Story: [AIFW-35569](https://cisco-sbg.atlassian.net/browse/AIFW-35569) — Add support for large model file scans in SDK and CLI.

## Goal

Bring `model-scan trigger -f` to parity with `cisco-aidefense-sdk` 2.2.0 file
uploads: upload model files to the presigned S3 multipart API in parallel
parts with per-part retries, so large models upload reliably.

Requirements use EARS phrasing.

## Requirements

### REQ-01 — Multipart upload is the default

- **REQ-01.1** When `model-scan trigger --file-path` uploads a non-empty file,
  the CLI shall create the scan object with `use_multipart_upload: true`.
- **REQ-01.2** When the create-object response contains `multipart_upload`, the
  CLI shall split the file into `ceil(size / part_size_bytes)` parts using the
  server-provided `part_size_bytes`, upload each part to a presigned part URL,
  and call `multipart/complete` with every part number and its `ETag` in
  ascending part order.
- **REQ-01.3** If multipart was requested and the create-object response has
  no `multipart_upload`, the CLI shall fail with "scan object response did not
  include multipart upload details" and shall not fall back to a single PUT
  (SDK 2.2.0 parity).
- **REQ-01.4** If the file is empty, the CLI shall fail with "file must not be
  empty" before registering a scan or making any API call, on both upload
  paths (SDK 2.2.0 parity).
- **REQ-01.5** If the computed part count is less than 1 or greater than
  10,000, the CLI shall abort the multipart upload and fail.

### REQ-02 — Concurrency and URL batching

- **REQ-02.1** The CLI shall upload up to `--upload-concurrency` parts in
  parallel (default **10**, range **1–32**).
- **REQ-02.2** The CLI shall request part URLs in batches of at most **32**
  part numbers (`multipart/part-urls`) and keep at most two batches (64 parts)
  outstanding (URL fetched, part not finished). When at most one batch is
  outstanding, the CLI shall fetch the next batch, so workers can cross batch
  boundaries without waiting for the slowest part (SDK 2.2.0 parity).
- **REQ-02.3** If a part-URL response does not return exactly the requested
  part numbers, the CLI shall fail the upload.
- **REQ-02.4** The upload HTTP client shall keep up to 32 idle connections per
  host, so parallel parts reuse connections instead of opening a new TLS
  connection per part (parity with the SDK's per-worker sessions).

### REQ-03 — Retries and failure handling

- **REQ-03.1** When a part upload fails with a transport error, HTTP 429,
  HTTP 5xx, or HTTP 400/403 whose body contains `ExpiredToken` or
  `RequestExpired`, the CLI shall request a fresh URL for that part and retry,
  up to **3** attempts in total, with exponential backoff starting at 0.5 s.
- **REQ-03.1a** When retries request fresh part URLs, the CLI shall send at
  most one single-part URL refresh at a time (SDK 2.2.0 refresh lock), so a
  batch of expired URLs does not send a burst of `multipart/part-urls` calls.
- **REQ-03.2** If a part upload fails with any other error, or exhausts its
  attempts, the CLI shall stop scheduling new parts, call `multipart/abort`
  (best effort), and return "multipart upload part N failed" (SDK 2.2.0
  parity). The error shall be an `api.UploadError` that records the part
  number, whether the last failure was transient (REQ-03.1 conditions), and
  the object-store cause.
- **REQ-03.2a** If `multipart/abort` fails, the CLI shall ignore that failure
  without writing any output (as `CleanupScanData` does) and return the
  original upload error.
- **REQ-03.3** If a part upload response has no `ETag` header or an `ETag`
  longer than 2048 characters, the CLI shall treat the part as failed
  (non-retryable).
- **REQ-03.4** If the upload fails, the CLI shall delete the scan
  (existing `CleanupScanData` behaviour). No new exit codes are added.
- **REQ-03.4a** If an upload to the object store fails (a multipart part, or
  the whole file with `--no-multipart`), the CLI shall classify it as an upload
  failure, not as an AI Defense API failure: exit **4** with a
  connectivity hint if the last failure was transient, otherwise exit **1**
  with a hint that the object store rejected the upload. The message shall
  keep "multipart upload part N failed" (or "file upload failed") followed by
  the sanitized object-store status and S3 code, and shall not show the
  API-key hint.
- **REQ-03.4b** If a call to the AI Defense API itself fails during the
  upload (create object, part URLs, complete), the CLI shall keep the existing
  API error classification.
- **REQ-03.5** If a part URL is not HTTPS or its host does not end in
  `.amazonaws.com` or `.amazonaws.com.cn`, the CLI shall fail the part
  (non-retryable) without sending it. Part PUTs shall not follow redirects; a
  3xx response fails the part (SDK 2.2.0 parity). Part URLs also pass the
  existing single-PUT SSRF checks (no private, loopback, or link-local hosts).
- **REQ-03.6** If a presigned upload request fails, the CLI shall not include
  the presigned URL query string (signature) or the raw object-store error
  document in the error message; it shall show only the S3 error `Code` and
  `Message`. Applies to single and multipart uploads.
- **REQ-03.6a** If an upload error body is not an S3 XML error document (for
  example a proxy or WAF page that reflects the request URI), the CLI shall
  show only "unexpected response from object store (N bytes, not shown)" and
  none of the body.
- **REQ-03.6b** When the CLI shows an S3 `Code` or `Message`, it shall replace
  the values of presigned-URL parameters (`X-Amz-*`, `AWSAccessKeyId`,
  `Signature`, `Credential`, `Security-Token`) with `REDACTED`.

### REQ-04 — Flags

- **REQ-04.1** `model-scan trigger` shall accept `--no-multipart`; where it is
  set, the CLI shall use the single presigned PUT path.
- **REQ-04.2** `model-scan trigger` shall accept `--upload-concurrency <n>`.
  Where `--file-path` is set without `--no-multipart`, if `n` is outside 1–32,
  the CLI shall exit **2** before making any API call.
- **REQ-04.3** `--no-multipart` and `--upload-concurrency` shall have no effect
  on `--repo-url` scans, and `--upload-concurrency` shall have no effect with
  `--no-multipart`: in those modes the CLI shall not validate or use its value
  (SDK 2.2.0 validates concurrency only on the multipart path).

### REQ-05 — Upload timeouts

- **REQ-05.1** The CLI shall apply an upload timeout of at least **300 s** per
  upload request (each multipart part, or the whole single PUT), independent of
  the 30 s API request timeout, unless `api.Config.UploadTimeout` is set
  explicitly.

### REQ-06 — Console output

`model-scan trigger` keeps its existing lines and adds the
`cisco-aidefense-sdk` 2.2.0 upload bar (`_ConsoleUploadProgress`) and polling
spinner (`_ConsoleStatusSpinner`).

- **REQ-06.1** When `model-scan trigger --file-path` starts, the CLI shall
  write `Scanning file (uploading and processing)...` to stderr.
- **REQ-06.2** While the file uploads, the CLI shall redraw one line on
  stderr, at most every 100 ms:
  `\rUploading [<bar>] <pct> (<uploaded>/<total> MiB)`, where `<bar>` is 30
  characters of `#` (done) and `-` (remaining), `<pct>` is formatted as
  `%6.1f%%`, and sizes are MiB with one decimal. When the upload completes,
  the CLI shall end the line with a newline; if the upload fails, the CLI
  shall end the unfinished line before printing the error.
- **REQ-06.3** The CLI shall count upload progress as part bodies are sent,
  not only when parts complete. If a part attempt fails, the CLI shall
  subtract that attempt's bytes so progress never exceeds the file size.
- **REQ-06.4** When `model-scan trigger --repo-url` starts, the CLI shall
  write `Scanning repository (processing)...` to stderr.
- **REQ-06.5** While waiting for file scan results, the CLI shall redraw
  `\r<frame> Upload complete. Waiting for scan status...` on stderr every
  100 ms, cycling `|`, `/`, `-`, `\`, starting once the scan is triggered,
  and shall clear it with `\r`, 64 spaces, and `\r` before printing the
  result or an error. While waiting for repository scan results, the CLI
  shall show the same spinner with the text `Waiting for scan status...`,
  because nothing is uploaded (the SDK shows the file text for both).
- **REQ-06.6** The CLI shall not write other progress or lifecycle lines (no
  part retries, abort failures, scan IDs, or status changes), and console
  output shall never contain presigned URLs.
- **REQ-06.7** The CLI shall write only the raw API JSON / verdict envelope
  to stdout, as before.

### REQ-07 — Quality gates

- **REQ-07.1** New Go meets the repo diff-coverage gate (75% of changed lines).
- **REQ-07.2** `make build-go`, `make vet`, `make test`, and
  `make check-exit-docs` pass.

### REQ-08 — Default poll budget of 15,000 s (SDK parity)

Multipart makes multi-hundred-GiB uploads succeed, but the scanner Job then
downloads the whole object from S3 before scanning it. A 456 GiB file took
about 60 minutes from trigger to result on a SAM, against a fixed 600 s poll
budget. scan-manager allows a model scan up to 4 h after it is triggered.
`cisco-aidefense-sdk` ([PR #127](https://github.com/cisco-ai-defense/ai-defense-python-sdk/pull/127),
AIFW-35569) waits 15,000 s by default for both file and repository scans;
the CLI matches it.

- **REQ-08.1** Where neither `AI_DEFENSE_RETRY_COUNT_FOR_SCANNING` nor
  `AI_DEFENSE_WAIT_TIME_SECS_SUCCESSIVE_SCAN_INFO_CHECK` is set, when
  `model-scan trigger` (file or repository) polls for the result, the CLI
  shall use a poll budget of **15,000 s**, polling every 10 s (1,500 polls).
- **REQ-08.2** Where either variable is set, the CLI shall poll every
  `AI_DEFENSE_WAIT_TIME_SECS_SUCCESSIVE_SCAN_INFO_CHECK` seconds (default 10)
  up to `AI_DEFENSE_RETRY_COUNT_FOR_SCANNING` times (default 60), so the
  budget is their product.

### REQ-09 — Poll timeout guidance

- **REQ-09.1** If a model scan does not reach a terminal status within the
  poll budget, the CLI shall exit **4** and state that the scan is still
  running on the server and was not cancelled.
- **REQ-09.2** If a model scan does not reach a terminal status within the
  poll budget, the CLI shall print `cisco-aid-cli model-scan info --scan-id
  <id>` and the environment variables that raise the model-scan poll budget.
- **REQ-09.5** `mcp-scan` poll timeouts are out of scope and keep their
  existing hint.
- **REQ-09.3** The poll-timeout hint shall not instruct the user to re-run the
  scan command, because a re-run registers a new scan and re-uploads the file.
- **REQ-09.4** If polling times out, the CLI shall not delete the scan.
