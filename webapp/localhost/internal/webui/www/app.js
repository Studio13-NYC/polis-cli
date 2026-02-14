// Polis Local App - Client-side JavaScript

const App = {
    currentDraftId: null,
    currentPostPath: null,  // Set when editing a published post
    currentFrontmatter: '',  // Stored frontmatter block for published posts
    currentCommentDraftId: null,
    currentView: 'posts-published',  // Current active view in sidebar
    sidebarMode: 'my-site',  // 'my-site' or 'social'
    filenameManuallySet: false,  // Track if user manually edited the filename

    // View mode state: 'list' or 'browser'
    viewMode: 'list',

    // Setup wizard state
    setupWizardStep: 0,       // 0=configure, 1=deploy, 2=register
    setupWizardDeployTimer: null,
    setupWizardDismissed: false,
    siteRegistered: false,

    // Site info (loaded from /api/settings)
    siteInfo: null,

    // Browser mode state
    browserState: {
        history: [],
        historyIndex: -1,
        currentUrl: null,
        currentContent: null,
        isEditing: false,
        originalMarkdown: null,  // Store original for cancel
        snippetEditMode: false,  // Toggle for snippet editing mode
        editingSnippet: null,    // { id, path, source, content } when editing a snippet
        showIncludes: false,     // Toggle for viewing includes in snippet
        snippetNavStack: [],     // Stack for nested snippet navigation: [{path, source, content}]
    },

    // Snippet state
    snippetState: {
        currentPath: '',
        editingPath: null,
        editingSource: null,
        activeTheme: null,
        filter: 'all',  // Filter: "all", "global", or "theme"
    },

    // Sample data for Mustache template preview
    snippetSampleData: {
        url: 'https://alice.polis.pub/posts/2026/01/sample-post.md',
        title: 'Sample Post Title',
        published_human: 'Jan 30, 2026',
        target_author: 'alice.polis.pub',
        author_name: 'Alice',
        preview: 'This is a preview of the comment content...',
        comment_count: '3',
        content: '<p>This is the full comment content with <strong>formatting</strong>.</p>',
    },

    // Data cache for counts
    counts: {
        posts: 0,
        drafts: 0,
        // My comments (outgoing)
        myPending: 0,
        myBlessed: 0,
        myDenied: 0,
        myCommentDrafts: 0,
        // Incoming (on my posts)
        incomingPending: 0,
        incomingBlessed: 0,
        // Social
        feed: 0,
        feedUnread: 0,
        following: 0,
    },

    // Feed state
    _feedItems: null,
    _feedTypeFilter: '',
    _feedRefreshing: false,
    _hideRead: false,

    // Screen management
    screens: {
        welcome: document.getElementById('welcome-screen'),
        error: document.getElementById('error-screen'),
        dashboard: document.getElementById('dashboard-screen'),
        editor: document.getElementById('editor-screen'),
        comment: document.getElementById('comment-screen'),
        snippet: document.getElementById('snippet-screen'),
    },

    // Browser mode elements
    browserElements: {
        container: document.getElementById('browser-container'),
        mainContainer: document.querySelector('.main-container'),
        backBtn: document.getElementById('browser-back'),
        forwardBtn: document.getElementById('browser-forward'),
        homeBtn: document.getElementById('browser-home'),
        urlInput: document.getElementById('browser-url-input'),
        settingsBtn: document.getElementById('browser-settings'),
        markdownDisplay: document.getElementById('browser-markdown-display'),
        markdownEdit: document.getElementById('browser-markdown-edit'),
        previewContent: document.getElementById('browser-preview-content'),
        actionsFooter: document.getElementById('browser-actions'),
        cancelBtn: document.getElementById('browser-cancel'),
        saveDraftBtn: document.getElementById('browser-save-draft'),
        publishBtn: document.getElementById('browser-publish'),
        listModeBtn: document.getElementById('list-mode-btn'),
        browserModeBtn: document.getElementById('browser-mode-btn'),
        frontmatterToggle: document.getElementById('frontmatter-toggle'),
        editBtn: document.getElementById('browser-edit-btn'),
        snippetEditToggle: document.getElementById('browser-snippet-edit'),
        codePaneLabel: document.getElementById('browser-code-pane-label'),
        snippetParentLink: document.getElementById('browser-snippet-parent-link'),
        includesToggle: document.getElementById('includes-toggle'),
    },

    // Site base URL for live links
    siteBaseUrl: '',

    // Show frontmatter in markdown pane (default true)
    showFrontmatter: true,

    showScreen(name) {
        Object.values(this.screens).forEach(s => {
            if (s) s.classList.add('hidden');
        });
        if (this.screens[name]) {
            this.screens[name].classList.remove('hidden');
        }
    },

    // Toast notification system
    showToast(message, type = 'info', duration = 4000) {
        const container = document.getElementById('toast-container');
        const toast = document.createElement('div');
        toast.className = `toast ${type}`;

        const icons = {
            success: '&#10003;',
            error: '&#10007;',
            warning: '!',
            info: 'i',
        };

        toast.innerHTML = `
            <div class="toast-icon">${icons[type] || icons.info}</div>
            <div class="toast-message">${this.escapeHtml(message)}</div>
            <button class="toast-close" onclick="this.parentElement.remove()">&times;</button>
        `;

        container.appendChild(toast);

        // Auto-dismiss
        if (duration > 0) {
            setTimeout(() => {
                toast.classList.add('toast-out');
                setTimeout(() => toast.remove(), 200);
            }, duration);
        }

        return toast;
    },

    // Confirm modal (replaces browser confirm())
    showConfirmModal(title, message, confirmText = 'Confirm', cancelText = 'Cancel', type = 'default') {
        return new Promise((resolve) => {
            const modal = document.createElement('div');
            modal.className = 'modal-overlay';

            const typeClass = type === 'danger' ? 'danger' : 'primary';

            modal.innerHTML = `
                <div class="modal confirm-modal">
                    <div class="modal-header">
                        <h3>${this.escapeHtml(title)}</h3>
                        <button class="modal-close" data-action="cancel">&times;</button>
                    </div>
                    <div class="modal-body">
                        <p>${this.escapeHtml(message)}</p>
                    </div>
                    <div class="modal-footer">
                        <button class="secondary" data-action="cancel">${this.escapeHtml(cancelText)}</button>
                        <button class="${typeClass}" data-action="confirm">${this.escapeHtml(confirmText)}</button>
                    </div>
                </div>
            `;

            const cleanup = (result) => {
                modal.remove();
                resolve(result);
            };

            modal.querySelectorAll('[data-action="cancel"]').forEach(btn => {
                btn.addEventListener('click', () => cleanup(false));
            });
            modal.querySelector('[data-action="confirm"]').addEventListener('click', () => cleanup(true));
            modal.addEventListener('click', (e) => {
                if (e.target === modal) cleanup(false);
            });

            document.body.appendChild(modal);
            modal.querySelector('[data-action="confirm"]').focus();
        });
    },

    // API calls
    async api(method, path, body = null) {
        const options = {
            method,
            headers: { 'Content-Type': 'application/json' },
        };
        if (body) {
            options.body = JSON.stringify(body);
        }
        const response = await fetch(path, options);
        if (!response.ok) {
            const text = await response.text();
            throw new Error(text || response.statusText);
        }
        return response.json();
    },

    // Initialize app
    async init() {
        try {
            const status = await this.api('GET', '/api/status');
            const validation = status.validation || {};

            switch (validation.status) {
                case 'valid':
                    document.getElementById('domain-display').textContent =
                        status.site_title || '';
                    // Show domain in header
                    this.updateDomainDisplay(status.base_url);
                    // Store base URL for live links in browser mode
                    this.siteBaseUrl = status.base_url || '';
                    // Load frontmatter visibility setting
                    this.showFrontmatter = status.show_frontmatter !== false;
                    this.updateFrontmatterToggle();
                    await this.loadAllCounts();
                    await this.initViewMode();
                    await this.loadViewContent();
                    this.initNotifications();
                    this.initFeedPolling();
                    this.showScreen('dashboard');
                    this.checkSetupBanner();
                    break;

                case 'not_found':
                    this.showScreen('welcome');
                    break;

                case 'incomplete':
                case 'invalid':
                    this.renderValidationErrors(validation.errors || []);
                    this.showScreen('error');
                    break;

                default:
                    // Legacy fallback for backwards compatibility
                    if (status.configured) {
                        document.getElementById('domain-display').textContent =
                            status.site_title || '';
                        this.updateDomainDisplay(status.base_url);
                        // Store base URL for live links in browser mode
                        this.siteBaseUrl = status.base_url || '';
                        // Load frontmatter visibility setting
                        this.showFrontmatter = status.show_frontmatter !== false;
                        this.updateFrontmatterToggle();
                        this.initNotifications();
                        await this.loadAllCounts();
                        await this.initViewMode();
                        await this.loadViewContent();
                        this.showScreen('dashboard');
                        this.checkSetupBanner();
                    } else {
                        this.showScreen('welcome');
                    }
            }
        } catch (err) {
            console.error('Failed to check status:', err);
            this.showScreen('welcome');
        }

        this.bindEvents();
    },

    // Render validation errors on the error screen
    renderValidationErrors(errors) {
        const container = document.getElementById('validation-errors');
        if (!container) return;

        if (errors.length === 0) {
            container.innerHTML = '<div class="error-item"><p>Unknown validation error</p></div>';
            return;
        }

        container.innerHTML = errors.map(err => `
            <div class="error-item">
                <div class="error-item-header">
                    <span class="error-code">${this.escapeHtml(err.code)}</span>
                </div>
                <p class="error-message">${this.escapeHtml(err.message)}</p>
                ${err.path ? `<p class="error-path">Path: <code>${this.escapeHtml(err.path)}</code></p>` : ''}
                ${err.suggestion ? `<p class="error-suggestion">${this.escapeHtml(err.suggestion)}</p>` : ''}
            </div>
        `).join('');
    },

    // Retry validation (reload the page essentially)
    async retryValidation() {
        await this.init();
    },

    // Load all counts for sidebar badges
    async loadAllCounts() {
        try {
            // Load posts count
            const posts = await this.api('GET', '/api/posts');
            this.counts.posts = (posts.posts || []).length;

            // Load drafts count
            const drafts = await this.api('GET', '/api/drafts');
            this.counts.drafts = (drafts.drafts || []).length;

            // Load MY comment counts (outgoing)
            const myPending = await this.api('GET', '/api/comments/pending');
            this.counts.myPending = (myPending.comments || []).length;

            const myBlessed = await this.api('GET', '/api/comments/blessed');
            this.counts.myBlessed = (myBlessed.comments || []).length;

            const myDenied = await this.api('GET', '/api/comments/denied');
            this.counts.myDenied = (myDenied.comments || []).length;

            const commentDrafts = await this.api('GET', '/api/comments/drafts');
            this.counts.myCommentDrafts = (commentDrafts.drafts || []).length;

            // Load INCOMING counts (on my posts)
            try {
                const incomingRequests = await this.api('GET', '/api/blessing/requests');
                this.counts.incomingPending = (incomingRequests.requests || []).length;
            } catch (e) {
                // Discovery service not configured
                this.counts.incomingPending = 0;
            }

            try {
                const blessedComments = await this.api('GET', '/api/blessed-comments');
                let incomingBlessed = 0;
                if (blessedComments.comments) {
                    for (const pc of blessedComments.comments) {
                        incomingBlessed += (pc.blessed || []).length;
                    }
                }
                this.counts.incomingBlessed = incomingBlessed;
            } catch (e) {
                this.counts.incomingBlessed = 0;
            }

            // Load social counts
            try {
                const followingData = await this.api('GET', '/api/following');
                this.counts.following = followingData.count || 0;
            } catch (e) {
                this.counts.following = 0;
            }

            try {
                const feedCounts = await this.api('GET', '/api/feed/counts');
                this.counts.feed = feedCounts.total || 0;
                this.counts.feedUnread = feedCounts.unread || 0;
            } catch (e) {
                this.counts.feed = 0;
                this.counts.feedUnread = 0;
            }

            try {
                const followerData = await this.api('GET', '/api/followers/count');
                this.counts.followers = followerData.count || 0;
            } catch (e) {
                this.counts.followers = 0;
            }

            this.updateBadges();
        } catch (err) {
            console.error('Failed to load counts:', err);
        }
    },

    // Update sidebar badges
    updateBadges() {
        this.updateBadge('posts-count', this.counts.posts);
        this.updateBadge('drafts-count', this.counts.drafts);
        // My comments (outgoing)
        this.updateBadge('my-comment-drafts-count', this.counts.myCommentDrafts);
        this.updateBadge('my-pending-count', this.counts.myPending, true);
        this.updateBadge('my-blessed-count', this.counts.myBlessed);
        this.updateBadge('my-denied-count', this.counts.myDenied);
        // Incoming (on my posts)
        this.updateBadge('incoming-pending-count', this.counts.incomingPending, true);
        this.updateBadge('incoming-blessed-count', this.counts.incomingBlessed);
        // Social
        this.updateBadge('feed-count', this.counts.feedUnread, this.counts.feedUnread > 0);
        this.updateBadge('following-count', this.counts.following);
        this.updateBadge('followers-count', this.counts.followers);
    },

    updateBadge(id, count, isWarning = false) {
        const badge = document.getElementById(id);
        if (badge) {
            badge.textContent = count;
            badge.style.display = count > 0 ? 'inline' : 'none';
            if (isWarning && count > 0) {
                badge.classList.add('warning');
            } else {
                badge.classList.remove('warning');
            }
        }
    },

    // Load content for current view
    async loadViewContent() {
        const contentTitle = document.getElementById('content-title');
        const contentActions = document.getElementById('content-actions');
        const contentList = document.getElementById('content-list');

        switch (this.currentView) {
            case 'posts-published':
                contentTitle.textContent = 'Published Posts';
                contentActions.innerHTML = '<button id="new-post-btn" class="primary" onclick="App.newPost()">New Post</button>';
                await this.renderPostsList(contentList);
                break;

            case 'posts-drafts':
                contentTitle.textContent = 'Post Drafts';
                contentActions.innerHTML = '<button id="new-post-btn" class="primary" onclick="App.newPost()">New Post</button>';
                await this.renderDraftsList(contentList);
                break;

            // MY COMMENTS (outgoing - I wrote these)
            case 'my-comments-drafts':
                contentTitle.textContent = 'My Comment Drafts';
                contentActions.innerHTML = '<button class="primary" onclick="App.newComment()">New Comment</button>';
                await this.renderMyCommentDraftsList(contentList);
                break;

            case 'my-comments-pending':
                contentTitle.textContent = 'My Pending Comments';
                contentActions.innerHTML = `
                    <button id="sync-comments-btn" class="secondary sync-btn" onclick="App.syncComments()">Sync Status</button>
                    <button class="primary" onclick="App.newComment()">New Comment</button>
                `;
                await this.renderMyCommentsList(contentList, 'pending');
                break;

            case 'my-comments-blessed':
                contentTitle.textContent = 'My Blessed Comments';
                contentActions.innerHTML = '<button class="primary" onclick="App.newComment()">New Comment</button>';
                await this.renderMyCommentsList(contentList, 'blessed');
                break;

            case 'my-comments-denied':
                contentTitle.textContent = 'My Denied Comments';
                contentActions.innerHTML = '<button class="primary" onclick="App.newComment()">New Comment</button>';
                await this.renderMyCommentsList(contentList, 'denied');
                break;

            // ON MY POSTS (incoming - others wrote these)
            case 'incoming-pending':
                contentTitle.textContent = 'Pending Blessing Requests';
                contentActions.innerHTML = '';
                await this.renderIncomingPendingList(contentList);
                break;

            case 'incoming-blessed':
                contentTitle.textContent = 'Blessed Comments on My Posts';
                contentActions.innerHTML = '';
                await this.renderIncomingBlessedList(contentList);
                break;

            case 'settings':
                contentTitle.textContent = 'Settings';
                contentActions.innerHTML = '';
                this.renderSettings(contentList);
                break;

            case 'snippets':
                contentTitle.textContent = 'All Snippets';
                contentActions.innerHTML = '<button class="primary" onclick="App.newSnippet()">New Snippet</button>';
                this.snippetState.filter = 'all';
                await this.renderSnippetsList(contentList);
                break;

            case 'snippets-global':
                contentTitle.textContent = 'Global Snippets';
                contentActions.innerHTML = '<button class="primary" onclick="App.newSnippet()">New Snippet</button>';
                this.snippetState.filter = 'global';
                await this.renderSnippetsList(contentList);
                break;

            case 'snippets-theme':
                contentTitle.textContent = 'Theme Snippets';
                contentActions.innerHTML = '';
                this.snippetState.filter = 'theme';
                await this.renderSnippetsList(contentList);
                break;

            // Social views
            case 'feed':
                contentTitle.textContent = 'Conversations';
                contentActions.innerHTML = '<button class="secondary sync-btn" onclick="App.markAllFeedRead()">Mark All Read</button> <button class="secondary sync-btn" onclick="App.refreshFeed()">Refresh</button>';
                await this.renderFeedList(contentList);
                break;

            case 'following':
                contentTitle.textContent = 'Following';
                contentActions.innerHTML = '<button class="primary" onclick="App.openFollowPanel()">Follow Author</button>';
                await this.renderFollowingList(contentList);
                break;

            case 'activity':
                contentTitle.textContent = 'Activity';
                contentActions.innerHTML = '<button class="secondary sync-btn" onclick="App.resetActivity()">Reset</button> <button class="secondary sync-btn" onclick="App.refreshActivity()">Refresh</button>';
                await this.renderActivityStream(contentList);
                break;

            case 'followers':
                contentTitle.textContent = 'Followers';
                contentActions.innerHTML = '<button class="secondary sync-btn" onclick="App.refreshFollowers(true)">Full Refresh</button>';
                await this.renderFollowersList(contentList);
                break;

        }
    },

    // Set active view and update UI
    setActiveView(view) {
        this.currentView = view;

        // Update sidebar active state
        document.querySelectorAll('.sidebar .nav-item').forEach(item => {
            item.classList.remove('active');
            if (item.dataset.view === view) {
                item.classList.add('active');
            }
        });

        // Load content for the view
        this.loadViewContent();
    },

    // Bind event handlers
    bindEvents() {
        // Init panel events
        const initCloseBtn = document.getElementById('init-close-btn');
        const initCancelBtn = document.getElementById('init-cancel-btn');
        const initExecuteBtn = document.getElementById('init-execute-btn');
        const initOverlay = document.querySelector('#init-panel .wizard-overlay');

        if (initCloseBtn) initCloseBtn.addEventListener('click', () => this.closeInitPanel());
        if (initCancelBtn) initCancelBtn.addEventListener('click', () => this.closeInitPanel());
        if (initExecuteBtn) initExecuteBtn.addEventListener('click', () => this.executeInit());
        if (initOverlay) initOverlay.addEventListener('click', () => this.closeInitPanel());

        // Link panel events
        const linkCloseBtn = document.getElementById('link-close-btn');
        const linkCancelBtn = document.getElementById('link-cancel-btn');
        const linkExecuteBtn = document.getElementById('link-execute-btn');
        const linkOverlay = document.querySelector('#link-panel .wizard-overlay');

        if (linkCloseBtn) linkCloseBtn.addEventListener('click', () => this.closeLinkPanel());
        if (linkCancelBtn) linkCancelBtn.addEventListener('click', () => this.closeLinkPanel());
        if (linkExecuteBtn) linkExecuteBtn.addEventListener('click', () => this.executeLink());
        if (linkOverlay) linkOverlay.addEventListener('click', () => this.closeLinkPanel());

        // Sidebar mode toggle
        document.querySelectorAll('.sidebar-mode-toggle .mode-tab').forEach(tab => {
            tab.addEventListener('click', () => {
                const mode = tab.dataset.sidebarMode;
                if (mode) this.setSidebarMode(mode);
            });
        });

        // Sidebar navigation
        document.querySelectorAll('.sidebar .nav-item').forEach(item => {
            item.addEventListener('click', () => {
                const view = item.dataset.view;
                if (view) {
                    this.setActiveView(view);
                }
            });
        });

        // Back button
        document.getElementById('back-btn').addEventListener('click', async () => {
            await this.loadAllCounts();
            await this.loadViewContent();
            this.showScreen('dashboard');
        });

        // Save draft button
        document.getElementById('save-draft-btn').addEventListener('click', async () => {
            await this.saveDraft();
        });

        // Publish button
        document.getElementById('publish-btn').addEventListener('click', async () => {
            await this.publish();
        });

        // Auto-generate filename from title and live preview as user types
        document.getElementById('markdown-input').addEventListener('input', (e) => {
            if (!this.filenameManuallySet && !this.currentPostPath) {
                const markdown = e.target.value;
                const title = this.extractTitleFromMarkdown(markdown);
                if (title) {
                    document.getElementById('filename-input').value = this.slugify(title);
                }
            }
            this.editorUpdatePreview();
        });

        // Editor frontmatter toggle
        const editorFmToggle = document.getElementById('editor-fm-toggle');
        if (editorFmToggle) {
            editorFmToggle.addEventListener('click', () => this.toggleEditorFrontmatter());
        }

        // Mark filename as manually set when user edits it
        document.getElementById('filename-input').addEventListener('input', () => {
            this.filenameManuallySet = true;
        });

        // Comment back button
        document.getElementById('comment-back-btn').addEventListener('click', async () => {
            await this.loadAllCounts();
            await this.loadViewContent();
            this.showScreen('dashboard');
        });

        // Save comment draft button
        document.getElementById('save-comment-draft-btn').addEventListener('click', async () => {
            await this.saveCommentDraft();
        });

        // Sign & send for blessing button
        document.getElementById('sign-send-btn').addEventListener('click', async () => {
            await this.signAndSendComment();
        });

        // Keyboard shortcuts
        document.addEventListener('keydown', (e) => {
            // Ctrl/Cmd + S to save
            if ((e.ctrlKey || e.metaKey) && e.key === 's') {
                e.preventDefault();
                if (!this.screens.editor.classList.contains('hidden')) {
                    this.saveDraft();
                } else if (!this.screens.comment.classList.contains('hidden')) {
                    this.saveCommentDraft();
                } else if (!this.screens.snippet.classList.contains('hidden')) {
                    this.saveSnippet();
                }
            }
            // Ctrl/Cmd + Enter to publish
            if ((e.ctrlKey || e.metaKey) && e.key === 'Enter') {
                e.preventDefault();
                if (!this.screens.editor.classList.contains('hidden')) {
                    this.publish();
                }
            }
        });

        // Snippet back button
        document.getElementById('snippet-back-btn').addEventListener('click', async () => {
            await this.loadViewContent();
            this.showScreen('dashboard');
        });

        // Save snippet button
        document.getElementById('save-snippet-btn').addEventListener('click', async () => {
            await this.saveSnippet();
        });

        // New snippet panel events
        document.getElementById('new-snippet-close-btn').addEventListener('click', () => {
            this.closeNewSnippetPanel();
        });
        document.getElementById('new-snippet-cancel-btn').addEventListener('click', () => {
            this.closeNewSnippetPanel();
        });
        document.getElementById('new-snippet-create-btn').addEventListener('click', async () => {
            await this.createSnippet();
        });
        document.querySelector('#new-snippet-panel .wizard-overlay').addEventListener('click', () => {
            this.closeNewSnippetPanel();
        });

        // Live preview for snippet editor (debounced)
        let snippetPreviewTimeout = null;
        document.getElementById('snippet-content').addEventListener('input', () => {
            // Debounce the preview update to avoid too many API calls
            if (snippetPreviewTimeout) {
                clearTimeout(snippetPreviewTimeout);
            }
            snippetPreviewTimeout = setTimeout(() => {
                this.updateSnippetPreview();
            }, 300);
        });

        // Browser mode events
        this.bindBrowserEvents();
    },

    // Default discovery service values (public, hardcoded to match server defaults)
    defaultDiscoveryURL: 'https://ltfpezriiaqvjupxbttw.supabase.co/functions/v1',
    defaultDiscoveryKey: 'eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJpc3MiOiJzdXBhYmFzZSIsInJlZiI6Imx0ZnBlenJpaWFxdmp1cHhidHR3Iiwicm9sZSI6ImFub24iLCJpYXQiOjE3NjcxNDQwODMsImV4cCI6MjA4MjcyMDA4M30.N9ScKbdcswutM6i__W9sPWWcBONIcxdAqIbsljqMKMI',

    // Show init flow panel
    showInitFlow() {
        const panel = document.getElementById('init-panel');
        if (panel) {
            // Clear any previous values
            const titleInput = document.getElementById('init-site-title');
            const urlInput = document.getElementById('init-base-url');
            const dsUrlInput = document.getElementById('init-discovery-url');
            const dsKeyInput = document.getElementById('init-discovery-key');
            if (titleInput) titleInput.value = '';
            if (urlInput) urlInput.value = '';
            if (dsUrlInput) dsUrlInput.value = this.defaultDiscoveryURL;
            if (dsKeyInput) dsKeyInput.value = this.defaultDiscoveryKey;
            panel.classList.remove('hidden');
        }
    },

    // Close init panel
    closeInitPanel() {
        const panel = document.getElementById('init-panel');
        if (panel) panel.classList.add('hidden');
    },

    // Execute site initialization
    async executeInit() {
        const titleInput = document.getElementById('init-site-title');
        const urlInput = document.getElementById('init-base-url');
        const dsUrlInput = document.getElementById('init-discovery-url');
        const dsKeyInput = document.getElementById('init-discovery-key');
        const executeBtn = document.getElementById('init-execute-btn');

        const siteTitle = titleInput ? titleInput.value.trim() : '';
        const baseUrl = urlInput ? urlInput.value.trim() : '';
        const discoveryUrl = dsUrlInput ? dsUrlInput.value.trim() : '';
        const discoveryKey = dsKeyInput ? dsKeyInput.value.trim() : '';

        // Disable button while processing
        if (executeBtn) {
            executeBtn.disabled = true;
            executeBtn.textContent = 'Initializing...';
        }

        try {
            const result = await this.api('POST', '/api/init', {
                site_title: siteTitle,
                base_url: baseUrl,
                discovery_url: discoveryUrl,
                discovery_key: discoveryKey,
            });

            this.closeInitPanel();
            this.showToast('Site initialized successfully!', 'success');

            // Update display and show dashboard
            document.getElementById('domain-display').textContent =
                result.site_title || '';
            this.updateDomainDisplay(result.base_url);
            this.siteBaseUrl = result.base_url || '';
            this.initNotifications();
            await this.loadAllCounts();
            await this.initViewMode();
            await this.loadViewContent();
            this.showScreen('dashboard');

            // Open setup wizard to guide through deploy & register
            this.openSetupWizard();
        } catch (err) {
            this.showToast('Failed to initialize site: ' + err.message, 'error');
        } finally {
            if (executeBtn) {
                executeBtn.disabled = false;
                executeBtn.textContent = 'Initialize Site';
            }
        }
    },

    // Show link flow panel
    showLinkFlow() {
        const panel = document.getElementById('link-panel');
        if (panel) {
            // Clear any previous values
            const pathInput = document.getElementById('link-path');
            if (pathInput) pathInput.value = '';
            panel.classList.remove('hidden');
        }
    },

    // Close link panel
    closeLinkPanel() {
        const panel = document.getElementById('link-panel');
        if (panel) panel.classList.add('hidden');
    },

    // Execute site linking
    async executeLink() {
        const pathInput = document.getElementById('link-path');
        const executeBtn = document.getElementById('link-execute-btn');

        const path = pathInput ? pathInput.value.trim() : '';

        if (!path) {
            this.showToast('Please enter a path to your polis site', 'warning');
            return;
        }

        // Disable button while processing
        if (executeBtn) {
            executeBtn.disabled = true;
            executeBtn.textContent = 'Linking...';
        }

        try {
            const result = await this.api('POST', '/api/link', {
                path: path,
            });

            this.closeLinkPanel();
            this.showToast('Site linked successfully!', 'success');

            // Update display and show dashboard
            document.getElementById('domain-display').textContent =
                result.site_title || '';
            this.updateDomainDisplay(result.base_url);
            this.initNotifications();
            await this.loadAllCounts();
            await this.initViewMode();
            await this.loadViewContent();
            this.showScreen('dashboard');
        } catch (err) {
            this.showToast('Failed to link site: ' + err.message, 'error');
        } finally {
            if (executeBtn) {
                executeBtn.disabled = false;
                executeBtn.textContent = 'Link Site';
            }
        }
    },

    // New post action
    newPost() {
        this.currentDraftId = null;
        this.currentPostPath = null;
        this.currentFrontmatter = '';
        this.filenameManuallySet = false;
        document.getElementById('markdown-input').value = '';
        document.getElementById('filename-input').value = '';
        document.getElementById('filename-input').disabled = false;
        document.getElementById('preview-content').innerHTML =
            '<p class="empty-state">Start writing to see a preview.</p>';

        this.updateEditorFmToggle();
        this.updatePublishButton();
        this.showScreen('editor');
    },

    // New comment action
    newComment() {
        this.currentCommentDraftId = null;
        document.getElementById('reply-to-url').value = '';
        document.getElementById('comment-input').value = '';
        this.showScreen('comment');
    },

    // Alias for newComment (used by sidebar + button)
    newCommentDraft() {
        this.newComment();
    },

    // Render posts list
    async renderPostsList(container) {
        try {
            const result = await this.api('GET', '/api/posts');
            const posts = result.posts || [];
            this.counts.posts = posts.length;
            this.updateBadge('posts-count', posts.length);

            if (posts.length === 0) {
                container.innerHTML = `
                    <div class="content-list">
                        <div class="empty-state">
                            <h3>No published posts yet</h3>
                            <p>Start writing your first post</p>
                            <button class="primary" onclick="App.newPost()">New Post</button>
                        </div>
                    </div>
                `;
                return;
            }

            container.innerHTML = `
                <div class="content-list">
                    ${posts.map(post => `
                        <div class="content-item" data-path="${this.escapeHtml(post.path)}" onclick="App.openPost('${this.escapeHtml(post.path)}')">
                            <div class="item-info">
                                <div class="item-title">${this.escapeHtml(post.title)}</div>
                                <div class="item-path">${this.escapeHtml(post.path)}</div>
                            </div>
                            <div class="item-date-group">
                                <span class="item-date">${this.formatDate(post.published)}</span>
                                <span class="item-time">${this.formatTime(post.published)}</span>
                            </div>
                        </div>
                    `).join('')}
                </div>
            `;
        } catch (err) {
            container.innerHTML = `
                <div class="content-list">
                    <div class="empty-state">
                        <h3>Failed to load posts</h3>
                        <p>${this.escapeHtml(err.message)}</p>
                    </div>
                </div>
            `;
        }
    },

    // Render drafts list
    async renderDraftsList(container) {
        try {
            const result = await this.api('GET', '/api/drafts');
            const drafts = result.drafts || [];
            this.counts.drafts = drafts.length;
            this.updateBadge('drafts-count', drafts.length);

            if (drafts.length === 0) {
                container.innerHTML = `
                    <div class="content-list">
                        <div class="empty-state">
                            <h3>No drafts yet</h3>
                            <p>Start writing your first post</p>
                            <button class="primary" onclick="App.newPost()">New Post</button>
                        </div>
                    </div>
                `;
                return;
            }

            container.innerHTML = `
                <div class="content-list">
                    ${drafts.map(draft => `
                        <div class="content-item" onclick="App.openDraft('${this.escapeHtml(draft.id)}')">
                            <div class="item-info">
                                <div class="item-title">${this.escapeHtml(draft.id)}</div>
                                <div class="item-path">drafts/${this.escapeHtml(draft.id)}.md</div>
                            </div>
                            <div class="item-date-group">
                                <span class="item-date">${this.formatDate(draft.modified)}</span>
                                <span class="item-time">${this.formatTime(draft.modified)}</span>
                            </div>
                        </div>
                    `).join('')}
                </div>
            `;
        } catch (err) {
            container.innerHTML = `
                <div class="content-list">
                    <div class="empty-state">
                        <h3>Failed to load drafts</h3>
                        <p>${this.escapeHtml(err.message)}</p>
                    </div>
                </div>
            `;
        }
    },

    // Render MY comments list by status (outgoing - I wrote these)
    async renderMyCommentsList(container, status) {
        try {
            const result = await this.api('GET', `/api/comments/${status}`);
            const comments = result.comments || [];

            // Update count
            const countKey = status === 'pending' ? 'myPending' : status === 'blessed' ? 'myBlessed' : 'myDenied';
            this.counts[countKey] = comments.length;
            const badgeId = status === 'pending' ? 'my-pending-count' : status === 'blessed' ? 'my-blessed-count' : 'my-denied-count';
            this.updateBadge(badgeId, comments.length, status === 'pending');

            if (comments.length === 0) {
                const messages = {
                    pending: { title: 'No pending comments', desc: 'Comments you sent awaiting blessing will appear here' },
                    blessed: { title: 'No blessed comments', desc: 'Your approved comments will appear here' },
                    denied: { title: 'No denied comments', desc: 'Your denied comments will appear here' },
                };
                const msg = messages[status] || { title: 'No comments', desc: '' };

                container.innerHTML = `
                    <div class="content-list">
                        <div class="empty-state">
                            <h3>${msg.title}</h3>
                            <p>${msg.desc}</p>
                            <button class="primary" onclick="App.newComment()">New Comment</button>
                        </div>
                    </div>
                `;
                return;
            }

            container.innerHTML = `
                <div class="content-list">
                    ${comments.map(c => `
                        <div class="content-item" data-id="${this.escapeHtml(c.id)}" onclick="App.viewMyComment('${this.escapeHtml(c.id)}', '${status}')">
                            <div class="item-info">
                                <div class="item-title">${this.escapeHtml(c.title || c.id)}</div>
                                <div class="item-path">Re: ${this.escapeHtml(this.truncateUrl(c.in_reply_to))}</div>
                            </div>
                            <div class="item-meta-right">
                                <div class="item-date-group">
                                    <span class="item-date">${this.formatDate(c.timestamp)}</span>
                                    <span class="item-time">${this.formatTime(c.timestamp)}</span>
                                </div>
                                <span class="comment-status-badge ${status}">${status}</span>
                            </div>
                        </div>
                    `).join('')}
                </div>
            `;
        } catch (err) {
            container.innerHTML = `
                <div class="content-list">
                    <div class="empty-state">
                        <h3>Failed to load comments</h3>
                        <p>${this.escapeHtml(err.message)}</p>
                    </div>
                </div>
            `;
        }
    },

    // View a comment (my outgoing comments)
    async viewMyComment(id, status) {
        try {
            const result = await this.api('GET', `/api/comments/${status}/${encodeURIComponent(id)}`);
            this.showCommentDetail(result.comment, status);
        } catch (err) {
            this.showToast('Failed to load comment: ' + err.message, 'error');
        }
    },

    // Show comment detail modal/view
    showCommentDetail(comment, status) {
        const modal = document.createElement('div');
        modal.className = 'modal-overlay';
        modal.innerHTML = `
            <div class="modal comment-detail-modal">
                <div class="modal-header">
                    <h3>${this.escapeHtml(comment.title || comment.id)}</h3>
                    <button class="modal-close" onclick="this.closest('.modal-overlay').remove()">&times;</button>
                </div>
                <div class="modal-body">
                    <div class="comment-detail-meta">
                        <div class="meta-row">
                            <span class="meta-label">Status:</span>
                            <span class="comment-status-badge ${status}">${status}</span>
                        </div>
                        <div class="meta-row">
                            <span class="meta-label">In Reply To:</span>
                            <a href="${this.escapeHtml(comment.in_reply_to)}" target="_blank">${this.escapeHtml(comment.in_reply_to)}</a>
                        </div>
                        <div class="meta-row">
                            <span class="meta-label">Timestamp:</span>
                            <span>${this.formatDate(comment.timestamp)}</span>
                        </div>
                        ${comment.comment_url ? `
                        <div class="meta-row">
                            <span class="meta-label">Comment URL:</span>
                            <a href="${this.escapeHtml(comment.comment_url)}" target="_blank">${this.escapeHtml(comment.comment_url)}</a>
                        </div>
                        ` : ''}
                    </div>
                    <div class="comment-detail-content">
                        ${this.escapeHtml(comment.content || '(No content preview)')}
                    </div>
                </div>
            </div>
        `;
        document.body.appendChild(modal);
        modal.addEventListener('click', (e) => {
            if (e.target === modal) modal.remove();
        });
    },

    // Render MY comment drafts list
    async renderMyCommentDraftsList(container) {
        try {
            const result = await this.api('GET', '/api/comments/drafts');
            const drafts = result.drafts || [];
            this.counts.myCommentDrafts = drafts.length;
            this.updateBadge('my-comment-drafts-count', drafts.length);

            if (drafts.length === 0) {
                container.innerHTML = `
                    <div class="content-list">
                        <div class="empty-state">
                            <h3>No comment drafts</h3>
                            <p>Start writing a comment to reply to a post</p>
                            <button class="primary" onclick="App.newComment()">New Comment</button>
                        </div>
                    </div>
                `;
                return;
            }

            container.innerHTML = `
                <div class="content-list">
                    ${drafts.map(d => `
                        <div class="comment-item" onclick="App.openCommentDraft('${this.escapeHtml(d.id)}')">
                            <div class="comment-header">
                                <div class="comment-target">
                                    Re: <a href="${this.escapeHtml(d.in_reply_to)}" target="_blank">${this.escapeHtml(this.truncateUrl(d.in_reply_to))}</a>
                                </div>
                                <span class="comment-status draft">draft</span>
                            </div>
                            <div class="comment-preview">${this.escapeHtml(d.content ? d.content.substring(0, 100) : d.id)}</div>
                            <div class="comment-meta">
                                <span>${this.formatDate(d.updated_at || d.created_at)}</span>
                            </div>
                        </div>
                    `).join('')}
                </div>
            `;
        } catch (err) {
            container.innerHTML = `
                <div class="content-list">
                    <div class="empty-state">
                        <h3>Failed to load drafts</h3>
                        <p>${this.escapeHtml(err.message)}</p>
                    </div>
                </div>
            `;
        }
    },

    // Render INCOMING pending blessing requests (on my posts - others wrote these)
    async renderIncomingPendingList(container) {
        try {
            const result = await this.api('GET', '/api/blessing/requests');
            const requests = result.requests || [];
            this.counts.incomingPending = requests.length;
            this.updateBadge('incoming-pending-count', requests.length, true);

            if (requests.length === 0) {
                container.innerHTML = `
                    <div class="content-list">
                        <div class="empty-state">
                            <h3>No pending blessing requests</h3>
                            <p>When someone comments on your posts, their blessing requests will appear here</p>
                        </div>
                    </div>
                `;
                return;
            }

            container.innerHTML = `
                <div class="content-list">
                    ${requests.map(r => `
                        <div class="comment-item incoming-request" data-version="${this.escapeHtml(r.comment_version)}" onclick="App.openPendingRequestDetail(${this.escapeHtml(JSON.stringify(JSON.stringify(r)))})">
                            <div class="comment-header">
                                <span class="comment-author">${this.escapeHtml(r.author)}</span>
                                <span class="comment-date">${this.formatDate(r.timestamp)}</span>
                            </div>
                            <div class="comment-target">
                                On: <a href="${this.escapeHtml(r.in_reply_to)}" target="_blank" onclick="event.stopPropagation()">${this.escapeHtml(this.truncateUrl(r.in_reply_to))}</a>
                            </div>
                            <div class="comment-url">
                                <a href="${this.escapeHtml(r.comment_url)}" target="_blank" onclick="event.stopPropagation()">View comment</a>
                            </div>
                            <div class="comment-actions">
                                <button class="primary" onclick="event.stopPropagation(); App.grantBlessing('${this.escapeHtml(r.comment_version)}', '${this.escapeHtml(r.comment_url)}', '${this.escapeHtml(r.in_reply_to)}')">Bless</button>
                                <button class="secondary danger" onclick="event.stopPropagation(); App.denyBlessing('${this.escapeHtml(r.comment_url)}', '${this.escapeHtml(r.in_reply_to)}')">Deny</button>
                            </div>
                        </div>
                    `).join('')}
                </div>
            `;
        } catch (err) {
            container.innerHTML = `
                <div class="content-list">
                    <div class="empty-state">
                        <h3>Failed to load requests</h3>
                        <p>${this.escapeHtml(err.message)}</p>
                    </div>
                </div>
            `;
        }
    },

    // Open pending request detail panel
    openPendingRequestDetail(requestJson) {
        const request = JSON.parse(requestJson);
        const panel = document.getElementById('comment-detail-panel');
        const body = document.getElementById('comment-detail-body');
        const footer = document.getElementById('comment-detail-footer');
        const title = document.getElementById('comment-detail-title');

        title.textContent = 'Blessing Request';

        body.innerHTML = `
            <div class="comment-detail-meta">
                <div class="comment-detail-row">
                    <span class="comment-detail-label">From:</span>
                    <span class="comment-detail-value">${this.escapeHtml(request.author)}</span>
                </div>
                <div class="comment-detail-row">
                    <span class="comment-detail-label">On post:</span>
                    <span class="comment-detail-value"><a href="${this.escapeHtml(request.in_reply_to)}" target="_blank">${this.escapeHtml(request.in_reply_to)}</a></span>
                </div>
                <div class="comment-detail-row">
                    <span class="comment-detail-label">Comment:</span>
                    <span class="comment-detail-value"><a href="${this.escapeHtml(request.comment_url)}" target="_blank">${this.escapeHtml(request.comment_url)}</a></span>
                </div>
                <div class="comment-detail-row">
                    <span class="comment-detail-label">Submitted:</span>
                    <span class="comment-detail-value">${this.formatDate(request.timestamp)}</span>
                </div>
            </div>
            <div class="comment-detail-preview">
                <div class="comment-detail-preview-label">View Comment</div>
                <div class="comment-detail-preview-content">
                    <a href="${this.escapeHtml(request.comment_url)}" target="_blank">Open comment in new tab &rarr;</a>
                </div>
            </div>
        `;

        footer.innerHTML = `
            <button class="primary" onclick="App.grantBlessing('${this.escapeHtml(request.comment_version)}', '${this.escapeHtml(request.comment_url)}', '${this.escapeHtml(request.in_reply_to)}'); App.closeCommentDetail();">Bless</button>
            <button class="secondary danger" onclick="App.denyBlessing('${this.escapeHtml(request.comment_url)}', '${this.escapeHtml(request.in_reply_to)}'); App.closeCommentDetail();">Deny</button>
        `;

        panel.classList.remove('hidden');
        this.bindCommentDetailEvents();
    },

    // Render INCOMING blessed comments (on my posts)
    async renderIncomingBlessedList(container) {
        try {
            const result = await this.api('GET', '/api/blessed-comments');
            const postComments = result.comments || [];

            // Flatten for display
            let allBlessed = [];
            for (const pc of postComments) {
                for (const c of (pc.blessed || [])) {
                    allBlessed.push({
                        ...c,
                        post: pc.post
                    });
                }
            }

            this.counts.incomingBlessed = allBlessed.length;
            this.updateBadge('incoming-blessed-count', allBlessed.length);

            if (allBlessed.length === 0) {
                container.innerHTML = `
                    <div class="content-list">
                        <div class="empty-state">
                            <h3>No blessed comments yet</h3>
                            <p>Comments you approve will appear here</p>
                        </div>
                    </div>
                `;
                return;
            }

            container.innerHTML = `
                <div class="content-list">
                    ${allBlessed.map(c => `
                        <div class="content-item" onclick="App.openBlessedCommentDetail(${this.escapeHtml(JSON.stringify(JSON.stringify(c)))})">
                            <div class="item-info">
                                <div class="item-title">${this.escapeHtml(this.truncateUrl(c.url))}</div>
                                <div class="item-path">On: ${this.escapeHtml(c.post)}</div>
                            </div>
                            <div class="item-meta-right">
                                <span class="item-date">${this.formatDate(c.blessed_at)}</span>
                                <span class="comment-status-badge blessed">blessed</span>
                            </div>
                        </div>
                    `).join('')}
                </div>
            `;
        } catch (err) {
            container.innerHTML = `
                <div class="content-list">
                    <div class="empty-state">
                        <h3>Failed to load blessed comments</h3>
                        <p>${this.escapeHtml(err.message)}</p>
                    </div>
                </div>
            `;
        }
    },

    // Open blessed comment detail panel
    openBlessedCommentDetail(commentJson) {
        const comment = JSON.parse(commentJson);
        const panel = document.getElementById('comment-detail-panel');
        const body = document.getElementById('comment-detail-body');
        const footer = document.getElementById('comment-detail-footer');
        const title = document.getElementById('comment-detail-title');

        title.textContent = 'Blessed Comment';

        body.innerHTML = `
            <div class="comment-detail-meta">
                <div class="comment-detail-row">
                    <span class="comment-detail-label">On post:</span>
                    <span class="comment-detail-value">${this.escapeHtml(comment.post)}</span>
                </div>
                <div class="comment-detail-row">
                    <span class="comment-detail-label">Comment:</span>
                    <span class="comment-detail-value"><a href="${this.escapeHtml(comment.url)}" target="_blank">${this.escapeHtml(comment.url)}</a></span>
                </div>
                <div class="comment-detail-row">
                    <span class="comment-detail-label">Version:</span>
                    <span class="comment-detail-value" style="font-family: var(--font-mono); font-size: 0.8rem;">${this.escapeHtml(comment.version)}</span>
                </div>
                <div class="comment-detail-row">
                    <span class="comment-detail-label">Blessed:</span>
                    <span class="comment-detail-value">${this.formatDate(comment.blessed_at)}</span>
                </div>
            </div>
            <div class="comment-detail-preview">
                <div class="comment-detail-preview-label">View Comment</div>
                <div class="comment-detail-preview-content">
                    <a href="${this.escapeHtml(comment.url)}" target="_blank">Open comment in new tab &rarr;</a>
                </div>
            </div>
        `;

        footer.innerHTML = `
            <button class="secondary danger" onclick="App.revokeBlessing('${this.escapeHtml(comment.url)}'); App.closeCommentDetail();">Revoke Blessing</button>
            <button class="secondary" onclick="App.closeCommentDetail()">Close</button>
        `;

        panel.classList.remove('hidden');
        this.bindCommentDetailEvents();
    },

    // Close comment detail panel
    closeCommentDetail() {
        const panel = document.getElementById('comment-detail-panel');
        panel.classList.add('hidden');
    },

    // Bind comment detail panel events
    bindCommentDetailEvents() {
        const closeBtn = document.getElementById('comment-detail-close-btn');
        const overlay = document.querySelector('.comment-detail-overlay');

        closeBtn.onclick = () => this.closeCommentDetail();
        overlay.onclick = () => this.closeCommentDetail();
    },

    // Revoke a blessing (remove from blessed-comments.json)
    async revokeBlessing(commentUrl) {
        const confirmed = await this.showConfirmModal('Revoke Blessing', 'Revoke this blessing? The comment will be removed from your blessed comments index.', 'Revoke', 'Cancel', 'danger');
        if (!confirmed) return;

        try {
            await this.api('POST', '/api/blessing/revoke', {
                comment_url: commentUrl
            });

            this.showToast('Blessing revoked', 'success');
            await this.loadAllCounts();
            await this.loadViewContent();
        } catch (err) {
            this.showToast('Failed to revoke: ' + err.message, 'error');
        }
    },

    // Render settings page
    async renderSettings(container) {
        try {
            const settings = await this.api('GET', '/api/settings');
            const site = settings.site || {};
            const automations = settings.automations || [];

            // Store existing hooks for advanced panel
            this.existingHooks = settings.existing_hooks || [];

            let automationsHtml = '';
            if (automations.length === 0) {
                automationsHtml = `
                    <div class="empty-state" style="padding: 1.5rem;">
                        <p style="color: var(--text-muted);">No automations configured yet.</p>
                    </div>
                `;
            } else {
                automationsHtml = automations.map(a => `
                    <div class="automation-item">
                        <div class="automation-header">
                            <div class="automation-name">
                                <span class="status-icon">&#10003;</span>
                                ${this.escapeHtml(a.name)}
                            </div>
                            <div class="automation-actions">
                                <button onclick="App.deleteAutomation('${this.escapeHtml(a.id)}')" class="danger">Remove</button>
                            </div>
                        </div>
                        <div class="automation-description">${this.escapeHtml(a.description)}</div>
                    </div>
                `).join('');
            }

            const discoveryStatus = site.discovery_configured
                ? `<span style="color: var(--success-color);">Connected</span>`
                : `<span style="color: var(--warning-color);">Not configured</span>`;
            const discoveryUrl = site.discovery_url || 'Not set';

            container.innerHTML = `
                <div class="settings-container">
                    <div class="settings-section">
                        <div class="settings-section-label">Your Site</div>
                        <div class="settings-card">
                            <div class="settings-row">
                                <span class="settings-row-label">Site:</span>
                                <span class="settings-row-value">${this.escapeHtml(site.site_title || 'Not configured')}</span>
                            </div>
                            <div class="settings-row">
                                <span class="settings-row-label">Public Key:</span>
                                <span class="settings-row-value" id="public-key-display">${this.escapeHtml(this.truncateKey(site.public_key))}</span>
                                <div class="settings-row-actions">
                                    <button class="btn-copy" onclick="App.copyPublicKey('${this.escapeHtml(site.public_key || '')}')">Copy</button>
                                </div>
                            </div>
                        </div>
                    </div>

                    <div class="settings-section">
                        <div class="settings-section-label">Discovery Service</div>
                        <div class="settings-card">
                            <div class="settings-row">
                                <span class="settings-row-label">Status:</span>
                                <span class="settings-row-value">${discoveryStatus}</span>
                            </div>
                            <div class="settings-row">
                                <span class="settings-row-label">URL:</span>
                                <span class="settings-row-value" style="font-size: 0.85em; word-break: break-all;">${this.escapeHtml(discoveryUrl)}</span>
                            </div>
                            <div class="settings-row">
                                <span class="settings-row-label">Registration:</span>
                                <span class="settings-row-value" id="registration-status">Checking...</span>
                            </div>
                            <div id="registration-action" class="settings-action-row"></div>
                            ${!site.discovery_configured ? `
                            <div class="settings-row" style="margin-top: 0.5rem;">
                                <span class="settings-row-value" style="font-size: 0.85em; color: var(--text-muted);">
                                    Set DISCOVERY_SERVICE_URL and DISCOVERY_SERVICE_KEY in your .env file, then restart the webapp.
                                </span>
                            </div>
                            ` : ''}
                        </div>
                    </div>

                    <div class="settings-section">
                        <div class="settings-section-label">Help me...</div>
                        <div class="settings-card">
                            <div class="task-list">
                                <div class="task-item" onclick="App.openWizard('deployment')">
                                    Deploy my content using git
                                    <span class="task-item-arrow">&rarr;</span>
                                </div>
                                <div class="task-item" onclick="App.openWizard('custom')">
                                    Run a custom script when I post or comment
                                    <span class="task-item-arrow">&rarr;</span>
                                </div>
                            </div>
                        </div>
                    </div>

                    <div class="settings-section">
                        <div class="settings-section-label">Active Automations</div>
                        <div class="settings-card">
                            ${automationsHtml}
                        </div>
                    </div>
                </div>
            `;

            // Fetch registration status after rendering
            if (site.discovery_configured) {
                this.fetchRegistrationStatus();
            } else {
                const statusEl = document.getElementById('registration-status');
                if (statusEl) {
                    statusEl.innerHTML = `<span style="color: var(--text-muted);">Not configured</span>`;
                }
            }
        } catch (err) {
            container.innerHTML = `
                <div class="content-list">
                    <div class="empty-state">
                        <h3>Failed to load settings</h3>
                        <p>${this.escapeHtml(err.message)}</p>
                    </div>
                </div>
            `;
        }
    },

    // Truncate public key for display
    truncateKey(key) {
        if (!key) return 'Not generated';
        if (key.length <= 50) return key;
        return key.substring(0, 30) + '...' + key.substring(key.length - 15);
    },

    // Copy public key to clipboard
    async copyPublicKey(key) {
        if (!key) {
            this.showToast('No public key to copy', 'warning');
            return;
        }
        try {
            await navigator.clipboard.writeText(key);
            this.showToast('Public key copied to clipboard', 'success');
            // Update button temporarily
            const btn = document.querySelector('.btn-copy');
            if (btn) {
                btn.classList.add('copied');
                btn.textContent = 'Copied!';
                setTimeout(() => {
                    btn.classList.remove('copied');
                    btn.textContent = 'Copy';
                }, 2000);
            }
        } catch (err) {
            this.showToast('Failed to copy: ' + err.message, 'error');
        }
    },

    // Delete an automation
    async deleteAutomation(id) {
        const confirmed = await this.showConfirmModal('Remove Automation', 'Remove this automation? The hook will no longer run.', 'Remove', 'Cancel', 'danger');
        if (!confirmed) return;
        try {
            await this.api('DELETE', `/api/automations/${encodeURIComponent(id)}`);
            this.showToast('Automation removed', 'success');
            await this.loadViewContent();
        } catch (err) {
            this.showToast('Failed to remove: ' + err.message, 'error');
        }
    },

    // Fetch site registration status from discovery service
    async fetchRegistrationStatus() {
        const statusEl = document.getElementById('registration-status');
        const actionEl = document.getElementById('registration-action');

        if (!statusEl) return;

        try {
            const result = await this.api('GET', '/api/site/registration-status');

            if (!result.configured) {
                statusEl.innerHTML = `<span style="color: var(--text-muted);">Not configured</span>`;
                if (actionEl) actionEl.innerHTML = '';
                return;
            }

            if (result.error) {
                statusEl.innerHTML = `<span style="color: var(--warning-color);">Unable to check</span>`;
                if (actionEl) {
                    actionEl.innerHTML = `<span style="font-size: 0.85em; color: var(--text-muted);">${this.escapeHtml(result.error)}</span>`;
                }
                return;
            }

            if (result.is_registered) {
                // Format the date nicely
                let dateStr = '';
                if (result.registered_at) {
                    try {
                        const date = new Date(result.registered_at);
                        dateStr = ` (since ${date.toLocaleDateString()})`;
                    } catch (e) {
                        dateStr = '';
                    }
                }
                statusEl.innerHTML = `<span style="color: var(--success-color);">Registered${dateStr}</span>`;
                if (actionEl) {
                    actionEl.innerHTML = `
                        <a class="settings-action-link" onclick="App.openUnregisterPanel()">Unregister from discovery service</a>
                    `;
                }
            } else {
                statusEl.innerHTML = `<span style="color: var(--text-muted);">Not registered</span>`;
                if (actionEl) {
                    actionEl.innerHTML = `
                        <a class="settings-action-link" onclick="App.openRegisterPanel()">Register with discovery service</a>
                    `;
                }
            }
        } catch (err) {
            statusEl.innerHTML = `<span style="color: var(--warning-color);">Unable to check</span>`;
            if (actionEl) {
                actionEl.innerHTML = `<span style="font-size: 0.85em; color: var(--text-muted);">${this.escapeHtml(err.message)}</span>`;
            }
        }
    },

    // Open registration panel
    openRegisterPanel() {
        const panel = document.getElementById('registration-panel');
        const titleEl = document.getElementById('registration-panel-title');
        const bodyEl = document.getElementById('registration-panel-body');
        const footerEl = document.getElementById('registration-panel-footer');

        titleEl.textContent = 'Register Your Site';

        bodyEl.innerHTML = `
            <div class="wizard-section">
                <p>Registering makes your site discoverable to other authors in the polis network.</p>
                <p style="margin-top: 1rem;">This is <strong>not</strong> a username/password account. Registration simply:</p>
                <ul class="wizard-checklist" style="margin-top: 0.5rem;">
                    <li>Lists your site in the public directory</li>
                    <li>Allows others to find and follow your content</li>
                    <li>Enables you to receive and respond to comments</li>
                    <li>Lets you participate in conversations across the network</li>
                </ul>
                <p style="margin-top: 1rem; color: var(--text-muted);">
                    You can unregister at any time. Your content stays on your server - only the directory listing is affected.
                </p>
            </div>
        `;

        footerEl.innerHTML = `
            <button class="secondary" onclick="App.closeRegistrationPanel()">Cancel</button>
            <div class="wizard-footer-spacer"></div>
            <button id="register-btn" class="primary" onclick="App.registerSite()">Register</button>
        `;

        panel.classList.remove('hidden');
        this.bindRegistrationPanelEvents();
    },

    // Open unregistration panel
    openUnregisterPanel() {
        const panel = document.getElementById('registration-panel');
        const titleEl = document.getElementById('registration-panel-title');
        const bodyEl = document.getElementById('registration-panel-body');
        const footerEl = document.getElementById('registration-panel-footer');

        titleEl.textContent = 'Unregister Your Site';

        bodyEl.innerHTML = `
            <div class="wizard-section">
                <p>Are you sure you want to unregister your site?</p>
                <p style="margin-top: 1rem;">Unregistering will:</p>
                <ul class="wizard-checklist" style="margin-top: 0.5rem;">
                    <li>Remove your site from the public directory</li>
                    <li>Prevent others from discovering you through the network</li>
                    <li>Stop new blessing requests from being delivered</li>
                </ul>
                <p style="margin-top: 1rem; font-weight: 500;">
                    Note: This does not delete any content or links that others have already made to your posts.
                    Existing blessed comments and references will remain intact.
                </p>
                <p style="margin-top: 1rem; color: var(--text-muted);">
                    You can re-register anytime to rejoin the community.
                </p>
            </div>
        `;

        footerEl.innerHTML = `
            <button class="secondary" onclick="App.closeRegistrationPanel()">Cancel</button>
            <div class="wizard-footer-spacer"></div>
            <button id="unregister-btn" class="danger" onclick="App.unregisterSite()">Unregister</button>
        `;

        panel.classList.remove('hidden');
        this.bindRegistrationPanelEvents();
    },

    // Close registration panel
    closeRegistrationPanel() {
        const panel = document.getElementById('registration-panel');
        panel.classList.add('hidden');
    },

    // Bind registration panel events
    bindRegistrationPanelEvents() {
        const closeBtn = document.getElementById('registration-close-btn');
        const overlay = document.querySelector('#registration-panel .wizard-overlay');

        if (closeBtn) closeBtn.onclick = () => this.closeRegistrationPanel();
        if (overlay) overlay.onclick = () => this.closeRegistrationPanel();
    },

    // Register site with discovery service
    async registerSite() {
        const btn = document.getElementById('register-btn');
        if (btn) {
            btn.disabled = true;
            btn.textContent = 'Registering...';
        }

        try {
            await this.api('POST', '/api/site/register');
            this.showToast('Site registered successfully!', 'success');
            this.closeRegistrationPanel();
            // Refresh the settings to show updated status
            await this.loadViewContent();
        } catch (err) {
            this.showToast('Registration failed: ' + err.message, 'error');
            if (btn) {
                btn.disabled = false;
                btn.textContent = 'Register';
            }
        }
    },

    // Unregister site from discovery service
    async unregisterSite() {
        const btn = document.getElementById('unregister-btn');
        if (btn) {
            btn.disabled = true;
            btn.textContent = 'Unregistering...';
        }

        try {
            await this.api('POST', '/api/site/unregister');
            this.showToast('Site unregistered successfully', 'success');
            this.closeRegistrationPanel();
            // Refresh the settings to show updated status
            await this.loadViewContent();
        } catch (err) {
            this.showToast('Unregistration failed: ' + err.message, 'error');
            if (btn) {
                btn.disabled = false;
                btn.textContent = 'Unregister';
            }
        }
    },

    // Wizard state
    wizard: {
        templateId: null,
        step: 1,
        totalSteps: 4,
        deploymentType: null, // 'vercel', 'github-pages', or 'git-commit'
        selectedHookTypes: [], // ['post-publish', 'post-republish', 'post-comment']
    },

    // Store existing hooks from settings
    existingHooks: [],

    // Open wizard panel
    openWizard(templateId) {
        this.wizard.templateId = templateId;
        this.wizard.step = 1;
        this.wizard.totalSteps = templateId === 'deployment' ? 4 : 3;
        this.wizard.deploymentType = null;
        this.wizard.selectedHookTypes = [];

        const panel = document.getElementById('wizard-panel');
        panel.classList.remove('hidden');

        this.renderWizardStep();
        this.bindWizardEvents();
    },

    // Close wizard panel
    closeWizard() {
        const panel = document.getElementById('wizard-panel');
        panel.classList.add('hidden');
        this.wizard.templateId = null;
        this.wizard.step = 1;
        this.wizard.deploymentType = null;
        this.wizard.selectedHookTypes = [];
    },

    // Bind wizard events
    bindWizardEvents() {
        const closeBtn = document.getElementById('wizard-close-btn');
        const overlay = document.querySelector('.wizard-overlay');
        const backBtn = document.getElementById('wizard-back-btn');
        const nextBtn = document.getElementById('wizard-next-btn');

        closeBtn.onclick = () => this.closeWizard();
        overlay.onclick = () => this.closeWizard();
        backBtn.onclick = () => this.wizardBack();
        nextBtn.onclick = () => this.wizardNext();
    },

    // Navigate wizard back
    wizardBack() {
        if (this.wizard.step > 1) {
            this.wizard.step--;
            this.renderWizardStep();
        }
    },

    // Navigate wizard forward or complete
    async wizardNext() {
        if (this.wizard.step < this.wizard.totalSteps) {
            if (this.wizard.templateId === 'deployment') {
                // Deployment wizard flow (4 steps)
                if (this.wizard.step === 1) {
                    // Step 1: Validate deployment method selection
                    const selected = document.querySelector('input[name="deployment-type"]:checked');
                    if (!selected) {
                        this.showToast('Please select a deployment method', 'warning');
                        return;
                    }
                    this.wizard.deploymentType = selected.value;
                    this.wizard.step++;
                    this.renderWizardStep();
                } else if (this.wizard.step === 2) {
                    // Step 2: Validate hook type selection
                    const selected = document.querySelectorAll('input[name="hook-type"]:checked');
                    this.wizard.selectedHookTypes = Array.from(selected).map(el => el.value);
                    if (this.wizard.selectedHookTypes.length === 0) {
                        this.showToast('Please select at least one hook type', 'warning');
                        return;
                    }
                    this.wizard.step++;
                    this.renderWizardStep();
                } else if (this.wizard.step === 3) {
                    // Step 3: Create the hooks
                    const nextBtn = document.getElementById('wizard-next-btn');
                    nextBtn.classList.add('btn-loading');
                    nextBtn.disabled = true;

                    try {
                        // Create hooks for each selected type
                        for (const hookType of this.wizard.selectedHookTypes) {
                            await this.api('POST', '/api/automations', {
                                template_id: this.wizard.deploymentType,
                                hook_type: hookType
                            });
                            this.existingHooks.push(hookType);
                        }
                        this.wizard.step++;
                        this.renderWizardStep();
                    } catch (err) {
                        this.showToast('Failed to create automation: ' + err.message, 'error');
                    } finally {
                        nextBtn.classList.remove('btn-loading');
                        nextBtn.disabled = false;
                    }
                }
            } else if (this.wizard.templateId === 'custom') {
                // Custom script wizard flow (3 steps)
                if (this.wizard.step === 1) {
                    // Step 1 -> 2: Just advance
                    this.wizard.step++;
                    this.renderWizardStep();
                } else if (this.wizard.step === 2) {
                    // Step 2: Validate hook type selection and create hooks
                    const selected = document.querySelectorAll('input[name="hook-type"]:checked');
                    this.wizard.selectedHookTypes = Array.from(selected).map(el => el.value);
                    if (this.wizard.selectedHookTypes.length === 0) {
                        this.showToast('Please select at least one hook type', 'warning');
                        return;
                    }

                    const nextBtn = document.getElementById('wizard-next-btn');
                    nextBtn.classList.add('btn-loading');
                    nextBtn.disabled = true;

                    try {
                        // Create hooks for each selected type
                        for (const hookType of this.wizard.selectedHookTypes) {
                            await this.api('POST', '/api/hooks/generate', { hook_type: hookType });
                            this.existingHooks.push(hookType);
                        }
                        this.wizard.step++;
                        this.renderWizardStep();
                    } catch (err) {
                        this.showToast('Failed to create hook: ' + err.message, 'error');
                    } finally {
                        nextBtn.classList.remove('btn-loading');
                        nextBtn.disabled = false;
                    }
                }
            } else {
                this.wizard.step++;
                this.renderWizardStep();
            }
        } else {
            // Complete: close wizard and refresh
            this.closeWizard();
            await this.loadViewContent();
        }
    },

    // Render current wizard step
    renderWizardStep() {
        const titleEl = document.getElementById('wizard-title');
        const currentEl = document.getElementById('wizard-step-current');
        const totalEl = document.getElementById('wizard-step-total');
        const bodyEl = document.getElementById('wizard-body');
        const backBtn = document.getElementById('wizard-back-btn');
        const nextBtn = document.getElementById('wizard-next-btn');

        currentEl.textContent = this.wizard.step;
        totalEl.textContent = this.wizard.totalSteps;

        // Show/hide back button
        backBtn.style.display = this.wizard.step > 1 ? 'inline-block' : 'none';

        // Update button text based on template and step
        if (this.wizard.templateId === 'deployment') {
            // 4-step deployment wizard
            if (this.wizard.step === 3) {
                nextBtn.textContent = 'Create scripts \u2192';
            } else if (this.wizard.step === 4) {
                nextBtn.textContent = 'Done \u2713';
            } else {
                nextBtn.textContent = 'Next \u2192';
            }
        } else if (this.wizard.templateId === 'custom') {
            // 3-step custom wizard
            if (this.wizard.step === 2) {
                nextBtn.textContent = 'Create scripts \u2192';
            } else if (this.wizard.step === 3) {
                nextBtn.textContent = 'Done \u2713';
            } else {
                nextBtn.textContent = 'Next \u2192';
            }
        } else {
            nextBtn.textContent = 'Next \u2192';
        }

        // Get wizard content based on template and step
        const content = this.getWizardContent(this.wizard.templateId, this.wizard.step);
        titleEl.textContent = content.title;
        bodyEl.innerHTML = content.body;
    },

    // Generate hook type checkboxes HTML
    getHookTypeCheckboxes() {
        const hookTypes = [
            { id: 'post-publish', name: 'post-publish', desc: 'Runs after a new post is published' },
            { id: 'post-republish', name: 'post-republish', desc: 'Runs after an existing post is updated' },
            { id: 'post-comment', name: 'post-comment', desc: 'Runs after you bless a comment on your site' },
        ];

        return hookTypes.map(hook => {
            const exists = this.existingHooks.includes(hook.id);
            const disabled = exists ? 'disabled' : '';
            const checked = !exists ? 'checked' : '';
            const existsLabel = exists ? '<span class="hook-exists-inline">(already exists)</span>' : '';
            return `
                <label class="hook-type-checkbox ${exists ? 'disabled' : ''}">
                    <input type="checkbox" name="hook-type" value="${hook.id}" ${disabled} ${checked}>
                    <div class="hook-type-checkbox-content">
                        <div class="hook-type-checkbox-name">${hook.name} ${existsLabel}</div>
                        <div class="hook-type-checkbox-desc">${hook.desc}</div>
                    </div>
                </label>
            `;
        }).join('');
    },

    // Get script preview for deployment type
    getDeploymentScriptPreview() {
        const scripts = {
            'vercel': `#!/bin/bash
set -e
cd "$POLIS_SITE_DIR"
git add -A
git commit -m "$POLIS_COMMIT_MESSAGE"
git push`,
            'github-pages': `#!/bin/bash
set -e
cd "$POLIS_SITE_DIR"
git add -A
git commit -m "$POLIS_COMMIT_MESSAGE"
git push`,
            'git-commit': `#!/bin/bash
set -e
cd "$POLIS_SITE_DIR"
git add -A
git commit -m "$POLIS_COMMIT_MESSAGE"`
        };
        return scripts[this.wizard.deploymentType] || scripts['git-commit'];
    },

    // Get wizard content for a template and step
    getWizardContent(templateId, step) {
        // Deployment wizard (4 steps)
        if (templateId === 'deployment') {
            if (step === 1) {
                return {
                    title: 'Deploy my content using git',
                    body: `
                        <div class="wizard-section">
                            <p>Which deployment method would you like to use?</p>
                            <div class="deployment-options">
                                <label class="deployment-option">
                                    <input type="radio" name="deployment-type" value="vercel">
                                    <div class="deployment-option-content">
                                        <div class="deployment-option-title">Vercel</div>
                                        <div class="deployment-option-desc">Commit, push, and let Vercel auto-deploy</div>
                                    </div>
                                </label>
                                <label class="deployment-option">
                                    <input type="radio" name="deployment-type" value="github-pages">
                                    <div class="deployment-option-content">
                                        <div class="deployment-option-title">GitHub Pages</div>
                                        <div class="deployment-option-desc">Commit, push, and let GitHub Pages rebuild</div>
                                    </div>
                                </label>
                                <label class="deployment-option">
                                    <input type="radio" name="deployment-type" value="git-commit">
                                    <div class="deployment-option-content">
                                        <div class="deployment-option-title">Git repository only</div>
                                        <div class="deployment-option-desc">Commit changes without pushing (manual deployment)</div>
                                    </div>
                                </label>
                            </div>
                        </div>
                    `
                };
            } else if (step === 2) {
                const methodNames = { 'vercel': 'Vercel', 'github-pages': 'GitHub Pages', 'git-commit': 'git' };
                const methodName = methodNames[this.wizard.deploymentType] || 'git';
                return {
                    title: 'Deploy my content using git',
                    body: `
                        <div class="wizard-section">
                            <p>Which events should trigger ${methodName} deployment?</p>
                            <div class="hook-type-checkboxes">
                                ${this.getHookTypeCheckboxes()}
                            </div>
                            <p class="wizard-hint">Scripts will be created at <code>.polis/hooks/{event}.sh</code></p>
                        </div>
                    `
                };
            } else if (step === 3) {
                const selectedHooks = this.wizard.selectedHookTypes.map(h => `<code>${h}</code>`).join(', ');
                return {
                    title: 'Deploy my content using git',
                    body: `
                        <div class="wizard-section">
                            <p>The following script will be created for: ${selectedHooks}</p>
                            <div class="wizard-code-block">
                                <code>${this.escapeHtml(this.getDeploymentScriptPreview())}</code>
                            </div>
                            <div class="wizard-prereqs">
                                <div class="wizard-prereqs-title">Prerequisites:</div>
                                <ul>
                                    <li>Your site directory is a git repository</li>
                                    <li>Git is configured with your name and email</li>
                                    ${this.wizard.deploymentType !== 'git-commit' ? '<li>Remote is configured (origin &rarr; your repo)</li>' : ''}
                                </ul>
                            </div>
                        </div>
                    `
                };
            } else if (step === 4) {
                const createdFiles = this.wizard.selectedHookTypes.map(h =>
                    `<div class="wizard-info-row"><span class="wizard-info-label">Created:</span><span class="wizard-info-value">.polis/hooks/${h}.sh</span></div>`
                ).join('');
                return {
                    title: 'Deploy my content using git',
                    body: `
                        <div class="wizard-section">
                            <div class="wizard-success">
                                <span class="wizard-success-icon">&#10003;</span>
                                Automation created
                            </div>
                            <div class="wizard-info-block">
                                ${createdFiles}
                            </div>
                            <div class="wizard-help-section">
                                <div class="wizard-help-title">To test it:</div>
                                <ul class="wizard-help-list">
                                    <li>Publish a post, update a post, or bless a comment</li>
                                    <li>Check that the corresponding hook ran successfully</li>
                                </ul>
                            </div>
                            <div class="wizard-help-section">
                                <div class="wizard-help-title">If something goes wrong:</div>
                                <ul class="wizard-help-list">
                                    <li>Check that <code>git</code> commands work from your terminal</li>
                                    <li>Edit the scripts at <code>.polis/hooks/*.sh</code></li>
                                </ul>
                            </div>
                        </div>
                    `
                };
            }
        }

        // Custom script wizard (3 steps)
        if (templateId === 'custom') {
            if (step === 1) {
                return {
                    title: 'Custom automation scripts',
                    body: `
                        <div class="wizard-section">
                            <p>Polis supports three hook types that run shell scripts when events occur:</p>
                            <div class="hook-types-explained">
                                <div class="hook-type-explained">
                                    <div class="hook-type-explained-name">post-publish</div>
                                    <div class="hook-type-explained-desc">Runs after you publish a <em>new</em> post. The post file and metadata have been written.</div>
                                </div>
                                <div class="hook-type-explained">
                                    <div class="hook-type-explained-name">post-republish</div>
                                    <div class="hook-type-explained-desc">Runs after you update an <em>existing</em> post. The updated file and metadata have been written.</div>
                                </div>
                                <div class="hook-type-explained">
                                    <div class="hook-type-explained-name">post-comment</div>
                                    <div class="hook-type-explained-desc">Runs after you bless a comment on your site. The comment file has been written to <code>comments/blessed/</code>.</div>
                                </div>
                            </div>
                            <p>Each hook receives environment variables you can use in your script:</p>
                            <div class="wizard-code-block">
                                <code>POLIS_SITE_DIR       # Path to your site directory
POLIS_PATH           # Relative path to the file
POLIS_TITLE          # Title of the post (or in_reply_to URL for comments)
POLIS_COMMIT_MESSAGE # Suggested commit message
POLIS_EVENT          # Event type (post-publish, post-republish, post-comment)
POLIS_VERSION        # Content hash
POLIS_TIMESTAMP      # ISO timestamp</code>
                            </div>
                        </div>
                    `
                };
            } else if (step === 2) {
                return {
                    title: 'Custom automation scripts',
                    body: `
                        <div class="wizard-section">
                            <p>Which hooks would you like to create?</p>
                            <div class="hook-type-checkboxes">
                                ${this.getHookTypeCheckboxes()}
                            </div>
                            <p>A starter script will be created that you can customize:</p>
                            <div class="wizard-code-block">
                                <code>#!/bin/bash
set -e
# Add your custom logic here
echo "Hook triggered: $POLIS_EVENT"
echo "File: $POLIS_PATH"</code>
                            </div>
                            <p class="wizard-hint">Scripts are saved to <code>.polis/hooks/{event}.sh</code></p>
                        </div>
                    `
                };
            } else if (step === 3) {
                const createdFiles = this.wizard.selectedHookTypes.map(h =>
                    `<div class="wizard-info-row"><span class="wizard-info-label">Created:</span><span class="wizard-info-value">.polis/hooks/${h}.sh</span></div>`
                ).join('');
                return {
                    title: 'Custom automation scripts',
                    body: `
                        <div class="wizard-section">
                            <div class="wizard-success">
                                <span class="wizard-success-icon">&#10003;</span>
                                Hook scripts created
                            </div>
                            <div class="wizard-info-block">
                                ${createdFiles}
                            </div>
                            <div class="wizard-help-section">
                                <div class="wizard-help-title">Next steps:</div>
                                <ul class="wizard-help-list">
                                    <li>Edit the scripts to add your custom logic</li>
                                    <li>Test by publishing a post, updating a post, or blessing a comment</li>
                                </ul>
                            </div>
                            <div class="wizard-help-section">
                                <div class="wizard-help-title">Troubleshooting:</div>
                                <ul class="wizard-help-list">
                                    <li>Scripts must be executable (<code>chmod +x</code>)</li>
                                    <li>Use <code>set -e</code> to stop on errors</li>
                                    <li>Check the webapp logs for hook output</li>
                                </ul>
                            </div>
                        </div>
                    `
                };
            }
        }

        // Fallback
        return { title: 'Setup', body: '<p>Unknown wizard step</p>'
        };
    },

    // Render markdown preview (always body-only, frontmatter shown separately)
    async renderPreview() {
        const body = document.getElementById('markdown-input').value;
        const previewContent = document.getElementById('preview-content');

        if (!body.trim()) {
            previewContent.innerHTML = '<p class="empty-state">Start writing to see a preview.</p>';
            return;
        }

        try {
            const result = await this.api('POST', '/api/render', { markdown: body });
            previewContent.innerHTML = result.html;
        } catch (err) {
            previewContent.innerHTML = `<p class="error">Render failed: ${this.escapeHtml(err.message)}</p>`;
        }
    },

    // Debounced live preview for editor (300ms, always body-only)
    editorUpdatePreview: (function() {
        let timeout = null;
        return function() {
            if (timeout) clearTimeout(timeout);
            timeout = setTimeout(async () => {
                const body = document.getElementById('markdown-input')?.value || '';
                const previewContent = document.getElementById('preview-content');
                if (!previewContent) return;

                if (!body.trim()) {
                    previewContent.innerHTML = '<p class="empty-state">Start writing to see a preview.</p>';
                    return;
                }

                try {
                    const result = await App.api('POST', '/api/render', { markdown: body });
                    previewContent.innerHTML = result.html;
                } catch (err) {
                    // Don't show errors during typing — leave last good preview
                }
            }, 300);
        };
    })(),

    // Save draft
    async saveDraft() {
        const markdown = document.getElementById('markdown-input').value;

        if (!markdown.trim()) {
            this.showToast('Nothing to save', 'warning');
            return;
        }

        // Extract title from first heading or first line
        let title = 'untitled';
        const lines = markdown.split('\n');
        for (const line of lines) {
            const trimmed = line.trim();
            if (trimmed) {
                const headingMatch = trimmed.match(/^#+\s+(.+)$/);
                if (headingMatch) {
                    title = headingMatch[1];
                } else {
                    title = trimmed.substring(0, 50);
                }
                break;
            }
        }

        const id = this.currentDraftId || this.slugify(title);

        try {
            const result = await this.api('POST', '/api/drafts', { id, markdown });
            this.currentDraftId = result.id;
            this.showToast('Draft saved', 'success');
        } catch (err) {
            this.showToast('Failed to save draft: ' + err.message, 'error');
        }
    },

    // Publish or republish post
    async publish() {
        const markdown = document.getElementById('markdown-input').value;

        if (!markdown.trim()) {
            this.showToast('Nothing to publish', 'warning');
            return;
        }

        const isRepublish = !!this.currentPostPath;
        const title = isRepublish ? 'Republish Post' : 'Publish Post';
        const message = isRepublish
            ? 'This post will be re-signed with an updated version.'
            : 'This post will be signed and saved to your posts directory.';
        const buttonText = isRepublish ? 'Republish' : 'Publish';

        const confirmed = await this.showConfirmModal(title, message, buttonText);
        if (!confirmed) {
            return;
        }

        const btn = document.getElementById('publish-btn');
        btn.classList.add('btn-loading');
        btn.disabled = true;

        try {
            let result;
            if (isRepublish) {
                result = await this.api('POST', '/api/republish', {
                    path: this.currentPostPath,
                    markdown
                });
            } else {
                // Use filename from input, fall back to auto-generated from title
                const filenameInput = document.getElementById('filename-input').value.trim();
                result = await this.api('POST', '/api/publish', {
                    markdown,
                    filename: filenameInput || ''
                });
            }

            if (result.success) {
                const action = isRepublish ? 'Republished' : 'Published';
                this.showToast(`${action}: ${result.title}`, 'success');

                // Clear editor and return to dashboard
                this.currentDraftId = null;
                this.currentPostPath = null;
                document.getElementById('markdown-input').value = '';
                document.getElementById('preview-content').innerHTML =
                    '<p class="empty-state">Start writing to see a preview.</p>';
        

                // Switch to Published view
                this.currentView = 'posts-published';
                await this.loadAllCounts();
                await this.loadViewContent();
                this.updatePublishButton();
                this.showScreen('dashboard');

                // Update sidebar active state
                document.querySelectorAll('.sidebar .nav-item').forEach(item => {
                    item.classList.remove('active');
                    if (item.dataset.view === 'posts-published') {
                        item.classList.add('active');
                    }
                });
            }
        } catch (err) {
            this.showToast('Failed to publish: ' + err.message, 'error');
        } finally {
            btn.classList.remove('btn-loading');
            btn.disabled = false;
        }
    },

    // Open a draft for editing
    async openDraft(id) {
        try {
            const result = await this.api('GET', `/api/drafts/${encodeURIComponent(id)}`);
            this.currentDraftId = id;
            this.currentPostPath = null;
            this.currentFrontmatter = '';
            this.filenameManuallySet = true;  // Draft already has a filename
            document.getElementById('markdown-input').value = result.markdown;
            document.getElementById('filename-input').value = id;  // Draft ID is the filename
            document.getElementById('filename-input').disabled = false;
    
            this.updateEditorFmToggle();
            this.updatePublishButton();
            this.showScreen('editor');
            this.editorUpdatePreview();
        } catch (err) {
            this.showToast('Failed to load draft: ' + err.message, 'error');
        }
    },

    // Open a published post for editing
    async openPost(path) {
        try {
            const result = await this.api('GET', `/api/posts/${encodeURIComponent(path)}`);
            this.currentDraftId = null;
            this.currentPostPath = path;
            // Store frontmatter separately — don't expose it in the textarea
            this.currentFrontmatter = '';
            if (result.raw_markdown) {
                const fmMatch = result.raw_markdown.match(/^---\r?\n[\s\S]*?\r?\n---\r?\n?/);
                if (fmMatch) this.currentFrontmatter = fmMatch[0];
            }
            document.getElementById('markdown-input').value = result.markdown;

            this.updateEditorFmToggle();
            this.updatePublishButton();
            this.showScreen('editor');
            this.editorUpdatePreview();
        } catch (err) {
            this.showToast('Failed to load post: ' + err.message, 'error');
        }
    },

    // Update publish button text and filename visibility based on current state
    updatePublishButton() {
        const btn = document.getElementById('publish-btn');
        const filenameContainer = document.getElementById('filename-container');
        const filenameInput = document.getElementById('filename-input');

        if (this.currentPostPath) {
            // Republishing - filename is locked
            btn.textContent = 'Republish';
            filenameContainer.style.display = 'none';
        } else {
            // New post - filename is editable
            btn.textContent = 'Publish';
            filenameContainer.style.display = 'flex';
            filenameInput.disabled = false;
        }
    },

    // Open a comment draft for editing
    async openCommentDraft(id) {
        try {
            const draft = await this.api('GET', `/api/comments/drafts/${encodeURIComponent(id)}`);
            this.currentCommentDraftId = id;
            document.getElementById('reply-to-url').value = draft.in_reply_to || '';
            document.getElementById('comment-input').value = draft.content || '';
            this.showScreen('comment');
        } catch (err) {
            this.showToast('Failed to load draft: ' + err.message, 'error');
        }
    },

    // Save comment draft
    async saveCommentDraft() {
        const inReplyTo = document.getElementById('reply-to-url').value.trim();
        const content = document.getElementById('comment-input').value;

        if (!inReplyTo) {
            this.showToast('Please enter the URL of the post you are replying to', 'warning');
            return;
        }

        try {
            const result = await this.api('POST', '/api/comments/drafts', {
                id: this.currentCommentDraftId || '',
                in_reply_to: inReplyTo,
                content: content
            });
            this.currentCommentDraftId = result.id;
            this.showToast('Comment draft saved', 'success');
        } catch (err) {
            this.showToast('Failed to save draft: ' + err.message, 'error');
        }
    },

    // Sign and send comment for blessing
    async signAndSendComment() {
        const inReplyTo = document.getElementById('reply-to-url').value.trim();
        const content = document.getElementById('comment-input').value;

        if (!inReplyTo) {
            this.showToast('Please enter the URL of the post you are replying to', 'warning');
            return;
        }

        if (!content.trim()) {
            this.showToast('Please write a comment', 'warning');
            return;
        }

        const confirmed = await this.showConfirmModal('Send for Blessing', 'Sign this comment and send it for blessing? The post author will need to approve it.', 'Sign & Send', 'Cancel');
        if (!confirmed) return;

        const btn = document.getElementById('sign-send-btn');
        btn.classList.add('btn-loading');
        btn.disabled = true;

        try {
            // First sign the comment
            const signResult = await this.api('POST', '/api/comments/sign', {
                draft_id: this.currentCommentDraftId || '',
                in_reply_to: inReplyTo,
                content: content
            });

            if (!signResult.success) {
                throw new Error('Failed to sign comment');
            }

            // Try to send for blessing
            try {
                const beseechResult = await this.api('POST', '/api/comments/beseech', {
                    comment_id: signResult.comment.id
                });

                if (beseechResult.status === 'blessed') {
                    this.showToast('Your comment was auto-blessed!', 'success');
                } else {
                    this.showToast('Comment signed and sent for blessing', 'success');
                }
            } catch (beseechErr) {
                this.showToast('Comment signed. Could not send blessing request: ' + beseechErr.message, 'warning', 6000);
            }

            // Clear form and return to dashboard
            this.currentCommentDraftId = null;
            document.getElementById('reply-to-url').value = '';
            document.getElementById('comment-input').value = '';

            // Switch to my comments pending view
            this.currentView = 'my-comments-pending';
            await this.loadAllCounts();
            await this.loadViewContent();
            this.showScreen('dashboard');

            // Update sidebar active state
            document.querySelectorAll('.sidebar .nav-item').forEach(item => {
                item.classList.remove('active');
                if (item.dataset.view === 'my-comments-pending') {
                    item.classList.add('active');
                }
            });
        } catch (err) {
            this.showToast('Failed to sign comment: ' + err.message, 'error');
        } finally {
            btn.classList.remove('btn-loading');
            btn.disabled = false;
        }
    },

    // Sync pending comments
    async syncComments() {
        const syncBtn = document.getElementById('sync-comments-btn');
        if (!syncBtn) return;

        syncBtn.classList.add('btn-loading');
        syncBtn.disabled = true;

        try {
            const result = await this.api('POST', '/api/comments/sync');

            let messages = [];
            if (result.blessed && result.blessed.length > 0) {
                messages.push(`${result.blessed.length} blessed`);
            }
            if (result.denied && result.denied.length > 0) {
                messages.push(`${result.denied.length} denied`);
            }
            if (result.still_pending && result.still_pending.length > 0) {
                messages.push(`${result.still_pending.length} still pending`);
            }

            const message = messages.length > 0 ? messages.join(', ') : 'No changes';
            this.showToast(`Sync complete: ${message}`, 'success');

            await this.loadAllCounts();
            await this.loadViewContent();
        } catch (err) {
            this.showToast('Sync failed: ' + err.message, 'error');
        } finally {
            syncBtn.classList.remove('btn-loading');
            syncBtn.disabled = false;
        }
    },

    // Grant blessing to an incoming comment request
    async grantBlessing(commentVersion, commentUrl, inReplyTo) {
        const confirmed = await this.showConfirmModal('Bless Comment', 'Bless this comment? It will be added to your blessed comments index.', 'Bless', 'Cancel');
        if (!confirmed) return;

        try {
            await this.api('POST', '/api/blessing/grant', {
                comment_version: commentVersion,
                comment_url: commentUrl,
                in_reply_to: inReplyTo
            });

            this.showToast('Comment blessed!', 'success');
            await this.loadAllCounts();
            await this.loadViewContent();
        } catch (err) {
            this.showToast('Failed to bless: ' + err.message, 'error');
        }
    },

    // Deny blessing to an incoming comment request
    async denyBlessing(commentURL, inReplyTo) {
        const confirmed = await this.showConfirmModal('Deny Blessing', 'Deny this blessing request? The commenter will be notified.', 'Deny', 'Cancel', 'danger');
        if (!confirmed) return;

        try {
            await this.api('POST', '/api/blessing/deny', {
                comment_url: commentURL,
                in_reply_to: inReplyTo
            });

            this.showToast('Blessing denied', 'success');
            await this.loadAllCounts();
            await this.loadViewContent();
        } catch (err) {
            this.showToast('Failed to deny: ' + err.message, 'error');
        }
    },

    // Snippet management methods

    // Render snippets list with directory navigation
    async renderSnippetsList(container) {
        try {
            const path = this.snippetState.currentPath || '';
            const filter = this.snippetState.filter || 'all';
            const result = await this.api('GET', `/api/snippets?path=${encodeURIComponent(path)}&filter=${filter}`);
            const entries = result.entries || [];
            this.snippetState.activeTheme = result.active_theme;

            // Build breadcrumb
            const breadcrumb = this.buildSnippetBreadcrumb(result.path, result.parent);

            if (entries.length === 0 && path === '') {
                container.innerHTML = `
                    <div class="content-list">
                        ${breadcrumb}
                        <div class="empty-state">
                            <h3>No snippets yet</h3>
                            <p>Snippets are reusable HTML/Markdown templates</p>
                            <button class="primary" onclick="App.newSnippet()">Create Snippet</button>
                        </div>
                    </div>
                `;
                return;
            }

            container.innerHTML = `
                <div class="content-list">
                    ${breadcrumb}
                    ${entries.map(entry => entry.is_dir
                        ? this.renderSnippetDirItem(entry)
                        : this.renderSnippetFileItem(entry)
                    ).join('')}
                </div>
            `;
        } catch (err) {
            container.innerHTML = `
                <div class="content-list">
                    <div class="empty-state">
                        <h3>Failed to load snippets</h3>
                        <p>${this.escapeHtml(err.message)}</p>
                    </div>
                </div>
            `;
        }
    },

    // Build breadcrumb navigation for snippets
    buildSnippetBreadcrumb(currentPath, parentPath) {
        if (!currentPath) {
            return ''; // No breadcrumb at root
        }

        const parts = currentPath.split('/');
        let html = '<div class="snippet-breadcrumb">';
        html += `<span class="breadcrumb-item" onclick="App.navigateSnippetDir('')">snippets</span>`;

        let accPath = '';
        for (let i = 0; i < parts.length; i++) {
            accPath = accPath ? accPath + '/' + parts[i] : parts[i];
            if (i === parts.length - 1) {
                html += ` / <span class="breadcrumb-current">${this.escapeHtml(parts[i])}</span>`;
            } else {
                html += ` / <span class="breadcrumb-item" onclick="App.navigateSnippetDir('${this.escapeHtml(accPath)}')">${this.escapeHtml(parts[i])}</span>`;
            }
        }

        html += '</div>';
        return html;
    },

    // Render a directory item in the snippets list
    renderSnippetDirItem(entry) {
        return `
            <div class="snippet-item" onclick="App.navigateSnippetDir('${this.escapeHtml(entry.path)}')">
                <span class="snippet-icon">&#128193;</span>
                <div class="snippet-info">
                    <div class="snippet-name">${this.escapeHtml(entry.name)}/</div>
                </div>
                <span class="snippet-arrow">&rarr;</span>
            </div>
        `;
    },

    // Render a file item in the snippets list
    renderSnippetFileItem(entry) {
        const sourceClass = entry.type === 'global' ? 'source-global' : 'source-theme';
        const sourceLabel = entry.type === 'global' ? 'Global' : 'Theme';
        const overrideNote = entry.has_override ? '<span class="override-note">(overrides theme)</span>' : '';

        // Build full path from polis root
        let fullPath;
        if (entry.type === 'global') {
            fullPath = `snippets/${entry.path}`;
        } else {
            const theme = this.snippetState.activeTheme || 'zane';
            fullPath = `themes/${theme}/snippets/${entry.path}`;
        }

        return `
            <div class="snippet-item" onclick="App.openSnippet('${this.escapeHtml(entry.path)}', '${entry.type}')">
                <span class="snippet-icon">&#128196;</span>
                <div class="snippet-info">
                    <div class="snippet-name">${this.escapeHtml(entry.name)}${overrideNote}</div>
                    <div class="snippet-path">${this.escapeHtml(fullPath)}</div>
                </div>
                <span class="snippet-source ${sourceClass}">${sourceLabel}</span>
            </div>
        `;
    },

    // Navigate to a subdirectory in snippets
    navigateSnippetDir(path) {
        this.snippetState.currentPath = path;
        this.loadViewContent();
    },

    // Open a snippet for editing
    async openSnippet(path, source) {
        try {
            const result = await this.api('GET', `/api/snippets/${encodeURIComponent(path)}?source=${source}`);

            this.snippetState.editingPath = path;
            this.snippetState.editingSource = result.source;

            // Update UI
            document.getElementById('snippet-path-label').textContent = path;

            const badge = document.getElementById('snippet-source-badge');
            badge.textContent = result.source === 'global' ? 'Global' : 'Theme';
            badge.className = `snippet-source-badge ${result.source === 'global' ? 'source-global' : 'source-theme'}`;

            // Show/hide theme warning
            const warning = document.getElementById('snippet-theme-warning');
            if (result.source === 'theme') {
                warning.classList.remove('hidden');
            } else {
                warning.classList.add('hidden');
            }

            // Set content
            document.getElementById('snippet-content').value = result.content;
            await this.updateSnippetPreview();

            this.showScreen('snippet');
        } catch (err) {
            this.showToast('Failed to load snippet: ' + err.message, 'error');
        }
    },

    // Inject sample data into Mustache template for preview
    injectSampleData(html) {
        let result = html;

        // Replace simple variables {{key}} with sample data
        for (const [key, value] of Object.entries(this.snippetSampleData)) {
            result = result.replace(new RegExp(`\\{\\{${key}\\}\\}`, 'g'), value);
        }

        // Replace Mustache partials {{> name }} with placeholder
        result = result.replace(/\{\{>\s*([^}]+)\s*\}\}/g, '<em class="partial-placeholder">[partial: $1]</em>');

        // Replace any remaining unmatched {{variables}} with placeholder styling
        result = result.replace(/\{\{([^}>#/]+)\}\}/g, '<code class="var-placeholder">{{$1}}</code>');

        return result;
    },

    // Update snippet preview
    async updateSnippetPreview() {
        const content = document.getElementById('snippet-content').value;
        const preview = document.getElementById('snippet-preview');

        if (!content.trim()) {
            preview.innerHTML = '<p class="empty-state">Preview will appear here</p>';
            return;
        }

        // For HTML snippets, render as HTML directly
        // For MD snippets, use the render API to convert markdown to HTML
        const ext = this.snippetState.editingPath ? this.snippetState.editingPath.split('.').pop() : 'html';

        if (ext === 'md') {
            // Use the render API to convert markdown to HTML
            try {
                const result = await this.api('POST', '/api/render', { markdown: content });
                preview.innerHTML = result.html || '<p class="empty-state">Preview will appear here</p>';
            } catch (err) {
                // Fallback to preformatted text on error
                preview.innerHTML = `<pre style="white-space: pre-wrap;">${this.escapeHtml(content)}</pre>`;
            }
        } else {
            // HTML preview - inject sample data then render
            const rendered = this.injectSampleData(content);
            preview.innerHTML = rendered || '<p class="empty-state">Preview will appear here</p>';
        }
    },

    // Save the current snippet
    async saveSnippet() {
        const content = document.getElementById('snippet-content').value;
        const path = this.snippetState.editingPath;
        const source = this.snippetState.editingSource;

        if (!path) {
            this.showToast('No snippet to save', 'warning');
            return;
        }

        const btn = document.getElementById('save-snippet-btn');
        btn.classList.add('btn-loading');
        btn.disabled = true;

        try {
            await this.api('PUT', `/api/snippets/${encodeURIComponent(path)}`, {
                content: content,
                source: source
            });

            this.showToast('Snippet saved', 'success');
        } catch (err) {
            this.showToast('Failed to save snippet: ' + err.message, 'error');
        } finally {
            btn.classList.remove('btn-loading');
            btn.disabled = false;
        }
    },

    // Show new snippet panel
    newSnippet() {
        document.getElementById('new-snippet-name').value = '';
        document.getElementById('new-snippet-content').value = '';
        document.getElementById('new-snippet-panel').classList.remove('hidden');
    },

    // Close new snippet panel
    closeNewSnippetPanel() {
        document.getElementById('new-snippet-panel').classList.add('hidden');
    },

    // Create a new snippet
    async createSnippet() {
        const name = document.getElementById('new-snippet-name').value.trim();
        const content = document.getElementById('new-snippet-content').value;

        if (!name) {
            this.showToast('Please enter a filename', 'warning');
            return;
        }

        // Validate extension
        if (!name.endsWith('.html') && !name.endsWith('.md')) {
            this.showToast('Filename must end with .html or .md', 'warning');
            return;
        }

        const btn = document.getElementById('new-snippet-create-btn');
        btn.classList.add('btn-loading');
        btn.disabled = true;

        try {
            await this.api('POST', '/api/snippets', {
                path: name,
                content: content
            });

            this.showToast('Snippet created', 'success');
            this.closeNewSnippetPanel();

            // Navigate to the parent directory of the new snippet
            const parts = name.split('/');
            if (parts.length > 1) {
                parts.pop();
                this.snippetState.currentPath = parts.join('/');
            } else {
                this.snippetState.currentPath = '';
            }

            await this.loadViewContent();
        } catch (err) {
            this.showToast('Failed to create snippet: ' + err.message, 'error');
        } finally {
            btn.classList.remove('btn-loading');
            btn.disabled = false;
        }
    },

    // ========================================
    // Browser Mode Functions
    // ========================================

    // Initialize view mode from settings
    async initViewMode() {
        try {
            const settings = await this.api('GET', '/api/settings');
            this.siteInfo = settings.site || {};
            const savedMode = this.siteInfo.view_mode || 'list';
            this.setViewMode(savedMode, false); // Don't save on init
            this._hideRead = !!settings.hide_read;
        } catch (err) {
            console.error('Failed to load view mode:', err);
            this.setViewMode('list', false);
        }
    },

    // Set view mode and update UI
    setViewMode(mode, save = true) {
        this.viewMode = mode;

        // Update toggle buttons
        if (this.browserElements.listModeBtn && this.browserElements.browserModeBtn) {
            this.browserElements.listModeBtn.classList.toggle('active', mode === 'list');
            this.browserElements.browserModeBtn.classList.toggle('active', mode === 'browser');
        }

        // Apply mode visibility
        this.applyViewMode();

        // Save to server if requested
        if (save) {
            this.api('POST', '/api/settings/view-mode', { view_mode: mode }).catch(err => {
                console.error('Failed to save view mode:', err);
            });
        }
    },

    // Apply view mode visibility
    applyViewMode() {
        const mainContainer = this.browserElements.mainContainer;
        const browserContainer = this.browserElements.container;

        if (!mainContainer || !browserContainer) return;

        if (this.viewMode === 'browser') {
            mainContainer.classList.add('hidden');
            browserContainer.classList.remove('hidden');
            // Show home page if no current content
            if (!this.browserState.currentUrl) {
                this.browserShowHome();
            }
        } else {
            mainContainer.classList.remove('hidden');
            browserContainer.classList.add('hidden');
        }
    },

    // Navigate to a URL or path in browser mode
    async browserNavigateTo(urlOrPath) {
        const localPath = this.translateUrlToLocalPath(urlOrPath);
        if (!localPath) {
            // External link - check if it's a polis comment/post we can fetch
            if (this.isPolisUrl(urlOrPath)) {
                await this.browserLoadRemoteContent(urlOrPath);
                return;
            }
            // Otherwise open in new tab
            window.open(urlOrPath, '_blank');
            return;
        }

        try {
            const content = await this.fetchBrowserContent(localPath);
            this.browserState.currentUrl = localPath;
            this.browserState.currentContent = content;

            // Update history
            if (this.browserState.historyIndex < this.browserState.history.length - 1) {
                // Clear forward history when navigating to new page
                this.browserState.history = this.browserState.history.slice(0, this.browserState.historyIndex + 1);
            }
            this.browserState.history.push(localPath);
            this.browserState.historyIndex = this.browserState.history.length - 1;

            this.renderBrowserContent(content);
            this.updateBrowserNavButtons();
        } catch (err) {
            this.showToast('Failed to load: ' + err.message, 'error');
        }
    },

    // Check if a URL is a polis site URL (comment or post)
    isPolisUrl(url) {
        if (!url) return false;
        // Match .polis.pub, .polis.site domains or any URL ending in .md
        return url.match(/\.polis\.(pub|site)\//) || url.endsWith('.md');
    },

    // Load remote content and render markdown in preview
    async browserLoadRemoteContent(url) {
        // Update URL bar immediately
        if (this.browserElements.urlInput) {
            this.browserElements.urlInput.value = url;
        }

        // Update live links to the remote site
        this.updateBrowserLiveLinks(url, true);

        // Show loading state
        if (this.browserElements.markdownDisplay) {
            this.browserElements.markdownDisplay.textContent = '# Loading...\n\nFetching content from ' + url;
        }
        if (this.browserElements.previewContent) {
            this.browserElements.previewContent.innerHTML = `
                <div class="browser-remote-content">
                    <p class="remote-notice">Loading remote content...</p>
                    <p class="remote-url">${this.escapeHtml(url)}</p>
                </div>
            `;
        }

        try {
            // Fetch the raw markdown from the remote URL
            const response = await fetch(url);
            if (!response.ok) {
                throw new Error(`HTTP ${response.status}: ${response.statusText}`);
            }
            const markdown = await response.text();

            // Render the markdown using the local render API (strip frontmatter for preview)
            const markdownForPreview = this.stripFrontmatter(markdown);
            let renderedHtml = '';
            try {
                const result = await this.api('POST', '/api/render', { markdown: markdownForPreview });
                renderedHtml = result.html || '';
            } catch (renderErr) {
                console.warn('Failed to render markdown, showing raw:', renderErr);
                renderedHtml = `<pre style="white-space: pre-wrap;">${this.escapeHtml(markdownForPreview)}</pre>`;
            }

            // Update state
            this.browserState.currentUrl = url;
            this.browserState.currentContent = {
                path: url,
                markdown: markdown,
                html: renderedHtml,
                editable: false,
                type: 'remote',
                remote: true
            };

            // Update history
            if (this.browserState.historyIndex < this.browserState.history.length - 1) {
                this.browserState.history = this.browserState.history.slice(0, this.browserState.historyIndex + 1);
            }
            this.browserState.history.push(url);
            this.browserState.historyIndex = this.browserState.history.length - 1;

            // Display markdown in left pane (respects frontmatter setting)
            this.displayMarkdownContent(markdown);

            // Render the preview with styled content
            if (this.browserElements.previewContent) {
                this.browserElements.previewContent.innerHTML = renderedHtml;
            }

            this.updateBrowserEditState(false);
            this.updateBrowserNavButtons();
        } catch (err) {
            console.error('Failed to fetch remote content:', err);
            if (this.browserElements.markdownDisplay) {
                this.browserElements.markdownDisplay.textContent = `# Failed to Load\n\nCould not fetch content from:\n${url}\n\nError: ${err.message}`;
            }
            if (this.browserElements.previewContent) {
                this.browserElements.previewContent.innerHTML = `
                    <div class="browser-remote-content error">
                        <p class="remote-notice">Failed to load remote content</p>
                        <p class="remote-url">${this.escapeHtml(url)}</p>
                        <p class="remote-error">${this.escapeHtml(err.message)}</p>
                        <p><a href="${this.escapeHtml(url)}" target="_blank">Open in new tab</a></p>
                    </div>
                `;
            }
            // Hide live links on error
            this.hideBrowserLiveLinks();
        }
    },

    // Fetch content for browser mode
    async fetchBrowserContent(path) {
        return await this.api('GET', `/api/content/${encodeURIComponent(path)}`);
    },

    // Render content in browser mode
    renderBrowserContent(content) {
        // Update URL bar
        if (this.browserElements.urlInput) {
            this.browserElements.urlInput.value = content.path;
        }

        // Update live links to the live site
        this.updateBrowserLiveLinks(content.path, false);

        // Update markdown display (respects frontmatter setting)
        this.displayMarkdownContent(content.markdown);

        // Update preview
        if (this.browserElements.previewContent) {
            this.browserElements.previewContent.innerHTML = content.html;
            this.attachBrowserLinkHandlers(this.browserElements.previewContent);

            // If viewing index page, append all blessed comments section
            if (content.path === 'index.html' || content.path === 'index.md') {
                this.appendBlessedCommentsSection();
            }
            // If viewing a post, append comments on that specific post
            else if (content.path && content.path.startsWith('posts/')) {
                this.appendPostComments(content.path);
            }
        }

        // Update status badge and edit button
        this.updateBrowserEditState(content.editable);

        // Exit edit mode if currently editing
        if (this.browserState.isEditing) {
            this.browserCancelEdit();
        }
    },

    // Append blessed comments section to browser preview
    async appendBlessedCommentsSection() {
        try {
            const data = await this.api('GET', '/api/blessed-comments');
            if (!data.comments || data.comments.length === 0) {
                return; // No blessed comments to show
            }

            // Flatten the nested structure: comments is array of {post, blessed[]}
            const allComments = [];
            data.comments.forEach(postEntry => {
                if (postEntry.blessed && Array.isArray(postEntry.blessed)) {
                    postEntry.blessed.forEach(bc => {
                        allComments.push({
                            comment_url: bc.url,
                            timestamp: bc.blessed_at,
                            post_url: postEntry.post
                        });
                    });
                }
            });

            if (allComments.length === 0) {
                return;
            }

            // Create the section HTML
            const section = document.createElement('div');
            section.className = 'blessed-comments-section';
            section.innerHTML = '<h2>Comments On My Posts</h2>';

            const list = document.createElement('div');
            list.className = 'blessed-comments-list';

            allComments.forEach(comment => {
                const item = document.createElement('div');
                item.className = 'blessed-comment-item';
                item.dataset.url = comment.comment_url;

                // Extract author domain from comment_url
                let author = 'unknown';
                try {
                    const url = new URL(comment.comment_url);
                    author = url.hostname;
                } catch (e) {
                    author = comment.comment_url.split('/')[2] || 'unknown';
                }

                // Format date
                let dateStr = '';
                if (comment.timestamp) {
                    const date = new Date(comment.timestamp);
                    dateStr = date.toLocaleDateString('en-US', { month: 'long', day: 'numeric', year: 'numeric' });
                }

                item.innerHTML = `
                    <span class="comment-date-author">
                        <span class="comment-date">${this.escapeHtml(dateStr)}</span>
                        <span class="comment-author">on ${this.escapeHtml(author)}</span>
                    </span>
                    <span class="comment-preview">View comment...</span>
                `;

                // Add click handler to load remote content
                item.addEventListener('click', () => {
                    this.browserNavigateTo(comment.comment_url);
                });

                list.appendChild(item);
            });

            section.appendChild(list);

            // Insert before footer if it exists, otherwise append
            const footer = this.browserElements.previewContent.querySelector('footer, .site-footer');
            if (footer) {
                footer.parentNode.insertBefore(section, footer);
            } else {
                this.browserElements.previewContent.appendChild(section);
            }

        } catch (err) {
            console.log('Could not load blessed comments:', err.message);
        }
    },

    // Append comments on a specific post
    async appendPostComments(postPath) {
        try {
            const data = await this.api('GET', '/api/blessed-comments');
            if (!data.comments || data.comments.length === 0) {
                console.log('appendPostComments: No blessed comments in data');
                return;
            }

            // Build the full URL for this post to match against the "post" field
            const siteBaseUrl = this.siteInfo?.base_url || '';
            // Convert .html path to .md for matching (blessed comments use .md URLs)
            const mdPath = postPath.replace(/\.html$/, '.md');
            const postUrl = siteBaseUrl ? `${siteBaseUrl}/${mdPath}` : mdPath;

            console.log('appendPostComments: Looking for post', { postPath, mdPath, postUrl, siteBaseUrl });
            console.log('appendPostComments: Available posts', data.comments.map(e => e.post));

            // Find the post entry that matches this post
            // data.comments is array of {post: "url", blessed: [{url, version, blessed_at}]}
            const postEntry = data.comments.find(entry => {
                const entryPost = entry.post || '';
                return entryPost === postUrl ||
                       entryPost.endsWith('/' + mdPath);
            });

            if (!postEntry || !postEntry.blessed || postEntry.blessed.length === 0) {
                return; // No comments on this post
            }

            // Create the section HTML
            const section = document.createElement('div');
            section.className = 'blessed-comments-section post-comments';
            section.innerHTML = `<h2>Comments (${postEntry.blessed.length})</h2>`;

            const list = document.createElement('div');
            list.className = 'blessed-comments-list';

            postEntry.blessed.forEach(bc => {
                const item = document.createElement('div');
                item.className = 'blessed-comment-item';
                item.dataset.url = bc.url;

                // Extract author domain from comment url
                let author = 'unknown';
                try {
                    const url = new URL(bc.url);
                    author = url.hostname;
                } catch (e) {
                    author = bc.url.split('/')[2] || 'unknown';
                }

                // Format date
                let dateStr = '';
                if (bc.blessed_at) {
                    const date = new Date(bc.blessed_at);
                    dateStr = date.toLocaleDateString('en-US', { month: 'long', day: 'numeric', year: 'numeric' });
                }

                item.innerHTML = `
                    <span class="comment-date-author">
                        <span class="comment-date">${this.escapeHtml(dateStr)}</span>
                        <span class="comment-author">on ${this.escapeHtml(author)}</span>
                    </span>
                    <span class="comment-preview">View comment...</span>
                `;

                // Add click handler to load remote content
                item.addEventListener('click', () => {
                    this.browserNavigateTo(bc.url);
                });

                list.appendChild(item);
            });

            section.appendChild(list);
            this.browserElements.previewContent.appendChild(section);

        } catch (err) {
            console.log('Could not load post comments:', err.message);
        }
    },

    // Attach click handlers to links in browser preview
    attachBrowserLinkHandlers(container) {
        const siteBaseUrl = this.siteInfo?.base_url || '';

        container.querySelectorAll('a').forEach(link => {
            const href = link.getAttribute('href');
            const localPath = this.translateUrlToLocalPath(href);

            if (localPath) {
                // Internal link - navigate within browser mode
                link.addEventListener('click', (e) => {
                    e.preventDefault();
                    this.browserNavigateTo(localPath);
                });
                link.style.cursor = 'pointer';
            } else if (href && this.isPolisUrl(href)) {
                // External polis link - load remote markdown
                link.addEventListener('click', (e) => {
                    e.preventDefault();
                    this.browserNavigateTo(href);
                });
                link.style.cursor = 'pointer';
            } else if (href && !href.startsWith('#') && !href.startsWith('mailto:')) {
                // External link - ensure opens in new tab
                link.setAttribute('target', '_blank');
                link.setAttribute('rel', 'noopener noreferrer');
            }
        });
    },

    // Translate URL to local path
    translateUrlToLocalPath(href) {
        if (!href) return null;

        const siteBaseUrl = this.siteInfo?.base_url || '';

        // Case 1: Full URL matching site base URL
        if (siteBaseUrl && href.startsWith(siteBaseUrl)) {
            let path = href.slice(siteBaseUrl.length).replace(/^\//, '');
            return this.normalizePath(path);
        }

        // Case 2: Absolute path starting with /
        if (href.startsWith('/') && !href.startsWith('//')) {
            return this.normalizePath(href.slice(1));
        }

        // Case 3: Relative path (no protocol)
        if (!href.includes('://') && !href.startsWith('#') && !href.startsWith('mailto:')) {
            // Resolve relative to current path
            return this.resolveRelativePath(href, this.browserState.currentUrl);
        }

        // Case 4: External URL or anchor - return null
        return null;
    },

    // Normalize path for browser mode navigation
    normalizePath(path) {
        if (!path) return null;
        // Keep .html files as-is for browser mode (we load rendered HTML)
        if (path.endsWith('.html')) {
            return path;
        }
        // Convert .md to .html for navigation (browser mode shows rendered content)
        if (path.endsWith('.md')) {
            return path.slice(0, -3) + '.html';
        }
        return path;
    },

    // Resolve relative path
    resolveRelativePath(relativePath, currentUrl) {
        if (!currentUrl) return relativePath;

        // Get directory of current URL
        const parts = currentUrl.split('/');
        parts.pop(); // Remove filename
        const currentDir = parts.join('/');

        if (relativePath.startsWith('./')) {
            relativePath = relativePath.slice(2);
        }

        // Handle ../ navigation
        while (relativePath.startsWith('../')) {
            relativePath = relativePath.slice(3);
            if (parts.length > 0) {
                parts.pop();
            }
        }

        const resolved = parts.length > 0 ? parts.join('/') + '/' + relativePath : relativePath;
        return this.normalizePath(resolved);
    },

    // Browser back navigation
    browserBack() {
        if (this.browserState.historyIndex > 0) {
            this.browserState.historyIndex--;
            const path = this.browserState.history[this.browserState.historyIndex];
            this.browserLoadPath(path);
        }
    },

    // Browser forward navigation
    browserForward() {
        if (this.browserState.historyIndex < this.browserState.history.length - 1) {
            this.browserState.historyIndex++;
            const path = this.browserState.history[this.browserState.historyIndex];
            this.browserLoadPath(path);
        }
    },

    // Load path without adding to history
    async browserLoadPath(path) {
        // Check if this is a remote URL
        if (path && path.includes('://')) {
            await this.browserLoadRemoteContentNoHistory(path);
            return;
        }

        try {
            const content = await this.fetchBrowserContent(path);
            this.browserState.currentUrl = path;
            this.browserState.currentContent = content;
            this.renderBrowserContent(content);
            this.updateBrowserNavButtons();
        } catch (err) {
            this.showToast('Failed to load: ' + err.message, 'error');
        }
    },

    // Load remote content without modifying history (for back/forward)
    async browserLoadRemoteContentNoHistory(url) {
        if (this.browserElements.urlInput) {
            this.browserElements.urlInput.value = url;
        }

        // Update live links
        this.updateBrowserLiveLinks(url, true);

        try {
            const response = await fetch(url);
            if (!response.ok) {
                throw new Error(`HTTP ${response.status}: ${response.statusText}`);
            }
            const markdown = await response.text();

            // Render the markdown (strip frontmatter for preview)
            const markdownForPreview = this.stripFrontmatter(markdown);
            let renderedHtml = '';
            try {
                const result = await this.api('POST', '/api/render', { markdown: markdownForPreview });
                renderedHtml = result.html || '';
            } catch (renderErr) {
                renderedHtml = `<pre style="white-space: pre-wrap;">${this.escapeHtml(markdownForPreview)}</pre>`;
            }

            this.browserState.currentUrl = url;
            this.browserState.currentContent = {
                path: url,
                markdown: markdown,
                html: renderedHtml,
                editable: false,
                type: 'remote',
                remote: true
            };

            // Display markdown (respects frontmatter setting)
            this.displayMarkdownContent(markdown);
            if (this.browserElements.previewContent) {
                this.browserElements.previewContent.innerHTML = renderedHtml;
            }

            this.updateBrowserEditState(false);
            this.updateBrowserNavButtons();
        } catch (err) {
            console.error('Failed to fetch remote content:', err);
            this.hideBrowserLiveLinks();
        }
    },

    // Update browser nav button states
    updateBrowserNavButtons() {
        if (this.browserElements.backBtn) {
            this.browserElements.backBtn.disabled = this.browserState.historyIndex <= 0;
        }
        if (this.browserElements.forwardBtn) {
            this.browserElements.forwardBtn.disabled = this.browserState.historyIndex >= this.browserState.history.length - 1;
        }
    },

    // Update live links in pane headers with URLs to the live site
    updateBrowserLiveLinks(path, isRemote = false) {
        const linkHtml = this.browserElements.linkHtml;
        const linkMd = this.browserElements.linkMd;

        // Helper to hide both links
        const hideLinks = () => {
            if (linkHtml) linkHtml.classList.add('hidden');
            if (linkMd) linkMd.classList.add('hidden');
        };

        // For remote URLs, extract base and show links
        if (isRemote && path) {
            try {
                const url = new URL(path);
                const basePath = url.pathname.replace(/\.(md|html)$/, '');
                const baseUrl = `${url.protocol}//${url.host}${basePath}`;
                const htmlUrl = baseUrl + '.html';
                const mdUrl = baseUrl + '.md';

                if (linkHtml) {
                    linkHtml.href = htmlUrl;
                    linkHtml.textContent = htmlUrl;
                    linkHtml.classList.remove('hidden');
                }
                if (linkMd) {
                    linkMd.href = mdUrl;
                    linkMd.textContent = mdUrl;
                    linkMd.classList.remove('hidden');
                }
            } catch (e) {
                hideLinks();
            }
            return;
        }

        // For local content, use the site base URL
        if (!this.siteBaseUrl || !path) {
            hideLinks();
            return;
        }

        // Build the live URLs
        const basePath = path.replace(/\.(md|html)$/, '');
        const baseUrl = this.siteBaseUrl.replace(/\/$/, '');
        const htmlUrl = `${baseUrl}/${basePath}.html`;
        const mdUrl = `${baseUrl}/${basePath}.md`;

        if (linkHtml) {
            linkHtml.href = htmlUrl;
            linkHtml.textContent = htmlUrl;
            linkHtml.classList.remove('hidden');
        }
        if (linkMd) {
            linkMd.href = mdUrl;
            linkMd.textContent = mdUrl;
            linkMd.classList.remove('hidden');
        }
    },

    // Hide live links in pane headers
    hideBrowserLiveLinks() {
        if (this.browserElements.linkHtml) {
            this.browserElements.linkHtml.classList.add('hidden');
        }
        if (this.browserElements.linkMd) {
            this.browserElements.linkMd.classList.add('hidden');
        }
    },

    // Update frontmatter toggle button appearance
    updateFrontmatterToggle() {
        const toggle = this.browserElements.frontmatterToggle;
        if (toggle) {
            toggle.classList.toggle('active', this.showFrontmatter);
            toggle.textContent = this.showFrontmatter ? 'Hide FM' : 'Show FM';
            toggle.title = this.showFrontmatter ? 'Hide frontmatter' : 'Show frontmatter';
        }
    },

    // Toggle frontmatter visibility and save to config
    async toggleFrontmatter() {
        this.showFrontmatter = !this.showFrontmatter;
        this.updateFrontmatterToggle();

        // Re-display current content with new setting
        if (this.browserState.currentContent) {
            this.displayMarkdownContent(this.browserState.currentContent.markdown);
        }

        // Save to server
        try {
            await this.api('POST', '/api/settings/show-frontmatter', {
                show_frontmatter: this.showFrontmatter
            });
        } catch (err) {
            console.error('Failed to save frontmatter setting:', err);
        }
    },

    // Update editor FM toggle button and display pane visibility
    updateEditorFmToggle() {
        const toggle = document.getElementById('editor-fm-toggle');
        const fmDisplay = document.getElementById('editor-fm-display');
        const fmContent = document.getElementById('editor-fm-content');
        const hasFm = !!this.currentFrontmatter;

        if (toggle) {
            // Only show the toggle button when there is frontmatter to display
            toggle.classList.toggle('hidden', !hasFm);
            toggle.classList.toggle('active', this.showFrontmatter);
            toggle.textContent = this.showFrontmatter ? 'Hide FM' : 'Show FM';
            toggle.title = this.showFrontmatter ? 'Hide frontmatter' : 'Show frontmatter';
        }

        if (fmDisplay && fmContent) {
            const visible = hasFm && this.showFrontmatter;
            fmDisplay.classList.toggle('hidden', !visible);
            if (visible) {
                fmContent.textContent = this.currentFrontmatter;
            }
        }
    },

    // Toggle frontmatter display pane and save setting
    async toggleEditorFrontmatter() {
        this.showFrontmatter = !this.showFrontmatter;
        this.updateEditorFmToggle();
        this.updateFrontmatterToggle();

        // Save to server
        try {
            await this.api('POST', '/api/settings/show-frontmatter', {
                show_frontmatter: this.showFrontmatter
            });
        } catch (err) {
            console.error('Failed to save frontmatter setting:', err);
        }
    },

    // Strip frontmatter from markdown content
    stripFrontmatter(markdown) {
        if (!markdown) return markdown;
        // Match frontmatter between --- markers at the start
        const match = markdown.match(/^---\r?\n[\s\S]*?\r?\n---\r?\n?/);
        if (match) {
            return markdown.slice(match[0].length);
        }
        return markdown;
    },

    // Display markdown content in the browser pane (respects frontmatter setting)
    displayMarkdownContent(markdown) {
        if (this.browserElements.markdownDisplay) {
            const content = this.showFrontmatter ? markdown : this.stripFrontmatter(markdown);
            this.browserElements.markdownDisplay.textContent = content;
        }
    },

    // Update edit state UI
    updateBrowserEditState(editable) {
        const editBtn = this.browserElements.editBtn;

        if (editBtn) {
            const textSpan = editBtn.querySelector('span');
            if (this.browserState.isEditing) {
                // Show cancel state
                editBtn.classList.remove('hidden');
                editBtn.classList.add('editing');
                if (textSpan) textSpan.textContent = 'CANCEL';
                editBtn.title = 'Cancel editing';
            } else if (editable) {
                // Show edit state
                editBtn.classList.remove('hidden', 'editing');
                if (textSpan) textSpan.textContent = 'EDIT';
                editBtn.title = 'Edit';
            } else {
                // Hide button for read-only content
                editBtn.classList.add('hidden');
                editBtn.classList.remove('editing');
            }
        }
    },

    // Show home page in browser mode (loads index.html)
    async browserShowHome() {
        // Clear history when going home
        this.browserState.history = [];
        this.browserState.historyIndex = -1;

        try {
            // Try to load index.html as the home page
            const content = await this.fetchBrowserContent('index.html');

            // Success - update state and render
            this.browserState.currentUrl = 'index.html';
            this.browserState.currentContent = content;
            this.browserState.history.push('index.html');
            this.browserState.historyIndex = 0;

            this.renderBrowserContent(content);
            this.updateBrowserNavButtons();
        } catch (err) {
            // index.html doesn't exist - site needs to be rendered
            console.log('No index.html found, prompting to render site');
            this.showRenderSitePrompt();
        }
    },

    // Show prompt to render the site
    showRenderSitePrompt() {
        if (this.browserElements.urlInput) {
            this.browserElements.urlInput.value = '';
        }
        // Hide live links when no content
        this.hideBrowserLiveLinks();
        if (this.browserElements.markdownDisplay) {
            this.browserElements.markdownDisplay.textContent = '# Site Not Rendered\n\nRun `polis render` to generate your site.';
        }
        if (this.browserElements.previewContent) {
            this.browserElements.previewContent.innerHTML = `
                <div class="browser-home-page">
                    <h2>Site Not Rendered</h2>
                    <p>Browser mode displays your rendered site, but it looks like your site hasn't been generated yet.</p>
                    <p>To generate your site, run:</p>
                    <pre class="browser-code-block">polis render</pre>
                    <p>This command:</p>
                    <ul>
                        <li>Converts your markdown posts to HTML</li>
                        <li>Generates index.html and other pages</li>
                        <li>Only modifies files in your local site directory</li>
                        <li>Does not upload or publish anything</li>
                    </ul>
                    <p>After rendering, refresh this page to browse your site.</p>
                </div>
            `;
        }

        this.browserState.currentUrl = null;
        this.browserState.currentContent = null;
        this.updateBrowserEditState(false);
        this.updateBrowserNavButtons();
    },

    // Toggle edit mode in browser
    browserToggleEdit() {
        const content = this.browserState.currentContent;
        if (!content || !content.editable) return;

        if (this.browserState.isEditing) {
            // Exit edit mode without saving
            this.browserCancelEdit();
        } else {
            // Enter edit mode
            this.browserState.isEditing = true;
            this.browserState.originalMarkdown = content.markdown;

            // Show edit textarea, hide display
            if (this.browserElements.markdownDisplay) {
                this.browserElements.markdownDisplay.classList.add('hidden');
            }
            if (this.browserElements.markdownEdit) {
                this.browserElements.markdownEdit.classList.remove('hidden');
                this.browserElements.markdownEdit.value = content.markdown;
                this.browserElements.markdownEdit.focus();
            }

            // Show actions footer
            if (this.browserElements.actionsFooter) {
                this.browserElements.actionsFooter.classList.remove('hidden');
            }

            this.updateBrowserEditState(true);
        }
    },

    // Cancel edit mode
    browserCancelEdit() {
        this.browserState.isEditing = false;

        // Hide edit textarea, show display
        if (this.browserElements.markdownEdit) {
            this.browserElements.markdownEdit.classList.add('hidden');
        }
        if (this.browserElements.markdownDisplay) {
            this.browserElements.markdownDisplay.classList.remove('hidden');
        }

        // Hide actions footer
        if (this.browserElements.actionsFooter) {
            this.browserElements.actionsFooter.classList.add('hidden');
        }

        // Restore original content in preview
        if (this.browserState.currentContent) {
            this.updateBrowserEditState(this.browserState.currentContent.editable);
        }
    },

    // Update preview while editing (debounced)
    browserUpdatePreview: (function() {
        let timeout = null;
        return async function() {
            if (timeout) clearTimeout(timeout);
            timeout = setTimeout(async () => {
                if (!this.browserState.isEditing) return;

                const markdown = this.browserElements.markdownEdit?.value || '';
                if (!markdown.trim()) return;

                try {
                    const result = await this.api('POST', '/api/render', { markdown });
                    if (this.browserElements.previewContent && this.browserState.isEditing) {
                        this.browserElements.previewContent.innerHTML = result.html;
                        this.attachBrowserLinkHandlers(this.browserElements.previewContent);
                    }
                } catch (err) {
                    console.error('Preview update failed:', err);
                }
            }, 300);
        };
    })(),

    // Publish changes from browser mode
    async browserPublish() {
        const content = this.browserState.currentContent;
        if (!content || !this.browserState.isEditing) return;

        const markdown = this.browserElements.markdownEdit?.value || '';
        if (!markdown.trim()) {
            this.showToast('Nothing to publish', 'warning');
            return;
        }

        const confirmed = await this.showConfirmModal(
            'Publish Changes',
            'This will re-sign and publish the updated content.',
            'Publish'
        );
        if (!confirmed) return;

        const publishBtn = this.browserElements.publishBtn;
        if (publishBtn) {
            publishBtn.classList.add('btn-loading');
            publishBtn.disabled = true;
        }

        try {
            await this.api('POST', '/api/republish', {
                path: content.path,
                markdown: markdown
            });

            this.showToast('Published successfully!', 'success');

            // Reload content
            await this.browserLoadPath(content.path);
            this.browserCancelEdit();
        } catch (err) {
            this.showToast('Failed to publish: ' + err.message, 'error');
        } finally {
            if (publishBtn) {
                publishBtn.classList.remove('btn-loading');
                publishBtn.disabled = false;
            }
        }
    },

    // Save draft from browser mode
    async browserSaveDraft() {
        const content = this.browserState.currentContent;
        if (!content || !this.browserState.isEditing) return;

        const markdown = this.browserElements.markdownEdit?.value || '';
        if (!markdown.trim()) {
            this.showToast('Nothing to save', 'warning');
            return;
        }

        const saveDraftBtn = this.browserElements.saveDraftBtn;
        if (saveDraftBtn) {
            saveDraftBtn.classList.add('btn-loading');
            saveDraftBtn.disabled = true;
        }

        try {
            // Extract title for draft ID
            const title = this.extractTitleFromMarkdown(markdown) || 'untitled';
            const id = this.slugify(title);

            await this.api('POST', '/api/drafts', { id, markdown });
            this.showToast('Draft saved', 'success');
        } catch (err) {
            this.showToast('Failed to save draft: ' + err.message, 'error');
        } finally {
            if (saveDraftBtn) {
                saveDraftBtn.classList.remove('btn-loading');
                saveDraftBtn.disabled = false;
            }
        }
    },

    // Bind browser mode events
    bindBrowserEvents() {
        const be = this.browserElements;

        // Mode toggle buttons
        if (be.listModeBtn) {
            be.listModeBtn.addEventListener('click', () => this.setViewMode('list'));
        }
        if (be.browserModeBtn) {
            be.browserModeBtn.addEventListener('click', () => this.setViewMode('browser'));
        }

        // Browser nav buttons
        if (be.backBtn) {
            be.backBtn.addEventListener('click', () => this.browserBack());
        }
        if (be.forwardBtn) {
            be.forwardBtn.addEventListener('click', () => this.browserForward());
        }
        if (be.homeBtn) {
            be.homeBtn.addEventListener('click', () => this.browserShowHome());
        }
        if (be.editBtn) {
            be.editBtn.addEventListener('click', () => this.browserToggleEdit());
        }
        if (be.settingsBtn) {
            be.settingsBtn.addEventListener('click', () => {
                // Switch to list mode and show settings
                this.setViewMode('list');
                this.setActiveView('settings');
            });
        }
        if (be.frontmatterToggle) {
            be.frontmatterToggle.addEventListener('click', () => this.toggleFrontmatter());
        }

        // Snippet edit toggle
        if (be.snippetEditToggle) {
            be.snippetEditToggle.addEventListener('click', () => this.toggleSnippetEditMode());
        }

        // Includes toggle (for viewing nested snippet includes)
        if (be.includesToggle) {
            be.includesToggle.addEventListener('click', () => this.toggleIncludesMode());
        }

        // Edit action buttons
        if (be.cancelBtn) {
            be.cancelBtn.addEventListener('click', () => this.browserCancelEdit());
        }
        if (be.saveDraftBtn) {
            be.saveDraftBtn.addEventListener('click', () => this.browserSaveDraft());
        }
        if (be.publishBtn) {
            be.publishBtn.addEventListener('click', () => this.browserPublish());
        }

        // Live preview while editing
        if (be.markdownEdit) {
            be.markdownEdit.addEventListener('input', () => this.browserUpdatePreview());
        }
    },

    // ========================================
    // Snippet Edit Mode Functions
    // ========================================

    // Toggle snippet edit mode on/off
    toggleSnippetEditMode() {
        this.browserState.snippetEditMode = !this.browserState.snippetEditMode;
        const toggle = this.browserElements.snippetEditToggle;

        if (this.browserState.snippetEditMode) {
            toggle.classList.add('active');
            this.enableSnippetEditMode();
        } else {
            toggle.classList.remove('active');
            this.disableSnippetEditMode();
        }
    },

    // Enable snippet edit mode - scan for markers and add click handlers
    enableSnippetEditMode() {
        const previewContent = this.browserElements.previewContent;
        if (!previewContent) return;

        // Find all snippet boundary markers
        const boundaries = previewContent.querySelectorAll('.polis-snippet-boundary');
        if (boundaries.length === 0) {
            this.showToast('No editable snippets found. Re-render the page to enable snippet editing.', 'info');
            return;
        }

        // For each boundary, wrap the snippet content in an editable container
        boundaries.forEach((boundary, index) => {
            const snippetId = boundary.dataset.snippet;
            const snippetPath = boundary.dataset.path;
            const snippetSource = boundary.dataset.source;

            // Find the end marker using the comment text
            const endComment = this.findSnippetEndComment(boundary, snippetId);
            if (!endComment) {
                console.warn('Could not find end marker for snippet:', snippetId);
                return;
            }

            // Get all nodes between start and end
            const wrapper = document.createElement('div');
            wrapper.className = 'snippet-editable';
            wrapper.dataset.snippetId = snippetId;
            wrapper.dataset.snippetPath = snippetPath;
            wrapper.dataset.snippetSource = snippetSource;
            wrapper.dataset.snippetLabel = snippetPath.split('/').pop(); // Just filename for label

            // Collect nodes between boundary and end comment
            let currentNode = boundary.nextSibling;
            const nodesToWrap = [];
            while (currentNode && currentNode !== endComment) {
                nodesToWrap.push(currentNode);
                currentNode = currentNode.nextSibling;
            }

            // Only wrap if we found content
            if (nodesToWrap.length > 0) {
                // Insert wrapper before first content node
                boundary.parentNode.insertBefore(wrapper, nodesToWrap[0]);
                // Move content nodes into wrapper
                nodesToWrap.forEach(node => wrapper.appendChild(node));
            }

            // Add click handler
            wrapper.addEventListener('click', (e) => {
                // Stop propagation to prevent parent snippets from firing
                e.stopPropagation();
                this.openSnippetForEditing(snippetId, snippetPath, snippetSource);
            });
        });
    },

    // Find the end comment for a snippet (HTML comment)
    findSnippetEndComment(boundary, snippetId) {
        const parent = boundary.parentNode;
        if (!parent) return null;

        // Walk through siblings looking for the end comment
        let current = boundary.nextSibling;
        while (current) {
            if (current.nodeType === Node.COMMENT_NODE) {
                const commentText = current.textContent.trim();
                if (commentText.includes('POLIS-SNIPPET-END:') && commentText.includes(snippetId)) {
                    return current;
                }
            }
            current = current.nextSibling;
        }
        return null;
    },

    // Disable snippet edit mode - remove click handlers and styling
    disableSnippetEditMode() {
        const previewContent = this.browserElements.previewContent;
        if (!previewContent) return;

        // If editing a snippet, cancel and return to markdown view
        if (this.browserState.editingSnippet) {
            this.cancelSnippetEdit();
        }

        // Find all editable wrappers and unwrap them
        const wrappers = previewContent.querySelectorAll('.snippet-editable');
        wrappers.forEach(wrapper => {
            // Move children back to parent
            const parent = wrapper.parentNode;
            while (wrapper.firstChild) {
                parent.insertBefore(wrapper.firstChild, wrapper);
            }
            parent.removeChild(wrapper);
        });
    },

    // Open a snippet for editing
    async openSnippetForEditing(snippetId, snippetPath, snippetSource) {
        try {
            // Fetch snippet content (include source to find correct file)
            const result = await this.api('GET', `/api/snippets/${encodeURIComponent(snippetPath)}?source=${snippetSource}`);

            // If we're already editing a snippet, push it to the nav stack for back navigation
            if (this.browserState.editingSnippet) {
                this.browserState.snippetNavStack.push({
                    path: this.browserState.editingSnippet.path,
                    source: this.browserState.editingSnippet.source,
                    content: this.browserState.editingSnippet.content,
                });
            }

            // Store editing state
            this.browserState.editingSnippet = {
                id: snippetId,
                path: snippetPath,
                source: snippetSource,
                content: result.content || '',
            };

            // Reset showIncludes when opening a new snippet
            this.browserState.showIncludes = false;
            if (this.browserElements.includesToggle) {
                this.browserElements.includesToggle.classList.remove('active');
            }

            // Update pane header to show snippet name
            const filename = snippetPath.split('/').pop();
            if (this.browserElements.codePaneLabel) {
                this.browserElements.codePaneLabel.textContent = `Snippet: ${filename}`;
            }

            // Show INCLUDES toggle when editing a snippet
            if (this.browserElements.includesToggle) {
                this.browserElements.includesToggle.classList.remove('hidden');
            }

            // Show parent link if we have a navigation stack
            this.updateSnippetParentLink();

            // Show the snippet content in the edit area (default: edit mode)
            this.browserElements.markdownDisplay.classList.add('hidden');
            this.browserElements.markdownEdit.classList.remove('hidden');
            this.browserElements.markdownEdit.value = result.content || '';

            // Show the actions footer for save/cancel
            if (this.browserElements.actionsFooter) {
                this.browserElements.actionsFooter.classList.remove('hidden');
            }

            // Update button text for snippet editing
            if (this.browserElements.saveDraftBtn) {
                this.browserElements.saveDraftBtn.textContent = 'Cancel';
                this.browserElements.saveDraftBtn.onclick = () => this.cancelSnippetEdit();
            }
            if (this.browserElements.publishBtn) {
                this.browserElements.publishBtn.textContent = 'Save Snippet';
                this.browserElements.publishBtn.onclick = () => this.saveSnippetEdit();
            }
            if (this.browserElements.cancelBtn) {
                this.browserElements.cancelBtn.classList.add('hidden');
            }

            // Update edit button to show editing state
            if (this.browserElements.editBtn) {
                this.browserElements.editBtn.classList.add('editing');
                this.browserElements.editBtn.innerHTML = '✎ <span>EDITING</span>';
            }

            // Focus the textarea
            this.browserElements.markdownEdit.focus();

        } catch (err) {
            this.showToast('Failed to load snippet: ' + err.message, 'error');
        }
    },

    // Update the parent link for snippet navigation
    updateSnippetParentLink() {
        const parentLink = this.browserElements.snippetParentLink;
        if (!parentLink) return;

        if (this.browserState.snippetNavStack.length > 0) {
            const parent = this.browserState.snippetNavStack[this.browserState.snippetNavStack.length - 1];
            const parentFilename = parent.path.split('/').pop();
            parentLink.innerHTML = `<a href="#" onclick="App.navigateToParentSnippet(); return false;">← ${this.escapeHtml(parentFilename)}</a>`;
            parentLink.classList.remove('hidden');
        } else {
            parentLink.classList.add('hidden');
        }
    },

    // Navigate back to parent snippet
    async navigateToParentSnippet() {
        if (this.browserState.snippetNavStack.length === 0) return;

        const parent = this.browserState.snippetNavStack.pop();

        // Restore parent snippet directly without fetching
        this.browserState.editingSnippet = {
            id: parent.path,
            path: parent.path,
            source: parent.source,
            content: parent.content,
        };

        // Reset showIncludes
        this.browserState.showIncludes = false;
        if (this.browserElements.includesToggle) {
            this.browserElements.includesToggle.classList.remove('active');
        }

        // Update pane header
        const filename = parent.path.split('/').pop();
        if (this.browserElements.codePaneLabel) {
            this.browserElements.codePaneLabel.textContent = `Snippet: ${filename}`;
        }

        // Update parent link
        this.updateSnippetParentLink();

        // Show content in edit area
        this.browserElements.markdownEdit.value = parent.content;
        this.browserElements.markdownDisplay.classList.add('hidden');
        this.browserElements.markdownEdit.classList.remove('hidden');
    },

    // Toggle INCLUDES mode (show snippet content as display with clickable includes)
    toggleIncludesMode() {
        if (!this.browserState.editingSnippet) return;

        // If switching from edit mode, save the textarea content first
        if (!this.browserState.showIncludes) {
            this.browserState.editingSnippet.content = this.browserElements.markdownEdit.value;
        }

        this.browserState.showIncludes = !this.browserState.showIncludes;
        const toggle = this.browserElements.includesToggle;

        if (this.browserState.showIncludes) {
            toggle.classList.add('active');
            this.renderSnippetWithIncludes();
        } else {
            toggle.classList.remove('active');
            this.renderSnippetForEditing();
        }
    },

    // Render snippet in display mode with clickable includes
    renderSnippetWithIncludes() {
        const content = this.browserState.editingSnippet?.content || '';

        // Parse {{> path }} patterns and make them clickable
        const html = this.parseIncludesToHtml(content);

        // Show in display area
        this.browserElements.markdownEdit.classList.add('hidden');
        this.browserElements.markdownDisplay.classList.remove('hidden');
        this.browserElements.markdownDisplay.innerHTML = html;
    },

    // Parse include patterns and convert to clickable HTML
    parseIncludesToHtml(content) {
        // Escape HTML first
        let html = this.escapeHtml(content);

        // Pattern for mustache partials: {{> path }} or {{>path}}
        // The path can be: global:name, theme:name, or just name
        const includePattern = /\{\{&gt;\s*([^\s}]+)\s*\}\}/g;

        html = html.replace(includePattern, (match, path) => {
            // Determine source from prefix
            let source = 'theme'; // default
            let snippetPath = path;

            if (path.startsWith('global:')) {
                source = 'global';
                snippetPath = path.substring(7);
            } else if (path.startsWith('theme:')) {
                source = 'theme';
                snippetPath = path.substring(6);
            }

            return `<span class="snippet-include-link" data-path="${this.escapeHtml(snippetPath)}" data-source="${source}" onclick="App.navigateToIncludedSnippet('${this.escapeHtml(snippetPath)}', '${source}')">{{&gt; ${this.escapeHtml(path)} }}</span>`;
        });

        return html;
    },

    // Navigate to an included snippet (when clicking a {{> path }} link)
    async navigateToIncludedSnippet(snippetPath, source) {
        // Open the included snippet for editing
        await this.openSnippetForEditing(snippetPath, snippetPath, source);
    },

    // Render snippet in edit mode (textarea)
    renderSnippetForEditing() {
        const content = this.browserState.editingSnippet?.content || '';

        // Show in edit area
        this.browserElements.markdownDisplay.classList.add('hidden');
        this.browserElements.markdownEdit.classList.remove('hidden');
        this.browserElements.markdownEdit.value = content;
    },

    // Cancel snippet editing
    cancelSnippetEdit() {
        this.browserState.editingSnippet = null;
        this.browserState.snippetNavStack = [];
        this.browserState.showIncludes = false;

        // Restore pane header
        if (this.browserElements.codePaneLabel) {
            this.browserElements.codePaneLabel.textContent = 'Markdown';
        }

        // Hide parent link
        if (this.browserElements.snippetParentLink) {
            this.browserElements.snippetParentLink.classList.add('hidden');
        }

        // Hide INCLUDES toggle
        if (this.browserElements.includesToggle) {
            this.browserElements.includesToggle.classList.add('hidden');
            this.browserElements.includesToggle.classList.remove('active');
        }

        // Restore markdown display
        this.browserElements.markdownEdit.classList.add('hidden');
        this.browserElements.markdownDisplay.classList.remove('hidden');

        // Hide actions footer
        if (this.browserElements.actionsFooter) {
            this.browserElements.actionsFooter.classList.add('hidden');
        }

        // Restore button handlers
        if (this.browserElements.saveDraftBtn) {
            this.browserElements.saveDraftBtn.textContent = 'Save Draft';
            this.browserElements.saveDraftBtn.onclick = () => this.browserSaveDraft();
        }
        if (this.browserElements.publishBtn) {
            this.browserElements.publishBtn.textContent = 'Publish';
            this.browserElements.publishBtn.onclick = () => this.browserPublish();
        }
        if (this.browserElements.cancelBtn) {
            this.browserElements.cancelBtn.classList.remove('hidden');
        }

        // Remove editing state from edit button
        if (this.browserElements.editBtn) {
            this.browserElements.editBtn.classList.remove('editing');
            this.browserElements.editBtn.innerHTML = '✎ <span>EDIT</span>';
        }
    },

    // Save snippet edit and re-render
    async saveSnippetEdit() {
        const editingSnippet = this.browserState.editingSnippet;
        if (!editingSnippet) return;

        // Get content: if in includes mode, use stored content; otherwise use textarea
        const newContent = this.browserState.showIncludes
            ? editingSnippet.content
            : this.browserElements.markdownEdit.value;

        try {
            // Save the snippet (preserve the source tier from the original include)
            await this.api('PUT', `/api/snippets/${encodeURIComponent(editingSnippet.path)}`, {
                content: newContent,
                source: editingSnippet.source,
            });

            this.showToast('Snippet saved', 'success');

            // Cancel edit mode UI
            this.cancelSnippetEdit();

            // Re-render the page to show updated snippet
            await this.reRenderCurrentPage();

        } catch (err) {
            this.showToast('Failed to save snippet: ' + err.message, 'error');
        }
    },

    // Re-render the current page (calls CLI render)
    async reRenderCurrentPage() {
        const currentUrl = this.browserState.currentUrl;
        if (!currentUrl) return;

        try {
            // Call the render-page endpoint
            await this.api('POST', '/api/render-page', {
                path: currentUrl,
            });

            // Reload the current page content
            await this.browserNavigateTo(currentUrl);

            // Re-enable snippet edit mode if it was on
            if (this.browserState.snippetEditMode) {
                // Small delay to ensure content is loaded
                setTimeout(() => this.enableSnippetEditMode(), 100);
            }
        } catch (err) {
            this.showToast('Failed to re-render page: ' + err.message, 'error');
        }
    },

    // ========================================
    // End Browser Mode Functions
    // ========================================

    // Truncate URL for display
    truncateUrl(url) {
        if (!url) return '';
        let display = url.replace(/^https?:\/\//, '');
        if (display.length > 50) {
            display = display.substring(0, 47) + '...';
        }
        return display;
    },

    // Utility: extract title from markdown (first # heading)
    extractTitleFromMarkdown(markdown) {
        const lines = markdown.split('\n');
        for (const line of lines) {
            const trimmed = line.trim();
            if (trimmed.startsWith('# ')) {
                return trimmed.substring(2).trim();
            }
        }
        return '';
    },

    // Utility: slugify text for filename
    slugify(text) {
        return text
            .toLowerCase()
            .replace(/[^a-z0-9]+/g, '-')
            .replace(/^-+|-+$/g, '')
            .substring(0, 50) || 'untitled';
    },

    // ========================================================================
    // Social features: sidebar mode, feed, following, remote post
    // ========================================================================

    setSidebarMode(mode) {
        this.sidebarMode = mode;
        const mySite = document.getElementById('sidebar-my-site');
        const social = document.getElementById('sidebar-social');

        // Toggle sidebar sections
        if (mode === 'social') {
            mySite.classList.add('hidden');
            social.classList.remove('hidden');
            this.setActiveView('feed');
        } else {
            social.classList.add('hidden');
            mySite.classList.remove('hidden');
            this.setActiveView('posts-published');
        }

        // Update tab active state
        document.querySelectorAll('.sidebar-mode-toggle .mode-tab').forEach(tab => {
            tab.classList.toggle('active', tab.dataset.sidebarMode === mode);
        });
    },

    async renderFeedList(container) {
        try {
            container.innerHTML = '<div class="content-list"><div class="empty-state"><p>Loading feed...</p></div></div>';
            const typeParam = this._feedTypeFilter ? `?type=${this._feedTypeFilter}` : '';
            const result = await this.api('GET', '/api/feed' + typeParam);
            const items = result.items || [];

            this._feedItems = items;
            this.counts.feed = result.total || 0;
            this.counts.feedUnread = result.unread || 0;
            this.updateBadge('feed-count', this.counts.feedUnread, this.counts.feedUnread > 0);

            // Build filter tabs
            const filterHtml = `
                <div class="feed-filter-tabs">
                    <button class="feed-filter-tab ${this._feedTypeFilter === '' ? 'active' : ''}" onclick="App.setFeedTypeFilter('')">All</button>
                    <button class="feed-filter-tab ${this._feedTypeFilter === 'post' ? 'active' : ''}" onclick="App.setFeedTypeFilter('post')">Posts</button>
                    <button class="feed-filter-tab ${this._feedTypeFilter === 'comment' ? 'active' : ''}" onclick="App.setFeedTypeFilter('comment')">Comments</button>
                    <button class="feed-read-toggle ${!this._hideRead ? 'active' : ''}" onclick="App.toggleHideRead(!App._hideRead)">${this._hideRead ? 'Show All' : 'Unread Only'}</button>
                </div>
            `;

            // Stale banner
            let staleHtml = '';
            if (result.stale) {
                staleHtml = `
                    <div class="feed-stale-banner" id="feed-stale-banner">
                        Cache is stale — <a href="#" onclick="event.preventDefault(); App.refreshFeed()">refresh now</a>
                    </div>
                `;
            }

            // Filter out read items if toggle is on
            const displayItems = this._hideRead ? items.filter(item => !item.read_at) : items;

            if (displayItems.length === 0) {
                const emptyMsg = this.counts.feed === 0 && this.counts.following === 0
                    ? `<h3>No authors followed</h3><p>Follow some authors to see their posts here.</p><button class="primary" onclick="App.setSidebarMode('social'); App.setActiveView('following');">Browse Following</button>`
                    : this._hideRead
                    ? `<h3>All caught up</h3><p>No unread items. Toggle "Hide Read" off to see all items.</p>`
                    : `<h3>No items</h3><p>${this._feedTypeFilter ? 'No ' + this._feedTypeFilter + 's in the feed.' : 'No items in the feed yet. Click Refresh to check for new content.'}</p>`;
                container.innerHTML = `${filterHtml}${staleHtml}<div class="content-list"><div class="empty-state">${emptyMsg}</div></div>`;

                // Always background-refresh to pick up new content
                if (!this._feedRefreshing) {
                    this._autoRefreshFeed();
                }
                return;
            }

            container.innerHTML = `
                ${filterHtml}
                ${staleHtml}
                <div class="content-list">
                    ${displayItems.map((item, idx) => {
                        const realIdx = items.indexOf(item);
                        const typeLabel = item.type === 'comment' ? 'Comment' : 'Post';
                        const badgeClass = item.type === 'comment' ? 'feed-type-badge comment' : 'feed-type-badge post';
                        const isUnread = !item.read_at;
                        const unreadClass = isUnread ? ' feed-item-unread' : '';
                        const unreadDot = isUnread ? '<span class="unread-dot"></span>' : '';
                        return `
                            <div class="content-item feed-item${unreadClass}" onclick="App.openFeedItem(${realIdx})">
                                <div class="item-info">
                                    <div class="item-title">${unreadDot}${this.escapeHtml(item.title)}</div>
                                    <div class="item-path">
                                        <span class="${badgeClass}">${typeLabel}</span>
                                        ${this.escapeHtml(item.author_domain)}
                                    </div>
                                </div>
                                <div class="item-date-group">
                                    <span class="item-date">${this.formatDate(item.published)}</span>
                                    <span class="item-time">${this.formatTime(item.published)}</span>
                                </div>
                                <div class="feed-item-actions">
                                    <button class="feed-action-btn" onclick="event.stopPropagation(); App.markFeedUnread('${item.id}')">Mark Unread</button>
                                    <button class="feed-action-btn" onclick="event.stopPropagation(); App.markUnreadFromHere('${item.id}')">Unread From Here</button>
                                </div>
                            </div>
                        `;
                    }).join('')}
                </div>
            `;

            // Always background-refresh to pick up new content
            if (!this._feedRefreshing) {
                this._autoRefreshFeed();
            }
        } catch (err) {
            container.innerHTML = `<div class="content-list"><div class="empty-state"><h3>Failed to load feed</h3><p>${this.escapeHtml(err.message)}</p></div></div>`;
        }
    },

    setFeedTypeFilter(type) {
        this._feedTypeFilter = type;
        const contentList = document.getElementById('content-list');
        if (contentList) this.renderFeedList(contentList);
    },

    toggleHideRead(hideRead) {
        this._hideRead = hideRead;
        // Persist to server
        this.api('POST', '/api/settings/hide-read', { hide_read: hideRead }).catch(err => {
            console.error('Failed to save hide-read setting:', err);
        });
        // Re-render feed
        const contentList = document.getElementById('content-list');
        if (contentList && this.currentView === 'feed') {
            this.renderFeedList(contentList);
        }
    },

    async openFeedItem(idx) {
        const item = this._feedItems && this._feedItems[idx];
        if (!item) return;

        // Fire-and-forget mark read
        if (!item.read_at) {
            item.read_at = new Date().toISOString();
            this.counts.feedUnread = Math.max(0, this.counts.feedUnread - 1);
            this.updateBadge('feed-count', this.counts.feedUnread, this.counts.feedUnread > 0);

            // Update DOM optimistically
            const feedItems = document.querySelectorAll('.feed-item');
            if (feedItems[idx]) {
                feedItems[idx].classList.remove('feed-item-unread');
                const dot = feedItems[idx].querySelector('.unread-dot');
                if (dot) dot.remove();
            }

            this.api('POST', '/api/feed/read', { id: item.id }).catch(() => {});
        }

        this.openRemotePost(item.url, item.author_url, item.title);
    },

    async refreshFeed() {
        if (this._feedRefreshing) return;
        this._feedRefreshing = true;
        this.showToast('Refreshing feed...', 'info', 3000);

        try {
            const result = await this.api('POST', '/api/feed/refresh');
            const newItems = result.new_items || 0;

            this.counts.feed = result.total || 0;
            this.counts.feedUnread = result.unread || 0;
            this.updateBadge('feed-count', this.counts.feedUnread, this.counts.feedUnread > 0);

            if (newItems > 0) {
                this.showToast(`${newItems} new item${newItems > 1 ? 's' : ''}`, 'success');
            } else {
                this.showToast('Conversations up to date', 'success');
            }

            // Re-render if still on feed view
            if (this.currentView === 'feed') {
                const contentList = document.getElementById('content-list');
                if (contentList) await this.renderFeedList(contentList);
            }

            // Update bell dot (feed refresh also syncs notifications server-side)
            this.fetchNotificationCount();
        } catch (err) {
            this.showToast('Refresh failed: ' + err.message, 'error');
        } finally {
            this._feedRefreshing = false;
        }
    },

    _feedPollTimer: null,

    initFeedPolling() {
        this.stopFeedPolling();
        this._feedPollTimer = setInterval(async () => {
            try {
                const counts = await this.api('GET', '/api/feed/counts');
                const prevUnread = this.counts.feedUnread || 0;
                this.counts.feed = counts.total || 0;
                this.counts.feedUnread = counts.unread || 0;
                this.updateBadge('feed-count', this.counts.feedUnread, this.counts.feedUnread > 0);

                if (this.counts.feedUnread !== prevUnread && this.currentView === 'feed') {
                    const contentList = document.getElementById('content-list');
                    if (contentList) await this.renderFeedList(contentList);
                }
            } catch (e) {
                // Silently fail — feed polling is non-critical
            }
        }, 60000);
    },

    stopFeedPolling() {
        if (this._feedPollTimer) {
            clearInterval(this._feedPollTimer);
            this._feedPollTimer = null;
        }
    },

    async _autoRefreshFeed() {
        if (this._feedRefreshing) return;
        this._feedRefreshing = true;

        try {
            const result = await this.api('POST', '/api/feed/refresh');
            const newItems = result.new_items || 0;

            this.counts.feed = result.total || 0;
            this.counts.feedUnread = result.unread || 0;
            this.updateBadge('feed-count', this.counts.feedUnread, this.counts.feedUnread > 0);

            // Remove stale banner
            const banner = document.getElementById('feed-stale-banner');
            if (banner) banner.remove();

            if (newItems > 0) {
                this.showToast(`${newItems} new item${newItems > 1 ? 's' : ''}`, 'success');
                // Re-render if still on feed view
                if (this.currentView === 'feed') {
                    const contentList = document.getElementById('content-list');
                    if (contentList) await this.renderFeedList(contentList);
                }
            }
        } catch (err) {
            console.error('Auto-refresh failed:', err);
        } finally {
            this._feedRefreshing = false;
        }
    },

    async markAllFeedRead() {
        try {
            await this.api('POST', '/api/feed/read', { all: true });
            this.counts.feedUnread = 0;
            this.updateBadge('feed-count', 0);

            // Update all items in memory
            if (this._feedItems) {
                const now = new Date().toISOString();
                this._feedItems.forEach(item => { item.read_at = now; });
            }

            // Re-render
            if (this.currentView === 'feed') {
                const contentList = document.getElementById('content-list');
                if (contentList) await this.renderFeedList(contentList);
            }
            this.showToast('All items marked as read', 'success');
        } catch (err) {
            this.showToast('Failed: ' + err.message, 'error');
        }
    },

    async markFeedUnread(id) {
        try {
            await this.api('POST', '/api/feed/read', { id, unread: true });
            this.counts.feedUnread++;
            this.updateBadge('feed-count', this.counts.feedUnread, true);

            // Update in memory
            if (this._feedItems) {
                const item = this._feedItems.find(i => i.id === id);
                if (item) item.read_at = '';
            }

            // Re-render
            if (this.currentView === 'feed') {
                const contentList = document.getElementById('content-list');
                if (contentList) await this.renderFeedList(contentList);
            }
        } catch (err) {
            this.showToast('Failed: ' + err.message, 'error');
        }
    },

    async markUnreadFromHere(id) {
        try {
            await this.api('POST', '/api/feed/read', { from_id: id });
            // Reload counts since multiple items changed
            const counts = await this.api('GET', '/api/feed/counts');
            this.counts.feedUnread = counts.unread || 0;
            this.updateBadge('feed-count', this.counts.feedUnread, this.counts.feedUnread > 0);

            // Re-render
            if (this.currentView === 'feed') {
                const contentList = document.getElementById('content-list');
                if (contentList) await this.renderFeedList(contentList);
            }
        } catch (err) {
            this.showToast('Failed: ' + err.message, 'error');
        }
    },

    async renderFollowingList(container) {
        try {
            const result = await this.api('GET', '/api/following');
            const follows = result.following || [];

            if (follows.length === 0) {
                container.innerHTML = `
                    <div class="content-list">
                        <div class="empty-state onboarding-empty">
                            <p>Following an author means their posts appear in your Conversations feed
                               and their comments on your site are automatically blessed.</p>
                            <div class="content-item following-item onboarding-follow-card">
                                <div class="item-info">
                                    <div class="item-title">discover.polis.pub</div>
                                </div>
                                <div class="following-item-actions">
                                    <button class="primary" onclick="App.followDiscover()">Follow</button>
                                </div>
                                <div class="onboarding-follow-desc">A community hub that aggregates conversations from across the polis network.</div>
                            </div>
                            <button class="secondary" onclick="App.openFollowPanel()">Follow Another Author</button>
                        </div>
                    </div>
                `;
                return;
            }

            container.innerHTML = `
                <div class="content-list">
                    ${follows.map(f => {
                        const domain = f.url.replace('https://', '').replace('http://', '').replace(/\/$/, '');
                        const title = f.site_title || f.author_name || domain;
                        const subtitle = f.author_name && f.author_name !== title
                            ? `${this.escapeHtml(f.author_name)} · ${this.escapeHtml(domain)}`
                            : this.escapeHtml(domain);
                        const addedAt = f.added_at ? this.formatDate(f.added_at) : '';
                        return `
                            <div class="content-item following-item">
                                <div class="item-info">
                                    <div class="item-title">${this.escapeHtml(title)}</div>
                                    <div class="item-path">${subtitle}</div>
                                </div>
                                <div class="following-item-actions">
                                    ${addedAt ? `<span class="item-date">Followed: ${addedAt}</span>` : ''}
                                    <button class="danger-small" onclick="event.stopPropagation(); App.unfollowAuthor('${this.escapeHtml(f.url)}')">Unfollow</button>
                                </div>
                            </div>
                        `;
                    }).join('')}
                </div>
            `;
        } catch (err) {
            container.innerHTML = `<div class="content-list"><div class="empty-state"><h3>Failed to load following</h3><p>${this.escapeHtml(err.message)}</p></div></div>`;
        }
    },

    openFollowPanel() {
        const panel = document.getElementById('follow-panel');
        const input = document.getElementById('follow-url-input');
        const suggestion = document.getElementById('follow-suggestion');
        if (panel) panel.classList.remove('hidden');
        if (input) { input.value = ''; input.focus(); }
        if (suggestion) {
            if (this.counts.following === 0) {
                suggestion.classList.remove('hidden');
            } else {
                suggestion.classList.add('hidden');
            }
        }
    },

    closeFollowPanel() {
        const panel = document.getElementById('follow-panel');
        if (panel) panel.classList.add('hidden');
    },

    async submitFollow() {
        const input = document.getElementById('follow-url-input');
        const url = (input.value || '').trim();
        if (!url) {
            this.showToast('Please enter a URL', 'error');
            return;
        }
        if (!url.startsWith('https://')) {
            this.showToast('URL must start with https://', 'error');
            return;
        }

        try {
            this.showToast('Following...', 'info', 2000);
            const result = await this.api('POST', '/api/following', { url });
            this.closeFollowPanel();
            if (result.data && result.data.already_followed) {
                this.showToast('Already following this author', 'info');
            } else {
                const blessed = result.data ? result.data.comments_blessed : 0;
                let msg = 'Now following ' + url;
                if (blessed > 0) msg += ` (blessed ${blessed} comment${blessed > 1 ? 's' : ''})`;
                this.showToast(msg, 'success');
            }
            await this.loadAllCounts();
            await this.loadViewContent();
        } catch (err) {
            this.showToast('Failed to follow: ' + err.message, 'error');
        }
    },

    async followDiscover() {
        try {
            this.showToast('Following discover.polis.pub...', 'info', 2000);
            const result = await this.api('POST', '/api/following', { url: 'https://discover.polis.pub/' });
            if (result.data && result.data.already_followed) {
                this.showToast('Already following discover.polis.pub', 'info');
            } else {
                this.showToast('Now following discover.polis.pub', 'success');
            }
            await this.loadAllCounts();
            await this.loadViewContent();
        } catch (err) {
            this.showToast('Failed to follow: ' + err.message, 'error');
        }
    },

    async unfollowAuthor(url) {
        const confirmed = await this.showConfirmModal(
            'Unfollow Author',
            'Are you sure you want to unfollow ' + url + '? Any blessed comments from this author will be denied.',
            'Unfollow',
            'Cancel',
            'danger'
        );
        if (!confirmed) return;

        try {
            const result = await this.api('DELETE', '/api/following', { url });
            const denied = result.data ? result.data.comments_denied : 0;
            let msg = 'Unfollowed ' + url;
            if (denied > 0) msg += ` (denied ${denied} comment${denied > 1 ? 's' : ''})`;
            this.showToast(msg, 'success');
            await this.loadAllCounts();
            await this.loadViewContent();
        } catch (err) {
            this.showToast('Failed to unfollow: ' + err.message, 'error');
        }
    },

    openRemotePostByIndex(idx) {
        const item = this._feedItems && this._feedItems[idx];
        if (!item) return;
        this.openRemotePost(item.url, item.author_url, item.title);
    },

    async openRemotePost(postUrl, authorUrl, title) {
        const panel = document.getElementById('remote-post-panel');
        const titleEl = document.getElementById('remote-post-title');
        const metaEl = document.getElementById('remote-post-meta');
        const bodyEl = document.getElementById('remote-post-body');

        titleEl.textContent = title || 'Remote Post';
        metaEl.innerHTML = '<p>Loading...</p>';
        bodyEl.innerHTML = '';
        panel.classList.remove('hidden');

        // Build the full post URL from relative path + author base
        let fullUrl;
        if (postUrl.startsWith('https://') || postUrl.startsWith('http://')) {
            fullUrl = postUrl;
        } else {
            const base = authorUrl.replace(/\/$/, '');
            const path = postUrl.startsWith('/') ? postUrl : '/' + postUrl;
            fullUrl = base + path;
        }

        // Build a browser-friendly URL for "Open original" (prefer .html over .md)
        let originalUrl = fullUrl;
        if (originalUrl.endsWith('.md')) {
            originalUrl = originalUrl.slice(0, -3) + '.html';
        }

        const domain = authorUrl.replace('https://', '').replace('http://', '').replace(/\/$/, '');
        metaEl.innerHTML = `
            <span class="remote-post-author">${this.escapeHtml(domain)}</span>
            <a href="${this.escapeHtml(originalUrl)}" target="_blank" class="remote-post-link">Open original &#x2197;</a>
        `;

        try {
            const result = await this.api('GET', '/api/remote/post?url=' + encodeURIComponent(fullUrl));
            bodyEl.innerHTML = `<div class="parchment-preview">${result.content}</div>`;
        } catch (err) {
            bodyEl.innerHTML = `<div class="empty-state"><h3>Failed to load post</h3><p>${this.escapeHtml(err.message)}</p><p><a href="${this.escapeHtml(fullUrl)}" target="_blank">Open in new tab</a></p></div>`;
        }
    },

    closeRemotePost() {
        const panel = document.getElementById('remote-post-panel');
        if (panel) panel.classList.add('hidden');
    },

    // Utility: escape HTML
    // ==================== Activity Stream ====================

    _activityCursor: '0',
    _activityEvents: [],
    _activityMaxEvents: 500,

    async renderActivityStream(container) {
        try {
            container.innerHTML = '<div class="content-list"><div class="empty-state"><p>Loading activity...</p></div></div>';
            const result = await this.api('GET', `/api/activity?since=${this._activityCursor}&limit=100`);
            const events = result.events || [];

            if (events.length > 0) {
                this._activityEvents = this._activityEvents.concat(events);
                if (this._activityEvents.length > this._activityMaxEvents) {
                    this._activityEvents = this._activityEvents.slice(
                        this._activityEvents.length - this._activityMaxEvents
                    );
                }
                this._activityCursor = result.cursor || this._activityCursor;
            }

            if (this._activityEvents.length === 0) {
                container.innerHTML = `<div class="content-list"><div class="empty-state">
                    <h3>No activity yet</h3>
                    <p>Follow some authors to see their activity here.</p>
                </div></div>`;
                return;
            }

            const hasMore = result.has_more;
            container.innerHTML = `
                <div class="content-list">
                    ${this._activityEvents.map(evt => this.renderActivityEvent(evt)).join('')}
                </div>
                ${hasMore ? '<div class="activity-load-more"><button class="secondary" onclick="App.loadMoreActivity()">Load More</button></div>' : ''}
            `;
        } catch (err) {
            container.innerHTML = `<div class="content-list"><div class="empty-state"><h3>Failed to load activity</h3><p>${this.escapeHtml(err.message)}</p></div></div>`;
        }
    },

    renderActivityEvent(evt) {
        const typeLabels = {
            'polis.post.published': 'published a post',
            'polis.post.republished': 'republished a post',
            'polis.comment.published': 'published a comment',
            'polis.comment.republished': 'republished a comment',
            'polis.blessing.requested': 'requested a blessing',
            'polis.blessing.granted': 'granted a blessing',
            'polis.blessing.denied': 'denied a blessing',
            'polis.follow.announced': 'followed someone',
            'polis.follow.removed': 'unfollowed someone',
        };
        const actionLabel = typeLabels[evt.type] || evt.type;
        const typeBadge = evt.type.split('.').pop();

        let detail = '';
        if (evt.payload) {
            if (evt.payload.title) {
                detail = `<span class="activity-detail">${this.escapeHtml(evt.payload.title)}</span>`;
            } else if (evt.payload.post_url) {
                detail = `<span class="activity-detail">${this.escapeHtml(evt.payload.post_url)}</span>`;
            } else if (evt.payload.target_domain) {
                detail = `<span class="activity-detail">${this.escapeHtml(evt.payload.target_domain)}</span>`;
            }
        }

        return `
            <div class="content-item activity-event">
                <div class="item-info">
                    <div class="item-title">
                        <span class="activity-actor">${this.escapeHtml(evt.actor)}</span>
                        ${actionLabel}
                    </div>
                    <div class="item-path">
                        <span class="activity-type-badge">${this.escapeHtml(typeBadge)}</span>
                        ${detail}
                    </div>
                </div>
                <div class="item-date-group">
                    <span class="item-date">${this.formatDate(evt.timestamp)}</span>
                    <span class="item-time">${this.formatTime(evt.timestamp)}</span>
                </div>
            </div>
        `;
    },

    async refreshActivity() {
        const contentList = document.getElementById('content-list');
        if (contentList) await this.renderActivityStream(contentList);
    },

    async resetActivity() {
        this._activityCursor = '0';
        this._activityEvents = [];
        const contentList = document.getElementById('content-list');
        if (contentList) await this.renderActivityStream(contentList);
    },

    async loadMoreActivity() {
        const contentList = document.getElementById('content-list');
        if (contentList) await this.renderActivityStream(contentList);
    },

    // ==================== Followers ====================

    async renderFollowersList(container) {
        try {
            container.innerHTML = '<div class="content-list"><div class="empty-state"><p>Loading followers...</p></div></div>';
            const result = await this.api('GET', '/api/followers/count');
            const followers = result.followers || [];
            const count = result.count || 0;

            this.counts.followers = count;
            this.updateBadge('followers-count', count);

            if (followers.length === 0) {
                container.innerHTML = `<div class="content-list"><div class="empty-state">
                    <h3>No followers yet</h3>
                    <p>When other polis authors follow you, they'll appear here.</p>
                </div></div>`;
                return;
            }

            container.innerHTML = `
                <div class="content-list">
                    <div class="followers-summary">${count} follower${count !== 1 ? 's' : ''}</div>
                    ${followers.map(domain => `
                        <div class="content-item follower-item">
                            <div class="item-info">
                                <div class="item-title">${this.escapeHtml(domain)}</div>
                            </div>
                        </div>
                    `).join('')}
                </div>
            `;
        } catch (err) {
            container.innerHTML = `<div class="content-list"><div class="empty-state"><h3>Failed to load followers</h3><p>${this.escapeHtml(err.message)}</p></div></div>`;
        }
    },

    async refreshFollowers(fullRefresh) {
        if (fullRefresh) {
            const contentList = document.getElementById('content-list');
            if (contentList) {
                contentList.innerHTML = '<div class="content-list"><div class="empty-state"><p>Refreshing followers...</p></div></div>';
                try {
                    const result = await this.api('GET', '/api/followers/count?refresh=true');
                    this.counts.followers = result.count || 0;
                    this.updateBadge('followers-count', this.counts.followers);
                    await this.renderFollowersList(contentList);
                } catch (err) {
                    contentList.innerHTML = `<div class="content-list"><div class="empty-state"><h3>Refresh failed</h3><p>${this.escapeHtml(err.message)}</p></div></div>`;
                }
            }
        } else {
            const contentList = document.getElementById('content-list');
            if (contentList) await this.renderFollowersList(contentList);
        }
    },

    escapeHtml(text) {
        if (!text) return '';
        const div = document.createElement('div');
        div.textContent = text;
        return div.innerHTML;
    },

    // Utility: format date
    formatDate(isoString) {
        if (!isoString) return '';
        const date = new Date(isoString);
        return date.toLocaleDateString('en-US', {
            month: 'short',
            day: 'numeric',
            year: 'numeric'
        });
    },

    // Utility: format time in local timezone
    formatTime(isoString) {
        if (!isoString) return '';
        const date = new Date(isoString);
        return date.toLocaleTimeString('en-US', {
            hour: 'numeric',
            minute: '2-digit',
            hour12: true
        });
    },

    // --- Notification Methods ---

    notificationState: { unreadCount: 0, pollTimer: null, showAll: false, offset: 0, hasMore: true },

    updateDomainDisplay(baseUrl) {
        const el = document.getElementById('domain-display');
        if (!el) return;
        if (!baseUrl) { el.textContent = ''; return; }
        el.textContent = baseUrl.replace(/^https?:\/\//, '');
    },

    initNotifications() {
        this.fetchNotificationCount();
        if (this.notificationState.pollTimer) {
            clearInterval(this.notificationState.pollTimer);
        }
        this.notificationState.pollTimer = setInterval(() => {
            this.fetchNotificationCount();
        }, 60000);
    },

    async fetchNotificationCount() {
        try {
            const resp = await this.api('GET', '/api/notifications/count');
            this.notificationState.unreadCount = resp.unread || 0;

            const dot = document.getElementById('notification-dot');
            if (dot) {
                dot.classList.toggle('hidden', this.notificationState.unreadCount === 0);
            }
        } catch (e) {
            // Silently fail — notifications are non-critical
        }
    },

    async toggleNotifications() {
        const panel = document.getElementById('notification-panel');
        if (!panel) return;

        if (!panel.classList.contains('hidden')) {
            this.closeNotifications();
            return;
        }

        panel.classList.remove('hidden');
        this.notificationState.offset = 0;
        this.notificationState.hasMore = true;
        this.notificationState.showAll = false;
        document.getElementById('notification-toggle-all').textContent = 'Show All';
        document.getElementById('notification-toggle-all').classList.remove('active');

        await this.loadNotifications(false);
    },

    closeNotifications() {
        const panel = document.getElementById('notification-panel');
        if (panel) panel.classList.add('hidden');
    },

    async toggleAllNotifications() {
        this.notificationState.showAll = !this.notificationState.showAll;
        this.notificationState.offset = 0;
        this.notificationState.hasMore = true;

        const btn = document.getElementById('notification-toggle-all');
        if (this.notificationState.showAll) {
            btn.textContent = 'Unread Only';
            btn.classList.add('active');
        } else {
            btn.textContent = 'Show All';
            btn.classList.remove('active');
        }

        await this.loadNotifications(false);
    },

    async loadNotifications(append) {
        const list = document.getElementById('notification-list');
        if (!list) return;

        if (!append) {
            list.innerHTML = '<div class="notification-loading">Loading...</div>';
            this.notificationState.offset = 0;
        }

        const includeRead = this.notificationState.showAll;
        const limit = 20;
        const offset = this.notificationState.offset;

        try {
            const resp = await this.api('GET',
                `/api/notifications?offset=${offset}&limit=${limit}&include_read=${includeRead}`);
            const items = resp.notifications || [];

            if (!append) {
                list.innerHTML = '';
            } else {
                // Remove loading indicator
                const loader = list.querySelector('.notification-loading');
                if (loader) loader.remove();
            }

            if (items.length === 0 && offset === 0) {
                list.innerHTML = '<div class="notification-empty">No notifications</div>';
                this.notificationState.hasMore = false;
                return;
            }

            items.forEach(n => {
                list.appendChild(this.renderNotification(n));
            });

            this.notificationState.offset += items.length;
            this.notificationState.hasMore = this.notificationState.offset < resp.total;

            // Mark displayed unread notifications as read
            const unreadIds = items.filter(n => !n.read_at).map(n => n.id);
            if (unreadIds.length > 0) {
                this.api('POST', '/api/notifications/read', { ids: unreadIds })
                    .then(() => this.fetchNotificationCount())
                    .catch(() => {});
            }

            // Set up infinite scroll
            if (this.notificationState.hasMore) {
                list.onscroll = () => {
                    if (list.scrollTop + list.clientHeight >= list.scrollHeight - 50) {
                        if (this.notificationState.hasMore) {
                            list.onscroll = null; // Prevent duplicate triggers
                            list.insertAdjacentHTML('beforeend',
                                '<div class="notification-loading">Loading more...</div>');
                            this.loadNotifications(true);
                        }
                    }
                };
            }
        } catch (e) {
            if (!append) {
                list.innerHTML = '<div class="notification-empty">Failed to load notifications</div>';
            }
        }
    },

    renderNotification(n) {
        const div = document.createElement('div');
        div.className = 'notification-item' + (n.read_at ? '' : ' unread');

        const icon = n.icon || '\u2139';
        const ruleId = n.rule_id || '';

        div.innerHTML = `
            <div class="notification-type-badge">${icon}</div>
            <div class="notification-body">
                <div class="notification-message">${this.escapeHtml(n.message || '')}</div>
                <div class="notification-meta">${this.formatRelativeTime(n.created_at)}</div>
            </div>
        `;

        // Click handler for blessing requests
        if (ruleId === 'blessing-requested') {
            div.onclick = () => {
                this.closeNotifications();
                this.switchView('blessing-requests');
            };
        }

        return div;
    },

    // --- Setup Wizard ---

    async openSetupWizard() {
        // Auto-detect current step (2-step: Deploy → Register)
        try {
            const result = await this.api('GET', '/api/site/deploy-check');
            if (!result.deployed) {
                this.setupWizardStep = 0; // Deploy
            } else if (!this.siteRegistered) {
                this.setupWizardStep = 1; // Register
            } else {
                return; // All done, don't open
            }
        } catch {
            this.setupWizardStep = 0; // Default to deploy step on error
        }

        this.renderSetupWizard();
        document.getElementById('setup-wizard-panel').classList.remove('hidden');
    },

    renderSetupWizard() {
        const stepsEl = document.getElementById('setup-wizard-steps');
        const contentEl = document.getElementById('setup-wizard-content');
        const actionBtn = document.getElementById('setup-wizard-action-btn');
        const steps = ['Deploy', 'Register'];

        // Render step indicators
        stepsEl.innerHTML = steps.map((label, i) => {
            let cls = 'setup-step';
            if (i < this.setupWizardStep) cls += ' completed';
            else if (i === this.setupWizardStep) cls += ' active';
            else cls += ' pending';
            const icon = i < this.setupWizardStep ? '&#10003;' : (i + 1);
            return `<div class="${cls}"><span class="step-dot">${icon}</span><span class="step-label">${label}</span></div>` +
                   (i < steps.length - 1 ? '<div class="step-line"></div>' : '');
        }).join('');

        // Render step content
        if (this.setupWizardStep === 0) {
            const domain = this.siteBaseUrl ? new URL(this.siteBaseUrl).hostname : 'yourdomain.com';
            contentEl.innerHTML = `
                <div class="wizard-section">
                    <p>Push your site files so that <strong>${this.escapeHtml(domain)}</strong> serves them publicly. Polis works with any static host.</p>
                    <div class="deploy-example">
                        <div class="deploy-example-header">Example: Git-based deploy</div>
                        <pre class="setup-code"><span class="code-comment"># From your site directory</span>
git add -A .
git commit -m "initial polis site"
git push</pre>
                    </div>
                    <p class="hint">Works with GitHub Pages, Netlify, Vercel, Cloudflare Pages, or any host that serves static files. The key file is <code>.well-known/polis</code> &mdash; once that's reachable, you're live.</p>
                    <div id="deploy-status" class="deploy-status">
                        <span class="deploy-spinner"></span>
                        <span id="deploy-status-text">Checking if your site is live...</span>
                    </div>
                </div>`;
            actionBtn.textContent = 'Next';
            actionBtn.disabled = true;
            this.startDeployPolling();
        } else if (this.setupWizardStep === 1) {
            const domain = this.siteBaseUrl ? new URL(this.siteBaseUrl).hostname : '';
            contentEl.innerHTML = `
                <div class="wizard-section">
                    <p>Register <strong>${this.escapeHtml(domain)}</strong> with the discovery network so others can find and interact with your content.</p>
                </div>`;
            actionBtn.textContent = 'Register';
        }
    },

    startDeployPolling() {
        this.stopDeployPolling();
        const poll = async () => {
            try {
                const result = await this.api('GET', '/api/site/deploy-check');
                const statusText = document.getElementById('deploy-status-text');
                if (result.deployed) {
                    this.stopDeployPolling();
                    if (statusText) statusText.textContent = 'Your site is live!';
                    const statusEl = document.getElementById('deploy-status');
                    if (statusEl) statusEl.classList.add('deployed');
                    const actionBtn = document.getElementById('setup-wizard-action-btn');
                    if (actionBtn) {
                        actionBtn.disabled = false;
                        actionBtn.textContent = 'Next';
                    }
                } else if (statusText) {
                    statusText.textContent = 'Waiting for your site to go live...';
                }
            } catch {
                // Silently continue polling
            }
        };
        poll();
        this.setupWizardDeployTimer = setInterval(poll, 5000);
    },

    stopDeployPolling() {
        if (this.setupWizardDeployTimer) {
            clearInterval(this.setupWizardDeployTimer);
            this.setupWizardDeployTimer = null;
        }
    },

    async setupWizardAction() {
        const actionBtn = document.getElementById('setup-wizard-action-btn');
        if (this.setupWizardStep === 0) {
            // Deploy step → advance to Register
            this.stopDeployPolling();
            this.setupWizardStep = 1;
            this.renderSetupWizard();
        } else if (this.setupWizardStep === 1) {
            // Register
            if (actionBtn) {
                actionBtn.disabled = true;
                actionBtn.textContent = 'Registering...';
            }
            try {
                await this.api('POST', '/api/site/register');
                this.siteRegistered = true;
                this.setupWizardDismissed = true;
                // Dismiss the wizard
                document.getElementById('setup-wizard-panel').classList.add('hidden');
                document.getElementById('setup-banner').classList.add('hidden');
                this.showToast('Site registered with discovery network!', 'success');
                // Persist dismissal
                try { await this.api('POST', '/api/site/setup-wizard-dismiss'); } catch {}
            } catch (err) {
                this.showToast('Registration failed: ' + err.message, 'error');
                if (actionBtn) {
                    actionBtn.disabled = false;
                    actionBtn.textContent = 'Register';
                }
            }
        }
    },

    dismissSetupWizard() {
        this.stopDeployPolling();
        document.getElementById('setup-wizard-panel').classList.add('hidden');
        // Persist dismissal
        this.api('POST', '/api/site/setup-wizard-dismiss').catch(() => {});
        this.setupWizardDismissed = true;
        // Show the banner on dashboard if not registered
        if (!this.siteRegistered) {
            document.getElementById('setup-banner').classList.remove('hidden');
        }
    },

    dismissSetupBanner() {
        document.getElementById('setup-banner').classList.add('hidden');
        // Also persist wizard dismissal
        this.api('POST', '/api/site/setup-wizard-dismiss').catch(() => {});
        this.setupWizardDismissed = true;
    },

    async checkSetupBanner() {
        // Check if wizard dismissed and site registered
        try {
            const settings = await this.api('GET', '/api/settings');
            this.setupWizardDismissed = settings.setup_wizard_dismissed || false;

            // Check registration status
            try {
                const regStatus = await this.api('GET', '/api/site/registration-status');
                this.siteRegistered = regStatus.is_registered || false;
            } catch {
                this.siteRegistered = false;
            }

            // Show banner if not dismissed AND not registered
            if (!this.setupWizardDismissed && !this.siteRegistered) {
                // Auto-open wizard on first load after init
                this.openSetupWizard();
            } else if (this.setupWizardDismissed && !this.siteRegistered) {
                document.getElementById('setup-banner').classList.remove('hidden');
            }
        } catch {
            // Can't check, don't show banner
        }
    },

    escapeHtml(str) {
        if (!str) return '';
        const div = document.createElement('div');
        div.textContent = str;
        return div.innerHTML;
    },

    formatRelativeTime(isoString) {
        if (!isoString) return '';
        const date = new Date(isoString);
        const now = new Date();
        const diffMs = now - date;
        const diffMin = Math.floor(diffMs / 60000);
        const diffHour = Math.floor(diffMs / 3600000);
        const diffDay = Math.floor(diffMs / 86400000);

        if (diffMin < 1) return 'just now';
        if (diffMin < 60) return `${diffMin} min ago`;
        if (diffHour < 24) return `${diffHour} hour${diffHour > 1 ? 's' : ''} ago`;
        if (diffDay < 2) return 'yesterday';
        if (diffDay < 7) return `${diffDay} days ago`;
        return this.formatDate(isoString);
    },
};

// Start the app
document.addEventListener('DOMContentLoaded', () => App.init());
