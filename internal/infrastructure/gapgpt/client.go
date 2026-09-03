// Package gapgpt calls GapGPT (the OpenAI-compatible proxy already used by
// findra/backend for AI research) to classify why a customer stopped
// buying, from their own chat/ticket messages. Unlike findra's usage, this
// client never enables the hosted web_search tool — classifying text the
// caller already provides needs no web research, so leaving tools off
// keeps each call fast (a few seconds, not 10-30s) and cheap.
package gapgpt

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"centropy-affilate/internal/domain/analysis"
	apperrors "centropy-affilate/pkg/errors"
)

const maxResponseBytes = 1024 * 1024

const systemPrompt = `You are analyzing a fitness-coaching app customer's OWN chat/support messages
(never the coach's or support staff's replies) to determine why this customer — who
bought at least one training/diet program before — has not purchased again in a while.

Classify into EXACTLY ONE of these categories:
- PROGRAM_DELAY: complained their training/diet program was late, delayed, or never arrived
- PRICE: expressed concern about cost, affordability, or asked for a discount/installments
- SUPPORT_QUALITY: complained support or their coach was slow, unresponsive, or unhelpful
- TECHNICAL_ISSUE: reported app bugs, upload/playback errors, or payment gateway failures
- HEALTH_PERSONAL: mentioned an injury, illness, or personal-life reason unrelated to the product
- LOST_INTEREST: no complaint of any kind — messages read as simply having gone quiet
- UNCLEAR: messages exist but give no clear signal either way

Write "summary" as ONE short sentence in Persian explaining your reasoning, grounded only
in what the messages actually say — never invent a complaint that isn't there.

Return ONLY valid JSON, no prose before or after, matching this shape:
{"category": "PROGRAM_DELAY" | "PRICE" | "SUPPORT_QUALITY" | "TECHNICAL_ISSUE" | "HEALTH_PERSONAL" | "LOST_INTEREST" | "UNCLEAR", "summary": "string", "confidence": "high" | "medium" | "low"}`

type Client struct {
	httpClient *http.Client
	baseURL    string
	apiKey     string
	model      string
}

func NewClient(baseURL, apiKey, model string) *Client {
	return &Client{
		httpClient: &http.Client{Timeout: 60 * time.Second},
		baseURL:    strings.TrimRight(baseURL, "/"),
		apiKey:     apiKey,
		model:      model,
	}
}

type responsesRequest struct {
	Model string    `json:"model"`
	Input []message `json:"input"`
}

type message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type responsesEnvelope struct {
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
	Output []outputItem `json:"output"`
}

type outputItem struct {
	Type    string `json:"type"`
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
}

type classifyAnswer struct {
	Category   string `json:"category"`
	Summary    string `json:"summary"`
	Confidence string `json:"confidence"`
}

var validCategories = map[string]analysis.Category{
	"PROGRAM_DELAY":   analysis.ProgramDelay,
	"PRICE":           analysis.Price,
	"SUPPORT_QUALITY": analysis.SupportQuality,
	"TECHNICAL_ISSUE": analysis.TechnicalIssue,
	"HEALTH_PERSONAL": analysis.HealthPersonal,
	"LOST_INTEREST":   analysis.LostInterest,
	"UNCLEAR":         analysis.Unclear,
}

// Classify sends the customer's own messages (chronological) to GapGPT and
// parses its verdict. Input is capped at ~6000 characters (oldest messages
// dropped first) to bound token cost — a customer's actual message history
// so far is nowhere near this, so the cap is a defensive ceiling, not a
// normal-case truncation.
func (c *Client) Classify(ctx context.Context, messages []string) (analysis.Category, string, string, error) {
	text, err := c.call(ctx, systemPrompt, buildUserContent(messages))
	if err != nil {
		return "", "", "", err
	}

	var answer classifyAnswer
	if err := json.Unmarshal([]byte(stripCodeFence(text)), &answer); err != nil {
		return "", "", "", apperrors.Wrap(apperrors.KindUnknown, "gapgpt: model answer was not valid JSON", err)
	}

	category, ok := validCategories[strings.ToUpper(strings.TrimSpace(answer.Category))]
	if !ok {
		category = analysis.Unclear
	}
	confidence := strings.ToLower(strings.TrimSpace(answer.Confidence))
	if confidence != "high" && confidence != "medium" && confidence != "low" {
		confidence = "low"
	}

	return category, strings.TrimSpace(answer.Summary), confidence, nil
}

// call posts a system+user message pair to GapGPT's Responses API (no
// tools — every use in this client classifies text already provided, never
// needs web research) and returns the model's raw answer text.
func (c *Client) call(ctx context.Context, systemPrompt, userContent string) (string, error) {
	if c.apiKey == "" {
		return "", apperrors.Wrap(apperrors.KindUnknown, "gapgpt: not configured", fmt.Errorf("GAPGPT_API_KEY is empty"))
	}

	reqBody, err := json.Marshal(responsesRequest{
		Model: c.model,
		Input: []message{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userContent},
		},
	})
	if err != nil {
		return "", apperrors.Wrap(apperrors.KindUnknown, "gapgpt: encode request", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/responses", bytes.NewReader(reqBody))
	if err != nil {
		return "", apperrors.Wrap(apperrors.KindUnknown, "gapgpt: build request", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", apperrors.Wrap(apperrors.KindUnknown, "gapgpt: request failed", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
	if err != nil {
		return "", apperrors.Wrap(apperrors.KindUnknown, "gapgpt: read response", err)
	}
	if resp.StatusCode != http.StatusOK {
		return "", apperrors.Wrap(apperrors.KindUnknown, "gapgpt: unexpected status "+strconv.Itoa(resp.StatusCode), fmt.Errorf("%s", truncate(respBody, 500)))
	}

	var envelope responsesEnvelope
	if err := json.Unmarshal(respBody, &envelope); err != nil {
		return "", apperrors.Wrap(apperrors.KindUnknown, "gapgpt: decode response", err)
	}
	if envelope.Error != nil {
		return "", apperrors.Wrap(apperrors.KindUnknown, "gapgpt: provider error", fmt.Errorf("%s", envelope.Error.Message))
	}

	text := extractAnswerText(envelope.Output)
	if text == "" {
		return "", apperrors.Wrap(apperrors.KindUnknown, "gapgpt: no answer produced", nil)
	}
	return text, nil
}

const verifyDelaySystemPrompt = `You will be given ONE short excerpt from a fitness-coaching app customer's own
chat/support message. It was matched by a keyword search for words like "دیر" (late) and
"برنامه" (program) — but keyword matching catches word co-occurrence, not meaning, so many
matches are false positives (e.g. a message about coffee/cortisol timing that happens to
contain both words unrelated to each other).

Decide: does this excerpt GENUINELY complain that their training/diet program itself was
late, delayed, or never arrived? Answer strictly from the excerpt's actual content.

Return ONLY valid JSON, no prose before or after:
{"is_genuine": boolean, "reasoning": "one short Persian sentence explaining the verdict"}`

type verifyAnswer struct {
	IsGenuine bool   `json:"is_genuine"`
	Reasoning string `json:"reasoning"`
}

// VerifyDelayComplaint judges one keyword-matched excerpt from
// complaint.ListDelayedProgramComplainers — see internal/domain/complaint's
// Verifier doc.
func (c *Client) VerifyDelayComplaint(ctx context.Context, excerpt string) (bool, string, error) {
	text, err := c.call(ctx, verifyDelaySystemPrompt, excerpt)
	if err != nil {
		return false, "", err
	}

	var answer verifyAnswer
	if err := json.Unmarshal([]byte(stripCodeFence(text)), &answer); err != nil {
		return false, "", apperrors.Wrap(apperrors.KindUnknown, "gapgpt: model answer was not valid JSON", err)
	}
	return answer.IsGenuine, strings.TrimSpace(answer.Reasoning), nil
}

const maxInputChars = 6000

// buildUserContent joins messages newest-relevant-first is unnecessary
// here — the prompt asks for chronological order, so oldest-dropped-first
// truncation (keep the most recent messages, which are most relevant to
// "why haven't they bought AGAIN recently") is what actually matters.
func buildUserContent(messages []string) string {
	joined := strings.Join(messages, "\n---\n")
	if len(joined) <= maxInputChars {
		return joined
	}
	return "…\n" + joined[len(joined)-maxInputChars:]
}

func extractAnswerText(items []outputItem) string {
	for _, item := range items {
		if item.Type != "message" {
			continue
		}
		for _, c := range item.Content {
			if c.Type == "output_text" && c.Text != "" {
				return c.Text
			}
		}
	}
	return ""
}

func stripCodeFence(s string) string {
	s = strings.TrimSpace(s)
	if !strings.HasPrefix(s, "```") {
		return s
	}
	s = strings.TrimPrefix(s, "```json")
	s = strings.TrimPrefix(s, "```")
	s = strings.TrimSuffix(s, "```")
	return strings.TrimSpace(s)
}

func truncate(b []byte, n int) string {
	if len(b) <= n {
		return string(b)
	}
	return string(b[:n]) + "…"
}
