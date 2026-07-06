package tui

import "time"

const (
	kindLap   = "lap"
	kindStint = "stint"
)

type Snapshot struct {
	Missing bool
	State   string
	Counts  Counts
	Claim   Claim
	Gate    *Gate
	Entries []Entry
}

type Counts struct {
	Todo  int `json:"todo"`
	Done  int `json:"done"`
	Total int `json:"total"`
}

type Claim struct {
	Valid      bool    `json:"valid"`
	Lap        string  `json:"lap"`
	File       string  `json:"file"`
	ClaimedAt  *string `json:"claimedAt"`
	AgeSeconds *int64  `json:"ageSeconds"`
}

type Gate struct {
	State   string `json:"state"`
	Stint   string `json:"stint,omitempty"`
	Scope   string `json:"scope,omitempty"`
	File    string `json:"file,omitempty"`
	Message string `json:"message,omitempty"`
}

type Entry struct {
	Kind     string
	ID       string
	Ref      string
	Title    string
	Assignee string
	IsDone   bool
	Order    int
	Stint    *Stint
	Laps     []Entry
}

type Stint struct {
	Name     string
	Scope    string
	File     string
	Todo     int
	Done     int
	Total    int
	Queued   bool
	Archived bool
	Active   bool
	Laps     []Entry
}

type statusSnapshot struct {
	File        string        `json:"file"`
	State       string        `json:"state"`
	Counts      Counts        `json:"counts"`
	Head        *statusHead   `json:"head"`
	Claim       Claim         `json:"claim"`
	Gate        *Gate         `json:"gate,omitempty"`
	Assignees   []assignee    `json:"assignees"`
	ActiveStint *statusStint  `json:"activeStint"`
	Stints      []statusStint `json:"stints"`
}

type statusHead struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	Assignee string `json:"assignee,omitempty"`
}

type assignee struct {
	Assignee string `json:"assignee"`
	Todo     int    `json:"todo"`
}

type statusStint struct {
	Name     string `json:"name"`
	Scope    string `json:"scope"`
	File     string `json:"file"`
	Todo     int    `json:"todo"`
	Done     int    `json:"done"`
	Total    int    `json:"total"`
	Queued   bool   `json:"queued"`
	Archived bool   `json:"archived"`
	Active   bool   `json:"active"`
}

type listResponse struct {
	Tasks []task `json:"tasks"`
}

type task struct {
	Kind        string     `json:"kind,omitempty"`
	ID          string     `json:"id"`
	Ref         string     `json:"ref,omitempty"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	Assignee    string     `json:"assignee,omitempty"`
	IsDone      bool       `json:"isDone"`
	Order       int        `json:"order"`
	CreatedAt   time.Time  `json:"createdAt"`
	UpdatedAt   time.Time  `json:"updatedAt"`
	CompletedAt *time.Time `json:"completedAt"`
}
