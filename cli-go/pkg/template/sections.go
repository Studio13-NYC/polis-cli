package template

import (
	"fmt"
	"regexp"
	"strings"
)

// sectionOpenPattern matches {{#name}} opening tags.
var sectionOpenPattern = regexp.MustCompile(`\{\{#(\w+)\}\}`)

// processSections expands all {{#section}}...{{/section}} loops in the template.
// Supported sections:
// - {{#posts}}...{{/posts}} - Loop over posts
// - {{#comments}}...{{/comments}} - Loop over comments
// - {{#blessed_comments}}...{{/blessed_comments}} - Loop over blessed comments on a post
// - {{#recent_posts}}...{{/recent_posts}} - Loop over 3 most recent posts
func (e *Engine) processSections(template string, ctx *RenderContext, depth int) (string, error) {
	// Process sections iteratively since Go regex doesn't support backreferences
	result := template
	var lastErr error

	// Keep processing until no more sections are found
	for {
		match := sectionOpenPattern.FindStringSubmatchIndex(result)
		if match == nil {
			break
		}

		// Extract the section name
		sectionName := result[match[2]:match[3]]
		openTag := result[match[0]:match[1]]
		closeTag := "{{/" + sectionName + "}}"

		// Find the matching close tag
		openTagEnd := match[1]
		closeTagStart := strings.Index(result[openTagEnd:], closeTag)
		if closeTagStart == -1 {
			// No matching close tag, skip this opening tag
			break
		}
		closeTagStart += openTagEnd

		// Extract section content
		sectionContent := result[openTagEnd:closeTagStart]

		var output string
		var err error

		switch sectionName {
		case "posts":
			output, err = e.renderPostsSection(sectionContent, ctx, depth)
		case "comments":
			output, err = e.renderCommentsSection(sectionContent, ctx, depth)
		case "blessed_comments":
			output, err = e.renderBlessedCommentsSection(sectionContent, ctx, depth)
		case "recent_posts":
			output, err = e.renderRecentPostsSection(sectionContent, ctx, depth)
		default:
			// Unknown section - leave as-is and continue
			break
		}

		if err != nil {
			lastErr = err
			break
		}

		// Replace the section with its rendered output
		result = result[:match[0]] + output + result[closeTagStart+len(closeTag):]

		// Avoid checking unsupported section names again
		if sectionName != "posts" && sectionName != "comments" && sectionName != "blessed_comments" && sectionName != "recent_posts" {
			// Skip to after this section to avoid infinite loop on unknown sections
			result = result[:match[0]] + openTag + sectionContent + closeTag + result[match[0]:]
			break
		}
	}

	return result, lastErr
}

// renderPostsSection renders the {{#posts}} section for each post.
func (e *Engine) renderPostsSection(content string, ctx *RenderContext, depth int) (string, error) {
	var builder strings.Builder

	for _, post := range ctx.Posts {
		// Create a temporary context for this iteration
		iterCtx := &RenderContext{
			URL:            post.URL,
			Title:          post.Title,
			Published:      post.Published,
			PublishedHuman: post.PublishedHuman,
			CommentCount:   post.CommentCount,

			// Copy site-level variables
			SiteURL:   ctx.SiteURL,
			SiteTitle: ctx.SiteTitle,
			Year:      ctx.Year,
		}

		// Substitute loop-specific variables
		rendered := e.substituteLoopVariables(content, map[string]string{
			"url":             post.URL,
			"title":           post.Title,
			"published":       post.Published,
			"published_human": post.PublishedHuman,
			"comment_count":   fmt.Sprintf("%d", post.CommentCount),
		})

		// Process any nested partials
		processed, err := e.processPartials(rendered, iterCtx, depth+1)
		if err != nil {
			return "", err
		}

		builder.WriteString(processed)
	}

	return builder.String(), nil
}

// renderCommentsSection renders the {{#comments}} section for each comment.
func (e *Engine) renderCommentsSection(content string, ctx *RenderContext, depth int) (string, error) {
	var builder strings.Builder

	for _, comment := range ctx.Comments {
		// Create a temporary context for this iteration
		iterCtx := &RenderContext{
			URL:            comment.URL,
			Published:      comment.Published,
			PublishedHuman: comment.PublishedHuman,

			// Copy site-level variables
			SiteURL:   ctx.SiteURL,
			SiteTitle: ctx.SiteTitle,
			Year:      ctx.Year,
		}

		// Substitute loop-specific variables
		rendered := e.substituteLoopVariables(content, map[string]string{
			"url":             comment.URL,
			"target_author":   comment.TargetAuthor,
			"published":       comment.Published,
			"published_human": comment.PublishedHuman,
			"preview":         comment.Preview,
		})

		// Process any nested partials
		processed, err := e.processPartials(rendered, iterCtx, depth+1)
		if err != nil {
			return "", err
		}

		builder.WriteString(processed)
	}

	return builder.String(), nil
}

// renderBlessedCommentsSection renders the {{#blessed_comments}} section for each blessed comment.
func (e *Engine) renderBlessedCommentsSection(content string, ctx *RenderContext, depth int) (string, error) {
	var builder strings.Builder

	for _, bc := range ctx.BlessedComments {
		// Create a temporary context for this iteration
		iterCtx := &RenderContext{
			URL:            bc.URL,
			AuthorName:     bc.AuthorName,
			Published:      bc.Published,
			PublishedHuman: bc.PublishedHuman,
			Content:        bc.Content,

			// Copy site-level variables
			SiteURL:   ctx.SiteURL,
			SiteTitle: ctx.SiteTitle,
			Year:      ctx.Year,
		}

		// Substitute loop-specific variables
		rendered := e.substituteLoopVariables(content, map[string]string{
			"url":             bc.URL,
			"author_name":     bc.AuthorName,
			"published":       bc.Published,
			"published_human": bc.PublishedHuman,
			"content":         bc.Content,
		})

		// Process any nested partials
		processed, err := e.processPartials(rendered, iterCtx, depth+1)
		if err != nil {
			return "", err
		}

		builder.WriteString(processed)
	}

	return builder.String(), nil
}

// renderRecentPostsSection renders the {{#recent_posts}} section.
// This shows the 3 most recent posts.
func (e *Engine) renderRecentPostsSection(content string, ctx *RenderContext, depth int) (string, error) {
	// Use RecentPosts if available, otherwise use first 3 posts
	posts := ctx.RecentPosts
	if len(posts) == 0 && len(ctx.Posts) > 0 {
		limit := 3
		if len(ctx.Posts) < limit {
			limit = len(ctx.Posts)
		}
		posts = ctx.Posts[:limit]
	}

	var builder strings.Builder

	for _, post := range posts {
		// Create a temporary context for this iteration
		iterCtx := &RenderContext{
			URL:            post.URL,
			Title:          post.Title,
			Published:      post.Published,
			PublishedHuman: post.PublishedHuman,
			CommentCount:   post.CommentCount,

			// Copy site-level variables
			SiteURL:   ctx.SiteURL,
			SiteTitle: ctx.SiteTitle,
			Year:      ctx.Year,
		}

		// Substitute loop-specific variables
		rendered := e.substituteLoopVariables(content, map[string]string{
			"url":             post.URL,
			"title":           post.Title,
			"published":       post.Published,
			"published_human": post.PublishedHuman,
			"comment_count":   fmt.Sprintf("%d", post.CommentCount),
		})

		// Process any nested partials
		processed, err := e.processPartials(rendered, iterCtx, depth+1)
		if err != nil {
			return "", err
		}

		builder.WriteString(processed)
	}

	return builder.String(), nil
}

// substituteLoopVariables replaces {{variable}} with values from a map.
// This is used for loop-specific variables within section content.
func (e *Engine) substituteLoopVariables(template string, vars map[string]string) string {
	re := regexp.MustCompile(`\{\{(\w+)\}\}`)
	return re.ReplaceAllStringFunc(template, func(match string) string {
		name := match[2 : len(match)-2]
		if val, ok := vars[name]; ok {
			return val
		}
		return match
	})
}
