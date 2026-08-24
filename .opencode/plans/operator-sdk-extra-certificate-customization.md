# operator-sdk-extra: certificate-property customization — DELIVERY STATUS (largely fulfilled)

**Target repository:** `/projects/operator-sdk-extra` (tag **v3.0.6** = commit `8f135ed050ae390f6d02ef3ad7b8aebcb9285756`, "fix: fix tls saga" — one commit ahead of v3.0.4).

> **Version note (verified):** **v3.0.6** generalizes the certificate package "a little bit more generic" and delivers most of what this plan originally proposed. `go.mod`: `module github.com/disaster37/operator-sdk-extra/v3`, `go 1.26`, controller-runtime v0.19.3 / k8s.io v0.32.0 (unchanged from v3.0.4). The `v3.0.3` tag remains a **mis-tag** (`refs/tags/v3.0.3` → `8c9ead49…`, module `/v2` / go 1.24) and must not be used.

**Original goal (now met):** operator developers can customize Subject attributes, SANs, key algorithm/size, and CA/leaf validity/renewal purely via `certificate.TLSSpec` — without reimplementing generation. v3.0.6 delivers this generically (no Elasticsearch-specific naming).

---

## 1. What was delivered by v3.0.6 (proposal → actual)

| Original proposal | Delivered as | Status |
|---|---|---|
| `OrganizationalUnits []string` (OU) | `TLSSpec.Subject.OrganizationalUnits []string` (`CertificateSubject`) | ✅ delivered |
| `Country/Locality/Province []string` | `Subject.Countries / Localities / Provinces []string` (+ `StreetAddresses`, `PostalCodes`, `SerialNumber`) | ✅ delivered (more generic) |
| `KeyAlgorithm` (ECDSA/RSA) | `TLSSpec.KeyAlgorithm` + consts `KeyAlgorithmECDSA`/`KeyAlgorithmRSA` | ✅ delivered |
| `KeySize` | `TLSSpec.KeySize` + `DefaultECDSAKeySize`/`DefaultRSAKeySize`/`MaxRSAKeySize`; `EffectiveKeySize()` | ✅ delivered |
| ECDSA curve/complexity selection | `TLSSpec.Curve` (`CurveP256`/`CurveP384`/`CurveP521`) + `curveFor`/`EffectiveKeySize` | ✅ already exposed — no new gap |
| CA validity override | `TLSSpec.CAValidityDays` (already existed) + `CACommonName`/`CASubject` (CA CN/RDN override) | ✅ (bonus) |
| — | `Usages []string` / `KeyUsages []string` (ExtKeyUsage/KeyUsage) | ✅ (bonus) |
| — | `certificate.CertificateCustomizer[T]` + `rotation.WithCertificateCustomizer` (generic cluster-state SAN/IP/CN computation) | ✅ (bonus — this is the generic version of the ES operator's auto-computation) |
| — | `ValidateContent()`, `ContentIgnoringBackend`, `CAContentChanged`, `KeyChanged`/`UsagesChanged`, new `LeafChangeReason` (`SubjectChanged`/`KeyChanged`/`UsagesChanged`) | ✅ (bonus) |
| `CARenewalDays` (CA-specific renewal window) | **NOT delivered** — `RenewalDays` is still the shared CA+leaf window; `CAValidityDays` is CA *validity*; `CANeedsRenewal` still uses `GetValidRenewalDays` | ❌ **remaining** |

## 2. Remaining gap — `CARenewalDays` (sole residual item)

The only unfulfilled proposal is a **CA-specific renewal window**, distinct from the leaf renewal window. Today the library has one shared `RenewalDays` used by both `CANeedsRenewal` (CA) and `LeafNeedsChange`/`LeafNeedsChange` (leaf), plus `CAValidityDays` (CA lifetime, not a renewal trigger). An operator that wants to rotate the CA on a different schedule than leaf certs (e.g., renew leaf at 30 days but rotate CA at 90 days before expiry) cannot express that today.

### 2.1 Proposed change (additive, optional, behavior-preserving)

`pkg/controller/certificate/backend.go`:
```go
// TLSSpec addition:
CARenewalDays int `json:"caRenewalDays,omitempty"` // CA-specific renewal window in days

const MaxCARenewalDays = MaxRenewalDays // reuse 36500 clamp

// GetValidCARenewalDays returns the CA renewal window, defaulting to
// GetValidRenewalDays (30) when CARenewalDays <= 0, clamped to MaxCARenewalDays.
func GetValidCARenewalDays(spec TLSSpec) int
```
`pkg/controller/certificate/selfmanaged/backend.go`: change `CANeedsRenewal` to use `GetValidCARenewalDays(spec)` instead of `GetValidRenewalDays(spec)` (leaf expiry in `LeafNeedsChange` keeps `GetValidRenewalDays`). This is backward compatible (default = the old shared 30-day window).

### 2.2 Tests / docs

- `selfmanaged/backend_test.go`: assert `CANeedsRenewal` uses the CA window (age CA inside CA window but outside leaf window → CA renews; and vice-versa). Assert `GetValidCARenewalDays` default == `GetValidRenewalDays` and clamps at `MaxCARenewalDays`.
- `documentations/tls-and-workflow.md`: document `CARenewalDays` and the CA-vs-leaf renewal distinction.

### 2.3 Verification (in `/projects/operator-sdk-extra`)

```
go build ./pkg/controller/certificate/...
go test ./pkg/controller/certificate/...
make test   # full suite incl. envtest
```

---

## 3. Notes for the consumer (elasticsearch-operator)

- The operator's `caRenewalDays` spec field maps to `TLSSpec.CARenewalDays` once this residual lands; until then map it to `TLSSpec.RenewalDays` (shared window) — see main migration plan §4.4.4/R3.
- The operator's new `KeyComplexity` spec field maps to `TLSSpec.KeyAlgorithm` + (`Curve` for `ecdsa-*` | `KeySize` for `rsa-*`); legacy `KeySize` maps to `KeySize` + `KeyAlgorithm=RSA`. `AltIps`/`AltNames` map to `IPAddresses`/`DNSNames`; O/OU/Country/Locality/Province map to `Subject.Organizations/OrganizationalUnits/Countries/Localities/Provinces`; CN/CA-CN map to `CommonName`/`CACommonName`; `ValidityDays`/`RenewalDays` map to `LeafValidityDays`/`RenewalDays`.
- Pin **v3.0.7** via `go get …@v3.0.7` (the operator's current pin; cert API == v3.0.6, commit `8f135ed0`). Do **not** use `v3.0.3` (mis-tag of the v2 commit `8c9ead49`, module `/v2`).
