package types

// PageOrderReq is a struct that represents the request parameters for pagination and ordering.
type PageOrderReq struct {
	Word     string `json:"word" form:"word,omitempty"`
	Page     uint   `json:"page" form:"page,omitempty" binding:"numeric,min=0"`
	PerPage  uint   `json:"perPage" form:"perPage,omitempty" binding:"numeric,min=0"`
	OrderBy  string `json:"orderBy" form:"orderBy,omitempty"`
	OrderDir string `json:"orderDir" form:"orderDir,omitempty"`
}

// PageOrderDefault sets the default values of the PageOrderReq struct.
//
// If any of the fields are invalid, this function will set the default values.
// The default values are as follows:
// - Page: 1
// - PerPage: 10
// - OrderBy: "create_at"
// - OrderDir: "DESC"
func (r *PageOrderReq) PageOrderDefault() {
	if r.Page <= 1 {
		r.Page = 1
	}
	if r.PerPage < 1 {
		r.PerPage = 10
	}
	if r.OrderBy == "" {
		r.OrderBy = "create_at"
	}
	if r.OrderDir == "" {
		r.OrderDir = "DESC"
	}
}
