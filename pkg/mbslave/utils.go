package mbslave

import "github.com/go-playground/validator/v10"

var validate = validator.New()

func complementModel(model *Model) {
	model.property = make(map[string]*Property)

	for _, property := range model.Properties {
		var length uint16 = 1

		switch property.ValueType {
		case "float32":
			length = 2
		}

		model.property[property.Name] = &Property{
			Name:      property.Name,
			Mode:      property.Mode,
			ValueType: property.ValueType,
			startAddr: model.registerNumber,
			length:    length,
		}

		model.registerNumber = model.registerNumber + length
	}
}

// ValidatorModel 校验模型
func ValidatorModel(model Model) error {
	return validate.Struct(model)
}
