package main

import (
	"github.com/go-playground/validator/v10"
)

type CustomValidator struct {
	validator *validator.Validate
}

func (cv *CustomValidator) Validate(i any) error {
	if err := cv.validator.Struct(i); err != nil {
		return err
	}
	return nil
}

func NewCustomValidator(v *validator.Validate) *CustomValidator {
	return &CustomValidator{validator: v}
}
