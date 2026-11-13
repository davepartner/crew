# Integrations

Settings → **Integrations**.

## Email

The server sends plain text through its SMTP relay. Configure it with:

```
CREW_SMTP_ADDR=127.0.0.1:25
CREW_SMTP_FROM=Crew <noreply@paytrackify.com>
# CREW_SMTP_USER=
# CREW_SMTP_PASS=
```

Use **Send test** to check the relay. This is email sending, not an inbox sync.

## WhatsApp

Meta WhatsApp Cloud API:

- phone number ID
- permanent access token
- optional default recipient

Crew sends text messages from your number. Meta only allows free-form text
inside the 24-hour customer service window; outside it, use an approved
template in Meta Business Manager.

## Ad accounts

Read-only connections:

- **Meta Ads** — ad account ID + access token. Reads campaign names, status and
  objective.
- **TikTok Ads** — advertiser ID + access token. Reads campaigns.
- **Google Ads** — developer token + OAuth client + refresh token. Lists the
  customer accounts the token can access.

Crew never changes budgets, bids or campaign status. The tokens are encrypted
at rest and never returned to the browser.

## Search

- **Brave Search** unlocks arbitrary web search for the advisor.
- Without it, the advisor uses live Google News industry results, Wikipedia and
  the company website.
- A public **ticker** in Settings → Company adds live market data from Yahoo
  Finance.

## Safety

- Tokens are encrypted with the server secret.
- Tests and reads are explicit; they do not mutate ad accounts.
- Remove a connection with **Disconnect**.
