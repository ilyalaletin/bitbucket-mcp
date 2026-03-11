package bitbucket

// PagedResponse is the generic wrapper for paginated Bitbucket responses.
type PagedResponse[T any] struct {
	Size          int  `json:"size"`
	Limit         int  `json:"limit"`
	IsLastPage    bool `json:"isLastPage"`
	Start         int  `json:"start"`
	NextPageStart int  `json:"nextPageStart"`
	Values        []T  `json:"values"`
}

type PullRequest struct {
	ID          int        `json:"id"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	State       string     `json:"state"`
	CreatedDate int64      `json:"createdDate"`
	UpdatedDate int64      `json:"updatedDate"`
	Author      PRUser     `json:"author"`
	Reviewers   []PRUser   `json:"reviewers"`
	FromRef     PRRef      `json:"fromRef"`
	ToRef       PRRef      `json:"toRef"`
	Links       Links      `json:"links"`
}

type PRUser struct {
	User     User   `json:"user"`
	Role     string `json:"role"`
	Approved bool   `json:"approved"`
	Status   string `json:"status"`
}

type User struct {
	Name        string `json:"name"`
	DisplayName string `json:"displayName"`
	Email       string `json:"emailAddress"`
}

type PRRef struct {
	ID           string     `json:"id"`
	DisplayID    string     `json:"displayId"`
	LatestCommit string     `json:"latestCommit"`
	Repository   Repository `json:"repository"`
}

type Repository struct {
	Slug    string  `json:"slug"`
	Project Project `json:"project"`
}

type Project struct {
	Key  string `json:"key"`
	Name string `json:"name"`
}

type Links struct {
	Self []Link `json:"self"`
}

type Link struct {
	Href string `json:"href"`
}

type Commit struct {
	ID              string        `json:"id"`
	DisplayID       string        `json:"displayId"`
	Message         string        `json:"message"`
	Author          CommitAuthor  `json:"author"`
	AuthorTimestamp int64         `json:"authorTimestamp"`
}

type CommitAuthor struct {
	Name  string `json:"name"`
	Email string `json:"emailAddress"`
}

type BuildStatus struct {
	State       string `json:"state"`
	Key         string `json:"key"`
	Name        string `json:"name"`
	URL         string `json:"url"`
	Description string `json:"description"`
	DateAdded   int64  `json:"dateAdded"`
}

type BrowseResponse struct {
	Path     BrowsePath                 `json:"path"`
	Children *PagedResponse[BrowseEntry] `json:"children,omitempty"`
	Lines    []BrowseLine               `json:"lines,omitempty"`
	Binary   bool                       `json:"binary,omitempty"`
}

type BrowsePath struct {
	Components []string `json:"components"`
	Name       string   `json:"name"`
}

type BrowseEntry struct {
	Path      BrowsePath `json:"path"`
	ContentID string     `json:"contentId"`
	Type      string     `json:"type"` // "FILE" or "DIRECTORY"
	Size      int64      `json:"size,omitempty"`
}

type BrowseLine struct {
	Text string `json:"text"`
}

// Note: Diff endpoints return raw text (not JSON), handled by getRaw with 1MB truncation.
