package jsonapi

import (
	"net/url"
	"testing"
)

func TestNewPagination(t *testing.T) {
	t.Run("creates pagination with valid values", func(t *testing.T) {
		p := NewPagination(100, 2, 10, "/users")

		if p.Total != 100 {
			t.Errorf("Total = %d, want 100", p.Total)
		}
		if p.Page != 2 {
			t.Errorf("Page = %d, want 2", p.Page)
		}
		if p.PerPage != 10 {
			t.Errorf("PerPage = %d, want 10", p.PerPage)
		}
		if p.BaseURL != "/users" {
			t.Errorf("BaseURL = %v, want /users", p.BaseURL)
		}
	})

	t.Run("normalizes page less than 1", func(t *testing.T) {
		p := NewPagination(100, 0, 10, "/users")
		if p.Page != 1 {
			t.Errorf("Page = %d, want 1 for page=0", p.Page)
		}

		p = NewPagination(100, -5, 10, "/users")
		if p.Page != 1 {
			t.Errorf("Page = %d, want 1 for page=-5", p.Page)
		}
	})

	t.Run("normalizes perPage less than 1", func(t *testing.T) {
		p := NewPagination(100, 1, 0, "/users")
		if p.PerPage != 20 {
			t.Errorf("PerPage = %d, want 20 for perPage=0", p.PerPage)
		}

		p = NewPagination(100, 1, -10, "/users")
		if p.PerPage != 20 {
			t.Errorf("PerPage = %d, want 20 for perPage=-10", p.PerPage)
		}
	})
}

func TestTotalPages(t *testing.T) {
	tests := []struct {
		name      string
		total     int64
		perPage   int
		wantPages int
	}{
		{"zero total returns 1", 0, 10, 1},
		{"exact division", 100, 10, 10},
		{"remainder rounds up", 101, 10, 11},
		{"one item", 1, 10, 1},
		{"less than page size", 5, 10, 1},
		{"large total", 1000, 25, 40},
		{"single item per page", 5, 1, 5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewPagination(tt.total, 1, tt.perPage, "/test")
			got := p.TotalPages()
			if got != tt.wantPages {
				t.Errorf("TotalPages() = %d, want %d", got, tt.wantPages)
			}
		})
	}
}

func TestHasPrev(t *testing.T) {
	tests := []struct {
		name     string
		page     int
		wantPrev bool
	}{
		{"first page has no prev", 1, false},
		{"second page has prev", 2, true},
		{"later page has prev", 10, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewPagination(100, tt.page, 10, "/test")
			got := p.HasPrev()
			if got != tt.wantPrev {
				t.Errorf("HasPrev() = %v, want %v", got, tt.wantPrev)
			}
		})
	}
}

func TestHasNext(t *testing.T) {
	tests := []struct {
		name     string
		total    int64
		page     int
		perPage  int
		wantNext bool
	}{
		{"has next when more pages", 100, 1, 10, true},
		{"no next on last page", 100, 10, 10, false},
		{"no next when single page", 5, 1, 10, false},
		{"has next on middle page", 100, 5, 10, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewPagination(tt.total, tt.page, tt.perPage, "/test")
			got := p.HasNext()
			if got != tt.wantNext {
				t.Errorf("HasNext() = %v, want %v", got, tt.wantNext)
			}
		})
	}
}

func TestOffset(t *testing.T) {
	tests := []struct {
		name       string
		page       int
		perPage    int
		wantOffset int
	}{
		{"first page offset is 0", 1, 10, 0},
		{"second page offset", 2, 10, 10},
		{"third page offset", 3, 25, 50},
		{"large page offset", 10, 20, 180},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewPagination(1000, tt.page, tt.perPage, "/test")
			got := p.Offset()
			if got != tt.wantOffset {
				t.Errorf("Offset() = %d, want %d", got, tt.wantOffset)
			}
		})
	}
}

func TestLimit(t *testing.T) {
	tests := []struct {
		name      string
		perPage   int
		wantLimit int
	}{
		{"returns perPage", 10, 10},
		{"returns perPage for 25", 25, 25},
		{"returns perPage for 1", 1, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewPagination(100, 1, tt.perPage, "/test")
			got := p.Limit()
			if got != tt.wantLimit {
				t.Errorf("Limit() = %d, want %d", got, tt.wantLimit)
			}
		})
	}
}

func TestPaginationLinks(t *testing.T) {
	t.Run("generates all links on middle page", func(t *testing.T) {
		p := NewPagination(100, 5, 10, "/users")
		links := p.Links()

		if links.Self == "" {
			t.Error("Self link should be set")
		}
		if links.First == "" {
			t.Error("First link should be set")
		}
		if links.Last == "" {
			t.Error("Last link should be set")
		}
		if links.Prev == "" {
			t.Error("Prev link should be set on middle page")
		}
		if links.Next == "" {
			t.Error("Next link should be set on middle page")
		}
	})

	t.Run("no prev on first page", func(t *testing.T) {
		p := NewPagination(100, 1, 10, "/users")
		links := p.Links()

		if links.Prev != "" {
			t.Errorf("Prev should be empty on first page, got %v", links.Prev)
		}
		if links.Next == "" {
			t.Error("Next should be set on first page with more pages")
		}
	})

	t.Run("no next on last page", func(t *testing.T) {
		p := NewPagination(100, 10, 10, "/users")
		links := p.Links()

		if links.Next != "" {
			t.Errorf("Next should be empty on last page, got %v", links.Next)
		}
		if links.Prev == "" {
			t.Error("Prev should be set on last page")
		}
	})

	t.Run("single page has no prev or next", func(t *testing.T) {
		p := NewPagination(5, 1, 10, "/users")
		links := p.Links()

		if links.Prev != "" {
			t.Errorf("Prev should be empty on single page, got %v", links.Prev)
		}
		if links.Next != "" {
			t.Errorf("Next should be empty on single page, got %v", links.Next)
		}
	})
}

func TestBuildURL(t *testing.T) {
	t.Run("builds URL with page params", func(t *testing.T) {
		p := NewPagination(100, 1, 10, "/users")
		url := p.buildURL(2)

		if url == "" {
			t.Error("URL should not be empty")
		}
		// URL should contain page[number] and page[size]
		if url == "/users" {
			t.Error("URL should have pagination params")
		}
	})

	t.Run("empty base URL returns empty", func(t *testing.T) {
		p := NewPagination(100, 1, 10, "")
		url := p.buildURL(1)

		if url != "" {
			t.Errorf("Empty base URL should return empty, got %v", url)
		}
	})

	t.Run("preserves existing query params", func(t *testing.T) {
		p := NewPagination(100, 1, 10, "/users?filter=active")
		url := p.buildURL(2)

		if url == "" {
			t.Error("URL should not be empty")
		}
	})

	t.Run("handles invalid URL gracefully", func(t *testing.T) {
		p := NewPagination(100, 1, 10, "://invalid")
		url := p.buildURL(1)

		// Should return the base URL as fallback
		if url != "://invalid" {
			t.Errorf("Invalid URL should return base, got %v", url)
		}
	})
}

func TestPaginationMeta(t *testing.T) {
	p := NewPagination(100, 2, 10, "/users")
	meta := p.Meta()

	if meta["total"] != int64(100) {
		t.Errorf("meta[total] = %v, want 100", meta["total"])
	}
	if meta["page"] != 2 {
		t.Errorf("meta[page] = %v, want 2", meta["page"])
	}
	if meta["per_page"] != 10 {
		t.Errorf("meta[per_page] = %v, want 10", meta["per_page"])
	}
	if meta["pages"] != 10 {
		t.Errorf("meta[pages] = %v, want 10", meta["pages"])
	}
}

func TestParsePaginationParams(t *testing.T) {
	t.Run("parses JSON:API style params", func(t *testing.T) {
		q := url.Values{}
		q.Set("page[number]", "3")
		q.Set("page[size]", "25")

		page, perPage := ParsePaginationParams(q, 20)

		if page != 3 {
			t.Errorf("page = %d, want 3", page)
		}
		if perPage != 25 {
			t.Errorf("perPage = %d, want 25", perPage)
		}
	})

	t.Run("parses simple style params", func(t *testing.T) {
		q := url.Values{}
		q.Set("page", "5")
		q.Set("per_page", "15")

		page, perPage := ParsePaginationParams(q, 20)

		if page != 5 {
			t.Errorf("page = %d, want 5", page)
		}
		if perPage != 15 {
			t.Errorf("perPage = %d, want 15", perPage)
		}
	})

	t.Run("parses limit param as fallback", func(t *testing.T) {
		q := url.Values{}
		q.Set("limit", "30")

		page, perPage := ParsePaginationParams(q, 20)

		if page != 1 {
			t.Errorf("page = %d, want 1", page)
		}
		if perPage != 30 {
			t.Errorf("perPage = %d, want 30", perPage)
		}
	})

	t.Run("uses defaults for empty query", func(t *testing.T) {
		q := url.Values{}

		page, perPage := ParsePaginationParams(q, 20)

		if page != 1 {
			t.Errorf("page = %d, want 1", page)
		}
		if perPage != 20 {
			t.Errorf("perPage = %d, want 20", perPage)
		}
	})

	t.Run("caps perPage at 100", func(t *testing.T) {
		q := url.Values{}
		q.Set("page[size]", "500")

		_, perPage := ParsePaginationParams(q, 20)

		if perPage != 100 {
			t.Errorf("perPage = %d, want 100 (capped)", perPage)
		}
	})

	t.Run("ignores invalid page number", func(t *testing.T) {
		q := url.Values{}
		q.Set("page[number]", "invalid")
		q.Set("page[size]", "10")

		page, _ := ParsePaginationParams(q, 20)

		if page != 1 {
			t.Errorf("page = %d, want 1 for invalid input", page)
		}
	})

	t.Run("ignores zero or negative values", func(t *testing.T) {
		q := url.Values{}
		q.Set("page[number]", "0")
		q.Set("page[size]", "-5")

		page, perPage := ParsePaginationParams(q, 20)

		if page != 1 {
			t.Errorf("page = %d, want 1 for zero", page)
		}
		if perPage != 20 {
			t.Errorf("perPage = %d, want 20 for negative", perPage)
		}
	})

	t.Run("JSON:API style takes precedence", func(t *testing.T) {
		q := url.Values{}
		q.Set("page[number]", "3")
		q.Set("page", "5")

		page, _ := ParsePaginationParams(q, 20)

		if page != 3 {
			t.Errorf("page = %d, want 3 (JSON:API should take precedence)", page)
		}
	})

	t.Run("per_page takes precedence over limit", func(t *testing.T) {
		q := url.Values{}
		q.Set("per_page", "15")
		q.Set("limit", "30")

		_, perPage := ParsePaginationParams(q, 20)

		if perPage != 15 {
			t.Errorf("perPage = %d, want 15 (per_page should take precedence)", perPage)
		}
	})
}
