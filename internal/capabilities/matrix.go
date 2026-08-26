// Copyright (c) 2026 Santusht Kotai
// SPDX-License-Identifier: MIT

package capabilities

import (
	"strings"
)

// AccessTier categorizes LinkedIn API access permissions
type AccessTier string

const (
	TierSelfService   AccessTier = "Self-Service (Open)"
	TierCommunityMgmt AccessTier = "Community Management (Approved)"
	TierEnterprise    AccessTier = "Enterprise / Partner Approved"
	TierAdvertising   AccessTier = "LinkedIn Marketing Developer Platform"
)

// Capability defines a specific LinkedIn API feature, its required scopes, and status
type Capability struct {
	ID          string     `json:"id" yaml:"id"`
	Name        string     `json:"name" yaml:"name"`
	Category    string     `json:"category" yaml:"category"`
	Description string     `json:"description" yaml:"description"`
	RequiredScopes []string `json:"required_scopes" yaml:"required_scopes"`
	Tier        AccessTier `json:"tier" yaml:"tier"`
	DocsURL     string     `json:"docs_url" yaml:"docs_url"`
}

// CapabilityStatus wraps a capability with evaluation against active user scopes
type CapabilityStatus struct {
	Capability Capability `json:"capability" yaml:"capability"`
	Available  bool       `json:"available" yaml:"available"`
	MissingScopes []string `json:"missing_scopes,omitempty" yaml:"missing_scopes,omitempty"`
}

// Global registry of LinkedIn API capabilities as of 2026
var Registry = []Capability{
	// Profile & Identity
	{
		ID:          "profile.read",
		Name:        "Member Profile (Basic)",
		Category:    "Profile",
		Description: "Read member name, vanity URL, headline, and profile picture",
		RequiredScopes: []string{"openid", "profile"},
		Tier:        TierSelfService,
		DocsURL:     "https://learn.microsoft.com/en-us/linkedin/consumer/integrations/self-serve/sign-in-with-linkedin",
	},
	{
		ID:          "profile.email",
		Name:        "Member Email Address",
		Category:    "Profile",
		Description: "Read member verified email address",
		RequiredScopes: []string{"email"},
		Tier:        TierSelfService,
		DocsURL:     "https://learn.microsoft.com/en-us/linkedin/consumer/integrations/self-serve/sign-in-with-linkedin",
	},

	// Posts & Content
	{
		ID:          "posts.create",
		Name:        "Create Member Posts",
		Category:    "Posts",
		Description: "Publish text, images, videos, documents, articles, and polls on member feed",
		RequiredScopes: []string{"w_member_social"},
		Tier:        TierSelfService,
		DocsURL:     "https://learn.microsoft.com/en-us/linkedin/marketing/community-management/shares/posts-api",
	},
	{
		ID:          "posts.edit",
		Name:        "Edit Member Posts",
		Category:    "Posts",
		Description: "Modify existing posts commentary and distribution settings",
		RequiredScopes: []string{"w_member_social"},
		Tier:        TierSelfService,
		DocsURL:     "https://learn.microsoft.com/en-us/linkedin/marketing/community-management/shares/posts-api",
	},
	{
		ID:          "posts.delete",
		Name:        "Delete Member Posts",
		Category:    "Posts",
		Description: "Delete posts authored by authenticated member",
		RequiredScopes: []string{"w_member_social"},
		Tier:        TierSelfService,
		DocsURL:     "https://learn.microsoft.com/en-us/linkedin/marketing/community-management/shares/posts-api",
	},
	{
		ID:          "posts.read",
		Name:        "Read Member Posts & Feed",
		Category:    "Posts",
		Description: "Retrieve past posts, feed activities, and post history",
		RequiredScopes: []string{"r_member_social"},
		Tier:        TierCommunityMgmt,
		DocsURL:     "https://learn.microsoft.com/en-us/linkedin/marketing/community-management/shares/posts-api#find-posts-by-authors",
	},

	// Comments & Reactions
	{
		ID:          "comments.create",
		Name:        "Create & Reply to Comments",
		Category:    "Social",
		Description: "Add comments and nested replies to posts",
		RequiredScopes: []string{"w_member_social"},
		Tier:        TierSelfService,
		DocsURL:     "https://learn.microsoft.com/en-us/linkedin/marketing/community-management/shares/comments-api",
	},
	{
		ID:          "comments.read",
		Name:        "Read Comments",
		Category:    "Social",
		Description: "List comments and conversation threads on posts",
		RequiredScopes: []string{"r_member_social"},
		Tier:        TierCommunityMgmt,
		DocsURL:     "https://learn.microsoft.com/en-us/linkedin/marketing/community-management/shares/comments-api",
	},
	{
		ID:          "reactions.create",
		Name:        "React to Posts (Like, Celebrate, etc.)",
		Category:    "Social",
		Description: "Add emotional reactions (LIKE, PRAISE, APPRECIATION, EMPATHY, INTEREST, ENTERTAINMENT)",
		RequiredScopes: []string{"w_member_social"},
		Tier:        TierSelfService,
		DocsURL:     "https://learn.microsoft.com/en-us/linkedin/marketing/community-management/shares/reactions-api",
	},
	{
		ID:          "reactions.read",
		Name:        "Read Social Actions & Summaries",
		Category:    "Social",
		Description: "Retrieve reaction counts, comment counts, and like statuses",
		RequiredScopes: []string{"r_member_social"},
		Tier:        TierCommunityMgmt,
		DocsURL:     "https://learn.microsoft.com/en-us/linkedin/marketing/community-management/shares/social-actions-api",
	},

	// Media Upload
	{
		ID:          "media.upload",
		Name:        "Upload Media Assets (Image, Video, Document)",
		Category:    "Media",
		Description: "Initialize 3-step media upload protocol and store digital assets",
		RequiredScopes: []string{"w_member_social"},
		Tier:        TierSelfService,
		DocsURL:     "https://learn.microsoft.com/en-us/linkedin/marketing/community-management/shares/images-api",
	},

	// Analytics
	{
		ID:          "analytics.posts",
		Name:        "Member Post Analytics",
		Category:    "Analytics",
		Description: "Query post impressions, reach, engagement, clicks, and share counts",
		RequiredScopes: []string{"r_member_postAnalytics"},
		Tier:        TierCommunityMgmt,
		DocsURL:     "https://learn.microsoft.com/en-us/linkedin/marketing/community-management/analytics/post-analytics",
	},
	{
		ID:          "analytics.profile",
		Name:        "Member Profile Analytics",
		Category:    "Analytics",
		Description: "Query profile views, follower growth, search appearances",
		RequiredScopes: []string{"r_member_profileAnalytics"},
		Tier:        TierCommunityMgmt,
		DocsURL:     "https://learn.microsoft.com/en-us/linkedin/marketing/community-management/analytics/profile-analytics",
	},

	// Organizations & Pages
	{
		ID:          "organization.posts",
		Name:        "Manage Company Pages & Post as Org",
		Category:    "Organization",
		Description: "Publish posts, update details, and view follower demographics for Company Pages",
		RequiredScopes: []string{"w_organization_social", "r_organization_social"},
		Tier:        TierEnterprise,
		DocsURL:     "https://learn.microsoft.com/en-us/linkedin/marketing/community-management/organizations/organization-access-control",
	},
	{
		ID:          "organization.analytics",
		Name:        "Organization Page Analytics",
		Category:    "Organization",
		Description: "Read organization engagement metrics and visitor demographics",
		RequiredScopes: []string{"r_organization_social"},
		Tier:        TierEnterprise,
		DocsURL:     "https://learn.microsoft.com/en-us/linkedin/marketing/community-management/organizations/page-statistics",
	},

	// Events
	{
		ID:          "events.manage",
		Name:        "LinkedIn Events Management",
		Category:    "Events",
		Description: "Create, manage, and query attendees for LinkedIn audio/video events",
		RequiredScopes: []string{"rw_events"},
		Tier:        TierEnterprise,
		DocsURL:     "https://learn.microsoft.com/en-us/linkedin/marketing/integrations/events-management",
	},

	// Messaging & DMs
	{
		ID:          "messaging.dms",
		Name:        "1-to-1 Direct Messages & InMail",
		Category:    "Messaging",
		Description: "Read member private message threads and send InMail messages",
		RequiredScopes: []string{"r_messages", "w_messages"},
		Tier:        TierEnterprise,
		DocsURL:     "https://learn.microsoft.com/en-us/linkedin/talent/integrations/recruiter/1-1-messaging",
	},

	// Network & Connections
	{
		ID:          "network.connections",
		Name:        "Member Connections & 1st-Degree Network",
		Category:    "Network",
		Description: "Read member connection lists and network relationships",
		RequiredScopes: []string{"r_network"},
		Tier:        TierEnterprise,
		DocsURL:     "https://learn.microsoft.com/en-us/linkedin/consumer/integrations/self-serve/sign-in-with-linkedin",
	},

	// Advertising
	{
		ID:          "ads.campaigns",
		Name:        "Advertising Campaigns & Creatives",
		Category:    "Advertising",
		Description: "Manage ad accounts, target audiences, campaigns, and creatives",
		RequiredScopes: []string{"rw_ads"},
		Tier:        TierAdvertising,
		DocsURL:     "https://learn.microsoft.com/en-us/linkedin/marketing/integrations/ads-reporting",
	},
}

// EvaluateCapabilities checks which capabilities are satisfied by granted scopes
func EvaluateCapabilities(grantedScopes []string) []CapabilityStatus {
	scopeSet := make(map[string]bool)
	for _, s := range grantedScopes {
		scopeSet[strings.TrimSpace(s)] = true
	}

	var results []CapabilityStatus
	for _, cap := range Registry {
		var missing []string
		for _, req := range cap.RequiredScopes {
			if !scopeSet[req] {
				missing = append(missing, req)
			}
		}

		results = append(results, CapabilityStatus{
			Capability:    cap,
			Available:     len(missing) == 0,
			MissingScopes: missing,
		})
	}

	return results
}

// CheckCapability tests a single capability by ID
func CheckCapability(id string, grantedScopes []string) (bool, *Capability, []string) {
	for _, cap := range Registry {
		if cap.ID == id {
			scopeSet := make(map[string]bool)
			for _, s := range grantedScopes {
				scopeSet[strings.TrimSpace(s)] = true
			}

			var missing []string
			for _, req := range cap.RequiredScopes {
				if !scopeSet[req] {
					missing = append(missing, req)
				}
			}
			return len(missing) == 0, &cap, missing
		}
	}
	return false, nil, nil
}
