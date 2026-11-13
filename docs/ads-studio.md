# Ads Studio

Marketing → **Ads Studio** turns one brief into platform-native campaigns.

## What it produces

For each selected platform (Meta, TikTok, YouTube):

- 3 materially distinct concepts, not three rewrites of one idea
- a hook, angle and visual direction
- platform copy: Meta primary text/headline/description/CTA; TikTok timed script,
  caption, hashtags, overlays and sound; YouTube timed script, voiceover,
  on-screen text, end card and thumbnail direction
- 4–6 hooks per platform with the category and why it fits

Across the campaign:

- strategy: audience, offer, positioning, proof and unknowns
- personas, angles, a 5-row test plan, a budget split
- compliance: claims to verify, things to avoid, spec checks
- scores out of 10 for hook, clarity, platform fit, distinctiveness, proof, CTA,
  compliance and overall

## Grounding rules

- Only supplied product facts, proof and offer are used. Missing proof becomes a
  claim to verify, not an invented testimonial or statistic.
- Exact ratios, durations and character limits change by placement. The prompt
  gives a sensible default and tells the operator to verify in Ads Manager.
- Scores are self-assessments, not predicted performance.
- No live ad account is changed. The output is copy and a production brief.

## Sources

The playbooks are distilled from three MIT-licensed projects:

- [Claude Ads](https://github.com/AgriciDaniel/claude-ads) — platform creative
  contracts, copy frameworks, non-fabrication rules
- [AI Ads Strategist](https://github.com/zubair-trabzada/ai-ads-claude) — hook
  categories, brief structure, test plans
- [Creatify video ad skills](https://github.com/creatify-ai/video-ad-generator) —
  video brief, script structures, creative evaluation rubric

See `THIRD_PARTY_NOTICES.md`. None of these projects endorse Crew.

## Using the output

- **Copy concept** copies one concept as plain text.
- **Copy** on a field copies that field.
- **Export CSV** produces one row per concept, ready for a spreadsheet.
- **Send to helper** creates a task for the marketing/editorial helper with the
  full campaign as the brief.
