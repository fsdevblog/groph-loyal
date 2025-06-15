package api

import (
	"fmt"
	"strconv"

	"github.com/gin-gonic/gin/binding"

	"github.com/go-playground/validator/v10"
)

// validateMaxBytes в отличии от тэга max который проверяет длину рун, - проверят длину байт в поле.
func validateMaxBytes(fl validator.FieldLevel) bool {
	param := fl.Param() // получаем значение из тега
	maxBytes, err := strconv.Atoi(param)
	if err != nil {
		return false
	}

	// нужно убедится что значение поля - строка.
	str, ok := fl.Field().Interface().(string)
	if !ok {
		return false
	}

	return len([]byte(str)) <= maxBytes
}

func registerValidators() error {
	v, _ := binding.Validator.Engine().(*validator.Validate)
	if err := v.RegisterValidation("max_bytes", validateMaxBytes); err != nil {
		return fmt.Errorf("validator registration: %s", err.Error())
	}
	return nil
}
