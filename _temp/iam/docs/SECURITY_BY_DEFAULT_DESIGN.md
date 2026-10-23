# Security by default — design record

> Closed wave (2026-10-21), archived from `webtyp/docs/SECURITY_BY_DEFAULT_MASTER_PLAN.md`. iam
> was the trigger and the main consumer; this is the permanent record of the audit and decisions.

Trigger: the API explorer in `veltylabs/iam` reported 22 of 25 routes as
`public`. The framework's own rule (skill **api-design**, `webtyp/router`
`route.go`) is that an undeclared route is `AccessGuarded` and refuses to
start. iam was not insecure because the framework was permissive. It was at
risk because **it had to leave the framework's model to express what it
needed**, and every exit was hand-rolled protection that nothing checked.

Target: OWASP ASVS 4.0 **Level 3**, the level ASVS assigns to "applications
that perform high value transactions" such as banking: deny-by-default
enforced by the framework, phishing-resistant MFA for privileged access,
revocable sessions with idle and absolute limits, machine clients
authenticated as principals, and an inventory of every unauthenticated
surface.

## Audit findings (2026-10-12, verified against the running dev server and source)

| # | Sev. | Finding | Industry rule broken |
|---|---|---|---|
| F1 | Critical | A compromised or malicious `*.velty.cl` app could take over the iam admin panel (cookie `Domain=.velty.cl` replayable server-side) | ASVS V3.4.4 (host-only admin cookie) |
| F2 | High | Authorization lived outside the framework: 22 routes `.Public()` + hand-rolled wrappers | ASVS V4.1.3/V1.4.4 (single access-control mechanism); OWASP API5:2023 |
| F3 | High | Sessions could not be revoked (stateless JWT, 7-day TTL, no idle timeout) | ASVS V3.3.1/V3.3.2; NIST 800-63B §7.2; PCI DSS 4.0 req. 8.2.8 |
| F4 | High | `DEV_AUTOLOGIN` compiled into production (no build passed `-tags prod`) | ASVS V14.1 |
| F5 | High | Admin access single-factor (Google login + email allow-list, no MFA) | PCI DSS 4.0 req. 8.4.2; NIST 800-63B AAL3 |
| F6 | Medium | No machine principal: `client_secret` in the JSON body, checked per handler | RFC 6749 §2.3.1; FAPI 2.0 |
| F7 | Medium | `/_routes` public in production | OWASP API9:2023; ASVS V14.3 |
| F8 | Medium | No throttling anywhere | OWASP API4:2023; ASVS V2.2.1 |
| F9 | Medium | Machine mutations (`AssignRole`, `ResolveUser`) not audited | PCI DSS 4.0 req. 10.2.1 |
| F10 | Low | `/api/health/db` public, reveals DB reachability | Kubernetes probe convention |
| F11 | Low | `POST /logout` with no CSRF/origin check | ASVS V4.2.2 |
| F12 | To verify | Audit log tamper-evidence not verified | PCI DSS 4.0 req. 10.3.2/10.3.3 |

## Decisions (closed 2026-10-13)

| # | Decision | Chosen |
|---|---|---|
| D1 | Machine client auth | `private_key_jwt` only (RFC 7523), ES256, no interim `client_secret` step |
| D2 | MFA for panel admins | Passkey mandatory for every admin session |
| D3 | Rate limiting | Cloudflare rate-limit rule applied by `goflare` on every deploy; no per-client lockout (DoS risk with asymmetric clients) |
| D4 | Panel admin model | `iam_admin` RBAC role in iam's own project, seeded from `IAM_ADMIN_EMAILS` |
| D5 | Public route inventory | `routertest.AssertPublicRoutes(t, r, ...)`, enforced in a reviewed test |

**MFA is a framework fact, not an app convention.** `model.Assurance` (`None`/`SingleFactor`/
`MultiFactor`, NIST SP 800-63B AAL) set by auth middleware; a route declares `.RequiresMFA()`; a
single-factor caller gets 401 + `WWW-Authenticate: Bearer error="insufficient_user_authentication"`
(RFC 9470).

**Two sessions, two scopes.** `.velty.cl` SSO session stays single-factor; the admin plane uses a
host-only `__Host-` session issued after a passkey assertion, 15 min idle / 8 h absolute. Both are
opaque, server-side, revocable (`auth/session/cookie`); `auth/session/jwt` was deleted.

**Passkey enrollment cannot be hijacked.** First passkey registrable only from a session younger
than 5 minutes; every later registration requires `MultiFactor`.

**Machines are principals.** Client-assertion authenticator sets `ctx.UserID()` to the project id
with assurance `SingleFactor`; `/api/token` is a token exchange (RFC 8693).

## What shipped

`model` v0.2.3, `crypto` v0.0.28, `app` v0.6.152, `devflow` v0.4.132, `router` v0.4.3, `server`
v0.2.86, `cloudflare` v0.0.39, `mcp` v0.3.4, `rpc`/`dbsync` in-session, `layout` v0.3.48, `jwt`
v0.1.24, `webauthn` v0.1.6, `auth` v0.0.75, `goflare` v0.5.42, `iam` v0.0.28, `misitio` v0.0.11
(§9: `IAM_CLIENT_PRIVATE_KEY`, one `Consumer`, public inventory test).

## Acceptance (verified end-to-end)

Every `/api/admin/*` route without a cookie → 403; with only the SSO cookie → 401 step-up; replayed
cookie with forged `Origin` → 401 (host-only cookie never reached the sibling); idle admin session
> 15 min → 401; client assertion replay (same `jti`) → 403; burst > 100 req/10s → blocked by
Cloudflare. `routertest.AssertPublicRoutes` green in iam and misitio.

## Out of scope (left for a future wave)

Asymmetric tokens for consumers (RFC 9068 + JWKS); a CSRF/Origin middleware in `router/security`
(closed by the host-only cookie instead); attestation verification (`none` stays the default);
formal pentest, WAF/bot management, secrets-rotation tooling.
