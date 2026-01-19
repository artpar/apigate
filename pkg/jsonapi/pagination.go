package jsonapi

import (
	"net/url"
	"strconv"
)

// Pagination holds pagination information for generating links and metadata.
type Pagination struct {
	Total   int64  // Total number of items
	Page    int    // Current page number (1-based)
	PerPage int    // Items per page
	BaseURL string // Base URL for generating links
}

// NewPagination creates a new Pagination instance.
func NewPagination(total int64, page, perPage int, baseURL string) *Pagination {
	if page < 1 {
		page = 1
	}
	if perPage < 1 {
		perPage = 20
	}
	return &Pagination{
		Total:   total,
		Page:    page,
		PerPage: perPage,
		BaseURL: baseURL,
	}
}

// TotalPages returns the total number of pages.
func (p *Pagination) TotalPages() int {
	if p.Total == 0 {
		return 1
	}
	pages := int((p.Total + int64(p.PerPage) - 1) / int64(p.PerPage))
	if pages < 1 {
		pages = 1
	}
	return pages
}

// HasPrev returns true if there is a previous page.
func (p *Pagination) HasPrev() bool {
	return p.Page > 1
}

// HasNext returns true if there is a next page.
func (p *Pagination) HasNext() bool {
	return p.Page < p.TotalPages()
}

// Offset returns the offset for database queries.
func (p *Pagination) Offset() int {
	return (p.Page - 1) * p.PerPage
}

// Limit returns the limit for database queries.
func (p *Pagination) Limit() int {
	return p.PerPage
}

// Links generates pagination links.
func (p *Pagination) Links() *Links {
	totalPages := p.TotalPages()

	links := &Links{
		Self:  p.buildURL(p.Page),
		First: p.buildURL(1),
		Last:  p.buildURL(totalPages),
	}

	if p.HasPrev() {
		links.Prev = p.buildURL(p.Page - 1)
	}
	if p.HasNext() {
		links.Next = p.buildURL(p.Page + 1)
	}

	return links
}

// buildURL builds a URL with pagination query parameters.
func (p *Pagination) buildURL(page int) string {
	if p.BaseURL == "" {
		return ""
	}

	u, err := url.Parse(p.BaseURL)
	if err != nil {
		return p.BaseURL
	}

	q := u.Query()
	q.Set("page[number]", strconv.Itoa(page))
	q.Set("page[size]", strconv.Itoa(p.PerPage))
	u.RawQuery = q.Encode()

	return u.String()
}

// Meta returns pagination metadata.
func (p *Pagination) Meta() Meta {
	return Meta{
		"total":    p.Total,
		"page":     p.Page,
		"per_page": p.PerPage,
		"pages":    p.TotalPages(),
	}
}

// ParsePaginationParams extracts pagination parameters from URL query.
// Returns page (1-based), perPage, and whether they were explicitly set.
func ParsePaginationParams(query url.Values, defaultPerPage int) (page, perPage int) {
	page = 1
	perPage = defaultPerPage

	// Try JSON:API style first: page[number] and page[size]
	if v := query.Get("page[number]"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			page = n
		}
	}
	if v := query.Get("page[size]"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			perPage = n
		}
	}

	// Fall back to simple style: page and per_page (or limit)
	if page == 1 {
		if v := query.Get("page"); v != "" {
			if n, err := strconv.Atoi(v); err == nil && n > 0 {
				page = n
			}
		}
	}
	if perPage == defaultPerPage {
		if v := query.Get("per_page"); v != "" {
			if n, err := strconv.Atoi(v); err == nil && n > 0 {
				perPage = n
			}
		} else if v := query.Get("limit"); v != "" {
			if n, err := strconv.Atoi(v); err == nil && n > 0 {
				perPage = n
			}
		}
	}

	// Cap perPage at reasonable limit
	if perPage > 100 {
		perPage = 100
	}

	return page, perPage
}
