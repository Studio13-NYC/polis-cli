package server

import "net/http"

// SetupRoutes registers all API routes on the given ServeMux.
func SetupRoutes(mux *http.ServeMux, s *Server) {
	// API routes
	mux.HandleFunc("/api/status", s.handleStatus)
	mux.HandleFunc("/api/validate", s.handleValidate)
	mux.HandleFunc("/api/init", s.handleInit)
	mux.HandleFunc("/api/link", s.handleLink)
	mux.HandleFunc("/api/render", s.handleRender)
	mux.HandleFunc("/api/publish", s.handlePublish)
	mux.HandleFunc("/api/drafts", s.handleDrafts)
	mux.HandleFunc("/api/drafts/", s.handleDraft)
	mux.HandleFunc("/api/posts", s.handlePosts)
	mux.HandleFunc("/api/posts/", s.handlePost)
	mux.HandleFunc("/api/republish", s.handleRepublish)

	// Comment API routes (MY comments - outgoing)
	mux.HandleFunc("/api/comments/drafts", s.handleCommentDrafts)
	mux.HandleFunc("/api/comments/drafts/", s.handleCommentDraft)
	mux.HandleFunc("/api/comments/sign", s.handleCommentSign)
	mux.HandleFunc("/api/comments/beseech", s.handleCommentBeseech)
	mux.HandleFunc("/api/comments/pending", s.handleCommentsPending)
	mux.HandleFunc("/api/comments/pending/", s.handleCommentByStatus)
	mux.HandleFunc("/api/comments/blessed", s.handleCommentsBlessed)
	mux.HandleFunc("/api/comments/blessed/", s.handleCommentByStatus)
	mux.HandleFunc("/api/comments/denied", s.handleCommentsDenied)
	mux.HandleFunc("/api/comments/denied/", s.handleCommentByStatus)
	mux.HandleFunc("/api/comments/sync", s.handleCommentsSync)

	// Blessing API routes (ON MY POSTS - incoming blessing requests)
	mux.HandleFunc("/api/blessing/requests", s.handleBlessingRequests)
	mux.HandleFunc("/api/blessing/grant", s.handleBlessingGrant)
	mux.HandleFunc("/api/blessing/deny", s.handleBlessingDeny)
	mux.HandleFunc("/api/blessing/revoke", s.handleBlessingRevoke)
	mux.HandleFunc("/api/blessed-comments", s.handleBlessedComments)

	// Settings and automation API routes
	mux.HandleFunc("/api/settings", s.handleSettings)
	mux.HandleFunc("/api/settings/view-mode", s.handleViewMode)
	mux.HandleFunc("/api/settings/show-frontmatter", s.handleShowFrontmatter)
	mux.HandleFunc("/api/settings/hide-read", s.handleHideRead)
	mux.HandleFunc("/api/settings/site-title", s.handleUpdateSiteTitle)
	mux.HandleFunc("/api/settings/theme", s.handleThemeSwitch)
	mux.HandleFunc("/api/download-site", s.handleDownloadSite)
	mux.HandleFunc("/api/content/", s.handleContent)
	mux.HandleFunc("/api/automations", s.handleAutomations)
	mux.HandleFunc("/api/automations/quick", s.handleAutomationsQuick)
	mux.HandleFunc("/api/automations/", s.handleAutomation)
	mux.HandleFunc("/api/templates", s.handleTemplates)
	mux.HandleFunc("/api/hooks/generate", s.handleHooksGenerate)

	// Site registration API routes
	mux.HandleFunc("/api/site/registration-status", s.handleSiteRegistrationStatus)
	mux.HandleFunc("/api/site/register", s.handleSiteRegister)
	mux.HandleFunc("/api/site/unregister", s.handleSiteUnregister)
	mux.HandleFunc("/api/site/deploy-check", s.handleDeployCheck)
	mux.HandleFunc("/api/site/setup-wizard-dismiss", s.handleSetupWizardDismiss)

	// About page API route
	mux.HandleFunc("/api/about", s.handleAbout)

	// Snippets API routes
	mux.HandleFunc("/api/snippets", s.handleSnippets)
	mux.HandleFunc("/api/snippets/", s.handleSnippet)

	// Social API routes (following, feed, remote content)
	mux.HandleFunc("/api/following", s.handleFollowing)
	mux.HandleFunc("/api/feed", s.handleFeed)
	mux.HandleFunc("/api/feed/refresh", s.handleFeedRefresh)
	mux.HandleFunc("/api/feed/read", s.handleFeedRead)
	mux.HandleFunc("/api/feed/counts", s.handleFeedCounts)
	mux.HandleFunc("/api/feed/grouped", s.handleFeedGrouped)
	mux.HandleFunc("/api/remote/post", s.handleRemotePost)

	// Notification API routes
	mux.HandleFunc("/api/notifications", s.handleNotifications)
	mux.HandleFunc("/api/notifications/count", s.handleNotificationCount)
	mux.HandleFunc("/api/notifications/read", s.handleNotificationRead)

	// Social plugin routes
	mux.HandleFunc("/api/pulse", s.handlePulse)
	mux.HandleFunc("/api/activity", s.handleActivityStream)
	mux.HandleFunc("/api/conversations", s.handleConversations)
	mux.HandleFunc("/api/followers/count", s.handleFollowerCount)

	// Render API routes (for snippet editing workflow)
	mux.HandleFunc("/api/render-page", s.handleRenderPage)

	// SSE and consolidated counts routes
	mux.HandleFunc("/api/sse", s.handleSSE)
	mux.HandleFunc("/api/counts", s.handleCounts)

	// Widget API routes (cross-origin, widget token auth)
	mux.HandleFunc("/api/widget/publish", s.handleWidgetPublish)
	mux.HandleFunc("/api/widget/comment", s.handleWidgetComment)
	mux.HandleFunc("/api/widget/follow", s.handleWidgetFollow)
	mux.HandleFunc("/api/widget/connect", s.handleWidgetConnect)
}
