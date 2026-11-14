# Cyber Security

Left menu → **Cyber Security**.

The workspace is modelled on OpenCTI's core ideas: company-scoped assets,
observables/indicators, reports, threat actors, relationships, confidence, TLP
markings, first/last seen dates, a graph and STIX 2.1 export.

## Assets

Crew automatically adds:

- the website domain
- the support email and its domain

Add domains, IPs, emails, URLs and brands manually. Every scan is limited to
these assets.

## Free checks

- DNS resolution
- SPF and DMARC records
- TLS certificate issuer and expiry
- HSTS, CSP, X-Frame-Options and Referrer-Policy
- `/.well-known/security.txt`
- URLhaus host lookup
- ThreatFox IOC search

## Connected feeds

Add keys in Cyber Security → Assets & sources:

- VirusTotal — domain reputation and detections
- AbuseIPDB — IP abuse score
- AlienVault OTX — domain pulses
- Shodan — exposed ports and known vulnerabilities

Each card links to **Get a key** and the provider docs. Keys are encrypted and
never returned to the browser.

## Assessment

A scan asks the configured model for:

- threat level: low, guarded, elevated, high or critical
- exposure score
- summary and business impact
- a risk register with a fix for each risk
- next actions

If no model is configured, the deterministic findings still populate the page.

## Graph

Assets, indicators, reports and actors are drawn as a simple relationship graph.
Edges come from stored relationships and indicator values that reference an asset.

## STIX 2.1

**Export STIX** downloads a bundle of indicators, reports, actors and an
identity object. It uses STIX indicator patterns for domain, IP, URL, email and
file hashes.

## Limits

- This is a defensive exposure and intelligence workspace, not a SIEM, EDR or
  pentest.
- A clean scan does not prove the company is safe.
- Threat actor attribution is only shown when supplied; Crew does not invent it.
- Free feeds and provider APIs have their own rate limits.

Inspired by [OpenCTI](https://github.com/OpenCTI-Platform/opencti) (Apache 2.0).
