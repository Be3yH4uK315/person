package response

type PersonResponse struct {
	ID        string `json:"id"`
	Email     string `json:"email"`
	Phone     string `json:"phone"`
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
	CreatedAt string `json:"createdAt"`
	UpdatedAt string `json:"updatedAt"`
	Version   int64  `json:"version"`
}

type Pagination struct {
	Limit  int `json:"limit"`
	Offset int `json:"offset"`
	Total  int `json:"total"`
	Page   int `json:"page"`
	Pages  int `json:"pages"`
}

type PersonListResponse struct {
	Items      []PersonResponse `json:"items"`
	Pagination Pagination       `json:"pagination"`
}

type MutationResponse struct {
	ID  string `json:"id"`
	URI string `json:"uri"`
}

type ErrorResponse struct {
	Error     ErrorBody `json:"error"`
	RequestID string    `json:"requestId"`
}

type ErrorBody struct {
	Code    string            `json:"code"`
	Message string            `json:"message"`
	Details map[string]string `json:"details,omitempty"`
}
