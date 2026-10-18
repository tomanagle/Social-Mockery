# Provider contract provenance

Copy this template next to the versioned contract and replace the prompts with evidence. Unavailable evidence should be recorded as unknown, not guessed. Delete instructional prompts after completing the record.

## Contract identity

- Provider / API product / login flow:
- Upstream API version:
- Contract path and OpenAPI version:
- Contract revision (repository commit or release, when known):
- Research date (YYYY-MM-DD, UTC):
- Intended client/use case:
- Scope and explicit exclusions:
- Generator module, exact version, config path, and commands:

## Sources

| Source ID | Exact official URL and section | Kind (docs / Postman / SDK) | Provider/API version | Revision, release, or export checksum | Retrieved date | Claims supported / limitations |
| --- | --- | --- | --- | --- | --- | --- |

For mutable documentation without a revision, say so. A retrieval date is not an API version. Record only permitted excerpts or references; do not vendor entire documentation collections without checking reuse terms.

## Endpoint coverage

| Method and upstream path | Operation ID | Implemented request/response branches | Classification | Source IDs | Tests / fixtures | Unsupported or unknown branches |
| --- | --- | --- | --- | --- | --- | --- |

Use documentation-based, live-verified, or synthetic for individual claims. Split rows when different branches have different evidence. Include requested but deferred operations so partial coverage is visible.

## Material semantic claims

| Operation / field / behavior | Claim (requiredness, encoding, status, lifecycle, etc.) | Evidence classification | Source ID and section or sanitized observation reference | Simulator deviation / unresolved question |
| --- | --- | --- | --- | --- |

Record success and error semantics separately. Include pagination, token placement, processing transitions, limits, and version differences only where applicable. Unknown provider behavior must not become an undocumented simulator guarantee.

## Simulator choices

Describe synthetic IDs, tokens, virtual timing, error injection, local URL rewriting, deterministic ordering, and other intentional deviations that apply to this slice. Synthetic fixture values can exercise documentation-based schemas without making their contents provider-verified.

## Live observations (omit if none)

- Authorized scope and date; no credentials or identifying account details:
- Observed upstream API version and date:
- Operation and sanitized input category:
- Observed behavior and limitations of the observation:
- Sanitized fixture reference and redaction review:

If none were performed, explicitly state that the contract was not live-verified.

## Verification record

| Check | Exact command and tool version | Result (pass / fail / blocked / not run) | Date and relevant output |
| --- | --- | --- | --- |

Include OpenAPI validation, deterministic regeneration check, Go compile/tests, interface assertion, and HTTP contract tests. End with remaining work and the next bounded endpoint slice.
