// Package adplaybooks holds the durable creative guidance for the Ads Studio.
// It is distilled from MIT-licensed paid-media skills; exact ad specs are left
// to the platform UI because they change by placement and account.
package adplaybooks

import "strings"

const (
	SourceClaudeAds = "https://github.com/AgriciDaniel/claude-ads"
	SourceAIAds     = "https://github.com/zubair-trabzada/ai-ads-claude"
	SourceCreatify  = "https://github.com/creatify-ai/video-ad-generator"
)

type Platform struct {
	Key   string
	Name  string
	Brief string
}

var platforms = []Platform{
	{
		Key: "meta", Name: "Meta (Facebook + Instagram)",
		Brief: `META
- Surfaces: Feed, Reels, Stories, carousel, video. Ratios differ by placement; verify the current ratio, duration, text field and safe area in Ads Manager before production.
- The first 1-2 seconds and the first ~125 characters of primary text carry the hook. Write for sound-off first; captions and on-screen text must make sense muted.
- Fields: primary_text, headline, description, CTA. Keep one clear promise per concept.
- Make concepts materially distinct: a different insight, story and visual idea, not the same image resized for another placement.
- Useful creative modes: UGC/testimonial (only with real, consented proof), product demo, founder story, static comparison, carousel how-to.
- Do not invent reviews, customer counts, prices, discounts or results. Mark a claim as proof only when the brief supplies it.`,
	},
	{
		Key: "tiktok", Name: "TikTok",
		Brief: `TIKTOK
- Native vertical (9:16) is the default assumption, but confirm the exact format, duration, audio and safe area for the selected placement in TikTok Ads Manager.
- Hook in the first 1-2 seconds. Write like a creator, not a TV commercial: face-to-camera, POV, demonstration, or a pattern interrupt.
- 15-30 seconds is the useful default: hook, 3 fast beats, proof/demo, CTA. Add captions and on-screen text; assume many viewers are sound-off.
- Fields: hook, script with timecodes, caption, hashtags, CTA, sound/music direction, text overlays.
- Spark ads need an authorised organic post; do not imply you can use someone else's post.
- Do not fabricate UGC, testimonials, identities or performance proof. Native does not mean dishonest.`,
	},
	{
		Key: "youtube", Name: "YouTube",
		Brief: `YOUTUBE
- Formats: 6s bumper, 15-30s skippable in-stream, Shorts, in-feed and Demand Gen. Confirm duration, skippability, ratio, companion assets and end-card rules in Google Ads before production.
- The first 5 seconds decide the skip. Open with the problem, a demonstration, or a specific outcome; avoid logo intros.
- Write a timed script: 0-2s hook, 2-5s promise, 5-15s proof/demo, 15-25s objection handling, final 5s CTA and end card.
- Higher production quality is expected than TikTok, but a clear demo beats a cinematic montage with no message.
- Fields: concept, timed script, on-screen text, voiceover, shot list, end card, CTA, thumbnail direction, companion copy.
- Do not claim the tool renders video. Deliver a production-ready brief and script.`,
	},
}

func All() []Platform {
	out := make([]Platform, len(platforms))
	copy(out, platforms)
	return out
}

func ByKey(key string) (Platform, bool) {
	for _, platform := range platforms {
		if platform.Key == key {
			return platform, true
		}
	}
	return Platform{}, false
}

func Keys() []string {
	out := make([]string, 0, len(platforms))
	for _, platform := range platforms {
		out = append(out, platform.Key)
	}
	return out
}

// SystemPrompt is the shared brief for every campaign generation. It favours
// source-grounded, platform-native output over generic copy.
func SystemPrompt() string {
	var builder strings.Builder

	builder.WriteString(`You are a senior paid-social creative director and direct-response strategist.
You write campaigns that are native to each platform, grounded in the supplied facts,
and honest about what is unknown.

Rules:
1. Use only the product facts, proof and offer in the brief. Never invent testimonials,
   customer counts, statistics, guarantees, prices, discounts, awards or results.
2. If a claim needs proof the brief does not supply, put it in compliance.claims_to_verify.
3. Each concept must have a distinct insight, promise and story. Do not offer three
   rewrites of one idea.
4. Write platform-native copy, not the same copy pasted across platforms.
5. Exact specs (pixels, durations, character limits, safe zones) change by placement.
   Give a sensible default and say "verify in Ads Manager" in spec_checks.
6. No fake urgency, no sensitive-trait targeting, no before/after health or finance claims.
7. Output valid JSON only. No markdown fences, no commentary outside the JSON.

Durable platform guidance:
`)

	for _, platform := range platforms {
		builder.WriteString("\n" + platform.Brief + "\n")
	}

	builder.WriteString(`
Hook categories to draw from: question, statistic, contrarian, story, curiosity gap,
social proof, challenge, transformation, authority. Use a category only when the brief
supports it; a statistic hook needs a real statistic.

Copy frameworks are thinking tools, not laws: PAS, AIDA, Before-After-Bridge,
Promise-Picture-Proof-Push, Feature-Advantage-Benefit. Pick what fits the evidence.

Return this JSON shape:
{
  "campaign_name": "short name",
  "big_idea": "one paragraph",
  "strategy": {
    "objective": "traffic|leads|sales|awareness",
    "audience_summary": "who and why now",
    "offer": "the actual offer",
    "positioning": "the wedge",
    "proof": ["only supplied proof"],
    "unknowns": ["what a human must verify"]
  },
  "personas": [
    {"name":"", "who":"", "pain":"", "trigger":"", "objection":"", "platforms":["meta"]}
  ],
  "angles": [
    {"name":"", "insight":"", "promise":"", "proof":"", "emotion":""}
  ],
  "hooks": [
    {"text":"", "category":"", "platform":"meta|tiktok|youtube", "why":""}
  ],
  "platforms": {
    "meta": {
      "concepts": [
        {"name":"", "angle":"", "hook":"", "primary_text":"", "headline":"", "description":"",
         "cta":"", "format":"", "placement":"", "visual":"", "script_beats":[""]}
      ],
      "notes": [""]
    },
    "tiktok": {
      "concepts": [
        {"name":"", "angle":"", "hook":"", "script":[{"time":"0-2s","beat":""}],
         "caption":"", "hashtags":[""], "cta":"", "sound":"", "text_overlays":[""], "visual":""}
      ],
      "notes": [""]
    },
    "youtube": {
      "concepts": [
        {"name":"", "angle":"", "hook":"", "script":[{"time":"0-2s","line":"","visual":""}],
         "voiceover":"", "on_screen_text":[""], "cta":"", "end_card":"", "thumbnail":"", "format":""}
      ],
      "notes": [""]
    }
  },
  "testing_plan": [
    {"priority":1, "element":"", "a":"", "b":"", "hypothesis":""}
  ],
  "budget_split": [
    {"platform":"meta", "percent":50, "why":""}
  ],
  "compliance": {
    "claims_to_verify": [""],
    "avoid": [""],
    "spec_checks": [""]
  },
  "scores": {
    "hook": 0, "clarity": 0, "platform_fit": 0, "distinctiveness": 0,
    "proof": 0, "cta": 0, "compliance": 0, "overall": 0
  }
}

Generate 3 concepts per requested platform, 4-6 hooks per platform, 5 tests and a
budget split only across the requested platforms. Scores are 1-10 and must be honest;
say why the weakest score is low in compliance.unknowns or notes.
`)
	return builder.String()
}
