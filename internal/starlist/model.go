package starlist

// List is a GitHub star list owned by a user.
type List struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Slug        string `json:"slug"`
	Description string `json:"description"`
	IsPrivate   bool   `json:"isPrivate"`
	RepoCount   int    `json:"repoCount"`
	UpdatedAt   string `json:"updatedAt"`
}

// Repo is a repository as it appears in a star list or in the starred repositories.
type Repo struct {
	NameWithOwner  string `json:"nameWithOwner"`
	Description    string `json:"description"`
	URL            string `json:"url"`
	StargazerCount int    `json:"stargazerCount"`
	Language       string `json:"language"`
	IsArchived     bool   `json:"isArchived"`
	// StarredAt is only known when the repository comes from the starred
	// repositories connection; star lists do not expose it.
	StarredAt string `json:"starredAt,omitempty"`
}

// Visibility renders the list visibility the way the GitHub UI labels it.
func (l List) Visibility() string {
	if l.IsPrivate {
		return "private"
	}
	return "public"
}
