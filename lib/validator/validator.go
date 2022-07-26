package validator

import (
	"errors"
	"strings"

	"github.com/go-playground/validator/v10"
)

const MAX_OPTION_LENGTH = 100
const MAX_TITLE_LENGTH = 300

type Validator struct {
	validator *validator.Validate
}

func NewValidator() *Validator {
	v := validator.New()
	_ = v.RegisterValidation("checkOption", func(fl validator.FieldLevel) bool {
		if fl.Field().Len() == 0 {
			return false
		}

		for i := 0; i < fl.Field().Len(); i++ {
			v := fl.Field().Index(i).String()
			if len(v) > MAX_OPTION_LENGTH {
				return false
			}
		}

		return true
	})

	return &Validator{validator: v}
}

func (val *Validator) Validate(i interface{}) error {
	err := val.validator.Struct(i)
	if err == nil {
		return nil
	}

	err = errors.New(strings.ReplaceAll(err.Error(), "\n", ", "))

	return err
}
