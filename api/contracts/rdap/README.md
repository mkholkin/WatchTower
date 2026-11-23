# RDAP contract

WatchTower requests `GET {base_url}/domain/{domain}` with `Accept: application/rdap+json`. The real base URL is `https://rdap.verisign.com/com/v1`; the local mock uses `http://rdap-mock:8089/com/v1`. The client uses one request per lookup and a bounded timeout.

The response is a JSON domain object. `objectClassName` is `domain`, `ldhName` identifies the domain, and the top-level `events` array carries `eventAction` and RFC3339 `eventDate` values. The client selects `registration` and `expiration` by action, independently of array order. Missing dates become JSON `null`. Entity events belong to their entities and do not supply domain dates.

`fixtures/example.com.json` is a synthetic representative response, written for repeatable tests. Its dates are fixed test data. Live registration data can change; real E2E validates the date format and domain without comparing fixed dates.

| Upstream behavior | WatchTower HTTP status |
| --- | --- |
| Domain object | 200 |
| 404 | 404 |
| 429, including Retry-After | 503 |
| Malformed JSON or 5xx | 502 |
| Request timeout | 504 |

The mock exposes `missing.com` (404), `busy.com` (429, `Retry-After: 1`), `broken.com` (malformed JSON), `down.com` (503), `slow.com` (two-second delay), and `partial.com` (no expiration event). `GET /__admin/stats` reports request counts by domain and method, excluding admin requests. Unknown domains share `_other`; methods share bounded buckets. Counters allow the runner to prove that load reached the upstream and that authorization failures never did.

Sources: [RFC 9083](https://www.rfc-editor.org/rfc/rfc9083), sections 4.5 and 5.3, defines events and domain objects; [Verisign RDAP Help](https://www.verisign.com/news-insights/registration-data-access-protocol/help/) documents the `.com` base URL and domain lookup path. Reviewed 2026-11-23.
