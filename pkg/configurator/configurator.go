package configurator

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"reflect"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/ilyakaznacheev/cleanenv"
)

var (
	ErrConfigPathNotSet   = errors.New("config path not set")
	ErrConfigFileNotFound = errors.New("config file not found")
	ErrConfigValidation   = errors.New("config validation failed")
)

func Load(cfg any) error {
	path := fetchConfigPath()
	if path == "" {
		return fmt.Errorf("%w (use --config flag or CONFIG_PATH env)", ErrConfigPathNotSet)
	}
	return LoadPath(path, cfg)
}

func LoadPath(configPath string, cfg any) error {
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return fmt.Errorf("%w, path: %s", ErrConfigFileNotFound, configPath)
	}

	if err := cleanenv.ReadConfig(configPath, cfg); err != nil {
		return fmt.Errorf("read config: %w", err)
	}

	validate := validator.New()

	validate.RegisterTagNameFunc(func(fld reflect.StructField) string {
		for _, tag := range []string{"env", "yaml", "json"} {
			name := strings.Split(fld.Tag.Get(tag), ",")[0]
			if name != "" && name != "-" {
				return name
			}
		}
		return fld.Name
	})

	if err := validate.Struct(cfg); err != nil {
		return formatValidationError(err)
	}

	return nil
}

func formatValidationError(err error) error {
	if validationErrs, ok := errors.AsType[validator.ValidationErrors](err); ok {
		msgs := make([]string, 0, len(validationErrs))
		for _, ve := range validationErrs {
			msgs = append(msgs, fmt.Sprintf("field '%s' failed on tag '%s' (current value: %v)", ve.Field(), ve.Tag(), ve.Value()))
		}
		return fmt.Errorf("%w: %s", ErrConfigValidation, strings.Join(msgs, "; "))
	}
	return fmt.Errorf("%w: %w", ErrConfigValidation, err)
}

func fetchConfigPath() string {
	if path := os.Getenv("CONFIG_PATH"); path != "" {
		return path
	}

	fs := flag.NewFlagSet("config-loader", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var path string
	fs.StringVar(&path, "config", "", "path to config file")

	_ = fs.Parse(os.Args[1:])

	return path
}
