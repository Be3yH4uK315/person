package app

import (
	"fmt"
	"net/mail"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/google/uuid"
)

const initialPersonVersion int64 = 1

type PersonDetails struct {
	Email     string
	Phone     string
	FirstName string
	LastName  string
}

type PersonSnapshot struct {
	ID        uuid.UUID
	Email     string
	Phone     string
	FirstName string
	LastName  string
	Version   int64
}

type Person struct {
	id        uuid.UUID
	email     string
	phone     string
	firstName string
	lastName  string
	version   int64
}

func NewPerson(details PersonDetails) (*Person, error) {
	details = normalizePersonDetails(details)
	person := &Person{
		id:        uuid.New(),
		email:     details.Email,
		phone:     details.Phone,
		firstName: details.FirstName,
		lastName:  details.LastName,
		version:   initialPersonVersion,
	}
	if err := person.validate(); err != nil {
		return nil, err
	}
	return person, nil
}

func ReconstructPerson(snapshot PersonSnapshot) (*Person, error) {
	details := normalizePersonDetails(PersonDetails{
		Email: snapshot.Email, Phone: snapshot.Phone,
		FirstName: snapshot.FirstName, LastName: snapshot.LastName,
	})
	person := &Person{
		id:        snapshot.ID,
		email:     details.Email,
		phone:     details.Phone,
		firstName: details.FirstName,
		lastName:  details.LastName,
		version:   snapshot.Version,
	}
	if err := person.validateIdentity(); err != nil {
		return nil, fmt.Errorf("%w: persisted person has invalid identity", ErrCorruptedPerson)
	}
	if err := person.validateMetadata(); err != nil {
		return nil, fmt.Errorf("%w: persisted person has invalid metadata", ErrCorruptedPerson)
	}
	return person, nil
}

func (p *Person) Replace(details PersonDetails) error {
	if p == nil {
		return fmt.Errorf("%w: person is nil", ErrInvalidPersonState)
	}

	details = normalizePersonDetails(details)
	next := &Person{
		id:        p.id,
		email:     details.Email,
		phone:     details.Phone,
		firstName: details.FirstName,
		lastName:  details.LastName,
		version:   p.version,
	}
	if err := next.validate(); err != nil {
		return err
	}

	p.email = next.email
	p.phone = next.phone
	p.firstName = next.firstName
	p.lastName = next.lastName
	return nil
}

func (p *Person) validate() error {
	if err := p.validateIdentity(); err != nil {
		return err
	}
	details := PersonDetails{
		Email: p.email, Phone: p.phone,
		FirstName: p.firstName, LastName: p.lastName,
	}
	if violations := validatePersonFields(details); len(violations) > 0 {
		return NewValidationError(violations)
	}
	return p.validateMetadata()
}

func (p *Person) validateIdentity() error {
	if p == nil || p.id == uuid.Nil {
		return fmt.Errorf("%w: invalid id", ErrInvalidPersonState)
	}
	return nil
}

func (p *Person) validateMetadata() error {
	if p.version < 1 {
		return fmt.Errorf("%w: version must be positive", ErrInvalidPersonState)
	}
	return nil
}

func normalizePersonDetails(details PersonDetails) PersonDetails {
	return PersonDetails{
		Email:     normalizeEmail(details.Email),
		Phone:     strings.TrimSpace(details.Phone),
		FirstName: strings.TrimSpace(details.FirstName),
		LastName:  strings.TrimSpace(details.LastName),
	}
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

const (
	maxEmailLength = 254
	maxNameLength  = 100
)

func validatePersonFields(details PersonDetails) map[string]string {
	violations := make(map[string]string)

	if details.Email == "" {
		violations["email"] = "is required"
	} else if len(details.Email) > maxEmailLength {
		violations["email"] = "must not exceed 254 characters"
	} else if containsControl(details.Email) {
		violations["email"] = "must not contain control characters"
	} else if address, err := mail.ParseAddress(details.Email); err != nil || address.Address != details.Email {
		violations["email"] = "must be a valid email"
	}

	if details.Phone == "" {
		violations["phone"] = "is required"
	} else if !validPhone(details.Phone) {
		violations["phone"] = "must match +?[0-9]{7,15}"
	}

	if details.FirstName == "" {
		violations["firstName"] = "is required"
	} else if utf8.RuneCountInString(details.FirstName) > maxNameLength {
		violations["firstName"] = "must not exceed 100 characters"
	} else if containsControl(details.FirstName) {
		violations["firstName"] = "must not contain control characters"
	}

	if details.LastName == "" {
		violations["lastName"] = "is required"
	} else if utf8.RuneCountInString(details.LastName) > maxNameLength {
		violations["lastName"] = "must not exceed 100 characters"
	} else if containsControl(details.LastName) {
		violations["lastName"] = "must not contain control characters"
	}

	return violations
}

func validPhone(phone string) bool {
	digits := 0
	for index, r := range phone {
		if r == '+' {
			if index != 0 {
				return false
			}
			continue
		}
		if r < '0' || r > '9' {
			return false
		}
		digits++
	}
	return digits >= 7 && digits <= 15
}

func containsControl(value string) bool {
	for _, r := range value {
		if unicode.IsControl(r) {
			return true
		}
	}
	return false
}

func (p *Person) ID() uuid.UUID     { return p.id }
func (p *Person) Email() string     { return p.email }
func (p *Person) Phone() string     { return p.phone }
func (p *Person) FirstName() string { return p.firstName }
func (p *Person) LastName() string  { return p.lastName }
func (p *Person) Version() int64    { return p.version }
