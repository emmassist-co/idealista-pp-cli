package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"slices"
	"strings"

	"golang.org/x/net/html"
	"idealista-pp-cli/internal/client"
)

type listingSearchCard struct {
	ListingID       string   `json:"listing_id"`
	ListingURL      string   `json:"listing_url,omitempty"`
	Title           string   `json:"title,omitempty"`
	Price           string   `json:"price,omitempty"`
	PriceDetails    string   `json:"price_details,omitempty"`
	Address         string   `json:"address,omitempty"`
	Description     string   `json:"description,omitempty"`
	PrimaryImageURL string   `json:"primary_image_url,omitempty"`
	Features        []string `json:"features,omitempty"`
}

type listingAjaxResultsPayload struct {
	URL           string              `json:"url"`
	Path          string              `json:"path"`
	Query         map[string]string   `json:"query"`
	ValidatedSpec searchResultsSpec   `json:"validated_spec"`
	CardCount     int                 `json:"card_count"`
	Cards         []listingSearchCard `json:"cards,omitempty"`
	RawBody       string              `json:"raw_body,omitempty"`
}

type listingAjaxEnrichedPayload struct {
	URL            string                  `json:"url"`
	Path           string                  `json:"path"`
	Query          map[string]string       `json:"query"`
	ValidatedSpec  searchResultsSpec       `json:"validated_spec"`
	ParsedCount    int                     `json:"parsed_count"`
	ShortlistLimit int                     `json:"shortlist_limit"`
	SelectedIDs    []string                `json:"selected_ids,omitempty"`
	Cards          []listingSearchCard     `json:"cards,omitempty"`
	Listings       []listingInspectSummary `json:"listings,omitempty"`
	PartialErrors  []string                `json:"partial_errors,omitempty"`
}

var listingHrefPattern = regexp.MustCompile(`/imovel/(\d+)/?`)
var listingPricePattern = regexp.MustCompile(`\b[\d\.\s]+€`)
var spacePattern = regexp.MustCompile(`\s+`)
var tagPattern = regexp.MustCompile(`<[^>]+>`)

func websiteListingAjaxHeaders(referer string) map[string]string {
	return map[string]string{
		"Accept":           "application/json, text/javascript, */*; q=0.01",
		"Referer":          referer,
		"Sec-Fetch-Mode":   "cors",
		"Sec-Fetch-Site":   "same-origin",
		"X-Requested-With": "XMLHttpRequest",
	}
}

func fetchListingAjax(ctx context.Context, c *client.Client, path string, query map[string]string, referer string) ([]byte, error) {
	data, err := c.GetWithHeaders(ctx, path, query, websiteListingAjaxHeaders(referer))
	if err != nil {
		return nil, err
	}
	if vendor := looksLikeDoctorInterstitial(data); vendor != "" {
		return nil, authErr(fmt.Errorf("%s interstitial rejected the listing results response", vendor))
	}
	return data, nil
}

func summarizeListingAjaxResults(url, path string, query map[string]string, spec searchResultsSpec, body []byte, includeRaw bool) listingAjaxResultsPayload {
	cards := parseListingCards(body)
	payload := listingAjaxResultsPayload{
		URL:           url,
		Path:          path,
		Query:         query,
		ValidatedSpec: spec,
		CardCount:     len(cards),
		Cards:         cards,
	}
	if includeRaw {
		payload.RawBody = string(body)
	}
	return payload
}

func summarizeListingAjaxTotals(url, path string, query map[string]string, spec searchResultsSpec, body []byte, includeRaw bool) map[string]any {
	payload := map[string]any{
		"url":            url,
		"path":           path,
		"query":          query,
		"validated_spec": spec,
	}
	var parsed any
	if json.Unmarshal(body, &parsed) == nil {
		payload["totals"] = parsed
	} else {
		payload["body"] = string(body)
	}
	if includeRaw {
		payload["raw_body"] = string(body)
	}
	return payload
}

func parseListingCards(body []byte) []listingSearchCard {
	if len(body) == 0 {
		return nil
	}
	if cards := parseListingCardsFromHTML(body); len(cards) > 0 {
		return cards
	}
	return parseListingCardsFallback(body)
}

func parseListingCardsFromHTML(body []byte) []listingSearchCard {
	doc, err := html.Parse(bytes.NewReader(body))
	if err != nil {
		return nil
	}
	var cards []listingSearchCard
	seen := map[string]bool{}
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "a" {
			href := getAttr(n, "href")
			matches := listingHrefPattern.FindStringSubmatch(href)
			if len(matches) == 2 && !seen[matches[1]] {
				cardRoot := nearestCardRoot(n)
				card := buildListingCard(cardRoot, n, matches[1], href)
				if card.ListingID != "" {
					seen[card.ListingID] = true
					cards = append(cards, card)
				}
			}
		}
		for child := n.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
	}
	walk(doc)
	return cards
}

func parseListingCardsFallback(body []byte) []listingSearchCard {
	matches := listingHrefPattern.FindAllSubmatchIndex(body, -1)
	if len(matches) == 0 {
		return nil
	}
	seen := map[string]bool{}
	cards := make([]listingSearchCard, 0, len(matches))
	for i, match := range matches {
		if len(match) < 4 {
			continue
		}
		id := string(body[match[2]:match[3]])
		if seen[id] {
			continue
		}
		seen[id] = true
		start := maxInt(0, match[0]-200)
		end := len(body)
		if i+1 < len(matches) {
			end = minInt(len(body), matches[i+1][0]+200)
		}
		segment := string(body[start:end])
		cards = append(cards, listingSearchCard{
			ListingID:   id,
			ListingURL:  "https://www.idealista.pt/imovel/" + id + "/",
			Title:       normalizeSegmentText(extractAnchorText(segment)),
			Price:       firstRegexMatch(segment, listingPricePattern),
			Description: normalizeSegmentText(segment),
		})
	}
	return cards
}

func buildListingCard(cardRoot, anchor *html.Node, listingID, href string) listingSearchCard {
	card := listingSearchCard{
		ListingID:  listingID,
		ListingURL: absolutizeListingURL(href),
	}
	card.Title = firstNonEmpty(
		normalizeText(getAttr(anchor, "title")),
		nodeText(anchor),
		firstTextByClass(cardRoot, "item-link"),
		firstTextByClass(cardRoot, "item-link-wrap"),
	)
	card.Price = firstNonEmpty(
		firstTextByClass(cardRoot, "item-price"),
		firstRegexMatch(nodeText(cardRoot), listingPricePattern),
	)
	card.PriceDetails = firstNonEmpty(
		firstTextByClass(cardRoot, "item-price-down"),
		firstTextByClass(cardRoot, "item-price-details"),
	)
	card.Address = firstNonEmpty(
		firstTextByClass(cardRoot, "item-detail-location"),
		firstTextByClass(cardRoot, "item-detail-char"),
	)
	card.Description = firstNonEmpty(
		firstTextByClass(cardRoot, "item-description"),
		firstTextByClass(cardRoot, "ellipsis"),
	)
	card.PrimaryImageURL = firstNonEmpty(
		firstImageAttr(cardRoot, "src"),
		firstImageAttr(cardRoot, "data-src"),
		firstImageAttr(cardRoot, "data-ondemand-img"),
	)
	card.Features = collectFeatureTexts(cardRoot)
	return card
}

func nearestCardRoot(n *html.Node) *html.Node {
	for cur := n; cur != nil; cur = cur.Parent {
		if cur.Type != html.ElementNode {
			continue
		}
		if cur.Data != "article" && cur.Data != "div" && cur.Data != "li" {
			continue
		}
		class := strings.ToLower(getAttr(cur, "class"))
		if class == "" {
			continue
		}
		if (strings.Contains(class, "item") || strings.Contains(class, "listing") || strings.Contains(class, "result")) &&
			!strings.Contains(class, "item-link") &&
			!strings.Contains(class, "item-price") {
			return cur
		}
	}
	return n.Parent
}

func firstTextByClass(root *html.Node, classNeedle string) string {
	if root == nil {
		return ""
	}
	var out string
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if out != "" {
			return
		}
		if n.Type == html.ElementNode && classContains(n, classNeedle) {
			out = nodeText(n)
			return
		}
		for child := n.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
	}
	walk(root)
	return out
}

func collectFeatureTexts(root *html.Node) []string {
	if root == nil {
		return nil
	}
	var out []string
	seen := map[string]bool{}
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode {
			class := strings.ToLower(getAttr(n, "class"))
			if strings.Contains(class, "item-detail") {
				text := nodeText(n)
				if text != "" && !seen[text] && !strings.Contains(text, "€") && text != firstTextByClass(root, "item-detail-location") {
					seen[text] = true
					out = append(out, text)
				}
			}
		}
		for child := n.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
	}
	walk(root)
	if len(out) > 6 {
		out = out[:6]
	}
	return out
}

func firstImageAttr(root *html.Node, attr string) string {
	if root == nil {
		return ""
	}
	var out string
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if out != "" {
			return
		}
		if n.Type == html.ElementNode && n.Data == "img" {
			out = normalizeText(getAttr(n, attr))
			if out != "" {
				return
			}
		}
		for child := n.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
	}
	walk(root)
	return out
}

func classContains(n *html.Node, needle string) bool {
	class := strings.ToLower(getAttr(n, "class"))
	return strings.Contains(class, strings.ToLower(needle))
}

func getAttr(n *html.Node, key string) string {
	if n == nil {
		return ""
	}
	for _, attr := range n.Attr {
		if strings.EqualFold(attr.Key, key) {
			return attr.Val
		}
	}
	return ""
}

func nodeText(n *html.Node) string {
	if n == nil {
		return ""
	}
	var parts []string
	var walk func(*html.Node)
	walk = func(cur *html.Node) {
		switch cur.Type {
		case html.TextNode:
			text := normalizeText(cur.Data)
			if text != "" {
				parts = append(parts, text)
			}
		case html.ElementNode:
			if cur.Data == "script" || cur.Data == "style" {
				return
			}
		}
		for child := cur.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
	}
	walk(n)
	return normalizeText(strings.Join(parts, " "))
}

func normalizeSegmentText(raw string) string {
	return normalizeText(html.UnescapeString(tagPattern.ReplaceAllString(raw, " ")))
}

func normalizeText(raw string) string {
	return strings.TrimSpace(spacePattern.ReplaceAllString(raw, " "))
}

func extractAnchorText(segment string) string {
	start := strings.Index(strings.ToLower(segment), "<a")
	if start < 0 {
		return ""
	}
	end := strings.Index(strings.ToLower(segment[start:]), "</a>")
	if end < 0 {
		return ""
	}
	return normalizeSegmentText(segment[start : start+end])
}

func firstRegexMatch(raw string, pattern *regexp.Regexp) string {
	match := pattern.FindString(raw)
	return normalizeText(match)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func absolutizeListingURL(href string) string {
	trimmed := strings.TrimSpace(href)
	if trimmed == "" {
		return ""
	}
	if strings.HasPrefix(trimmed, "http://") || strings.HasPrefix(trimmed, "https://") {
		return trimmed
	}
	if strings.HasPrefix(trimmed, "/") {
		return "https://www.idealista.pt" + trimmed
	}
	if parsed, err := url.Parse(trimmed); err == nil && parsed.Host != "" {
		return parsed.String()
	}
	return trimmed
}

func shortlistCards(cards []listingSearchCard, limit int) []listingSearchCard {
	if limit <= 0 || limit >= len(cards) {
		return slices.Clone(cards)
	}
	return slices.Clone(cards[:limit])
}

func selectedListingIDs(cards []listingSearchCard) []string {
	ids := make([]string, 0, len(cards))
	for _, card := range cards {
		if card.ListingID != "" {
			ids = append(ids, card.ListingID)
		}
	}
	return ids
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
