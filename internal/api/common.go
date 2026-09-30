package api

// CouchbaseAuditData is common audit metadata returned by Capella APIs.
type CouchbaseAuditData struct {
	CreatedBy  string `json:"createdBy,omitempty"`
	CreatedAt  string `json:"createdAt,omitempty"`
	ModifiedBy string `json:"modifiedBy,omitempty"`
	ModifiedAt string `json:"modifiedAt,omitempty"`
	Version    int    `json:"version,omitempty"`
}

// Cursor is pagination metadata from list endpoints.
type Cursor struct {
	Pages Pages `json:"pages"`
	Hrefs Hrefs `json:"hrefs"`
}

// Pages describes pagination page numbers.
type Pages struct {
	Page       int `json:"page"`
	Next       int `json:"next"`
	Previous   int `json:"previous"`
	Last       int `json:"last"`
	PerPage    int `json:"perPage"`
	TotalItems int `json:"totalItems"`
}

// Hrefs describes pagination links.
type Hrefs struct {
	First    string `json:"first"`
	Last     string `json:"last"`
	Previous string `json:"previous"`
	Next     string `json:"next"`
}
