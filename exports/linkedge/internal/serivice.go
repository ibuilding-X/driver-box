package internal

import "github.com/ibuilding-x/driver-box/exports/linkedge/model"

type Service interface {
	Create(model.Config) error
	Update(model.Config) error
	Delete(id string) error
}
