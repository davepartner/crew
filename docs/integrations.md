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
inside the 24-hour customer service window. For outside the window, add an
approved template name and language in the integration, then tick **Use an
approved template** when sending. The message text becomes the template's body
parameter.

## Ad accounts

- **Meta Ads** — ad account ID + access token. Lists campaigns, creates paused
  campaigns, pauses/enables campaigns and changes daily budgets.
- **TikTok Ads** — advertiser ID + access token. Lists campaigns, creates paused
  campaigns, pauses/enables campaigns and changes budgets.
- **Google Ads** — developer token + OAuth client + refresh token. Lists
  campaigns, creates paused campaigns and pauses/enables them. Budget changes
  stay in Google Ads.

Marketing → Ads Studio → **Live ad accounts** is the control surface. Enabling a
campaign asks for confirmation because it can spend immediately. Tokens are
encrypted at rest and never returned to the browser.

New campaigns are created **paused**. For Meta, **Delivery** builds a paused ad
set, creative and ad from the campaign, headline, primary text, landing URL,
image URL and country. TikTok and Google campaign controls are available; their
ad-group/ad assembly remains in the provider's Ads Manager. All write actions
require a connected key and explicit confirmation before enabling spend.

## Search

- Crew ships with a built-in SearXNG endpoint at `CREW_SEARCH_URL` (default
  `http://127.0.0.1:8888/search`). It gives arbitrary web results without a
  third-party key.
- A **Brave Search** key can be connected as a higher-quality fallback.
- If neither is available, the advisor uses live Google News industry results,
  Wikipedia and the company website.
- A public **ticker** in Settings → Company adds live market data from Yahoo
  Finance.

## OCR.space

Optional fallback for scans and images. Add the key in Settings → Integrations;
Crew runs local Tesseract and OCR.space and keeps the richer result. Get a key
at https://ocr.space/ocrapi.

## ExchangeRate-API

Portfolio converts products into a chosen reporting currency. A key from
https://www.exchangerate-api.com/ gives a supported SLA; without one Crew uses
the provider's public endpoint and still shows the add-key prompt.

## Safety

- Tokens are encrypted with the server secret.
- Tests and reads are explicit; they do not mutate ad accounts.
- Remove a connection with **Disconnect**.
