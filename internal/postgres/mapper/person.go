package mapper

import (
	"person/internal/app"
	"person/internal/postgres/model"
)

func ToDB(person *app.Person) model.Person {
	return model.Person{
		ID:        person.ID(),
		Email:     person.Email(),
		Phone:     person.Phone(),
		FirstName: person.FirstName(),
		LastName:  person.LastName(),
		Version:   person.Version(),
	}
}

func ToDomain(m model.Person) (*app.Person, error) {
	return app.ReconstructPerson(app.PersonSnapshot{
		ID:        m.ID,
		Email:     m.Email,
		Phone:     m.Phone,
		FirstName: m.FirstName,
		LastName:  m.LastName,
		Version:   m.Version,
	})
}
