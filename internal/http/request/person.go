package request

type CreatePersonRequest struct {
	Email     string `json:"email"`
	Phone     string `json:"phone"`
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
}

type UpdatePersonRequest struct {
	Email     string `json:"email"`
	Phone     string `json:"phone"`
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
	Version   int64  `json:"version"`
}

type DeletePersonRequest struct {
	Version int64 `json:"version"`
}
