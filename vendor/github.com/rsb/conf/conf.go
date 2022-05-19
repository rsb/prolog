/*
Package conf is a package that specializes in parsing out environment variables
using structs with annotated tags to control how it is done.
*/
package conf

import (
	"fmt"
	"os"
	"reflect"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ssm"
	"github.com/aws/aws-sdk-go/service/ssm/ssmiface"
	"github.com/rsb/failure"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	AWSLambdaFunctionNameVar = "AWS_LAMBDA_FUNCTION_NAME"
	AWSProfile               = "AWS_PROFILE"
	AWSRegion                = "AWS_REGION"
	AppName                  = "APP_NAME"
	GlobalParamStoreKey      = "global"
)

var excludedVars = []string{
	AppName,
	AWSProfile,
	AWSRegion,
	AWSLambdaFunctionNameVar,
}

type Config struct {
	Data interface{}
}

func NewConfig(d interface{}) *Config {
	return &Config{Data: d}
}

func (c *Config) ProcessCLI(v *viper.Viper, prefix ...string) error {
	if err := ProcessCLI(v, c.Data, prefix...); err != nil {
		return failure.Wrap(err, "ProcessCLI failed")
	}

	return nil
}

func (c *Config) ProcessEnv(prefix ...string) error {
	if err := ProcessEnv(c.Data, prefix...); err != nil {
		return failure.Wrap(err, "ProcessEnv failed")
	}

	return nil
}

func (c *Config) ProcessParamStore(pstore ssmiface.SSMAPI, appTitle string, isEncrypted bool, prefix ...string) (map[string]string, error) {
	result, err := ProcessParamStore(pstore, appTitle, isEncrypted, c.Data, prefix...)
	if err != nil {
		return nil, failure.Wrap(err, "ProcessParamStore failed")
	}

	return result, nil
}

func (c *Config) CollectParamStoreFromEnv(appTitle string, prefix ...string) (map[string]string, error) {
	result, err := CollectParamStoreFromEnv(appTitle, c.Data, prefix...)
	if err != nil {
		return nil, failure.Wrap(err, "CollectParamStoreFromEnv failed")
	}

	return result, nil
}

func (c *Config) EnvNames(prefix ...string) ([]string, error) {
	name, err := EnvNames(c.Data, prefix...)
	if err != nil {
		return nil, failure.Wrap(err, "EnvNames failed")
	}

	return name, nil
}

func (c *Config) EnvToMap(prefix ...string) (map[string]string, error) {
	result, err := EnvToMap(c.Data, prefix...)
	if err != nil {
		return nil, failure.Wrap(err, "EnvToMap failed")
	}

	return result, nil
}

func (c *Config) EnvReport(prefix ...string) (map[string]string, error) {
	result, err := EnvReport(c.Data, prefix...)
	if err != nil {
		return nil, failure.Wrap(err, "Report failed")
	}

	return result, nil
}

func BindCLI(cmd *cobra.Command, v *viper.Viper, spec interface{}, prefix ...string) error {
	fields, err := Fields(spec, prefix...)
	if err != nil {
		return failure.Wrap(err, "Fields failed")
	}

	for _, field := range fields {
		if !field.IsCLI() {
			continue
		}

		flag := field.CLIFlag()
		short := field.CLIShortFlag()
		usage := field.CLIUsage()
		defaultValue := field.DefaultValue()

		flagSet := cmd.Flags()
		if field.IsPersistentFlag() {
			flagSet = cmd.PersistentFlags()
		}

		switch field.ReflectValue.Type().Kind() {
		case reflect.Bool:
			if defaultValue == "" {
				defaultValue = "false"
			}
			dv, err := strconv.ParseBool(defaultValue)
			if err != nil {
				return failure.ToSystem(err, "strconv.ParseBool failed")
			}
			if short != "" {
				flagSet.BoolP(flag, short, dv, usage)
			} else {
				flagSet.Bool(flag, dv, usage)
			}
		default:
			if short != "" {
				flagSet.StringP(flag, short, defaultValue, usage)
			} else {
				flagSet.String(flag, defaultValue, usage)
			}
		}

		lookupFlag := flagSet.Lookup(flag)
		flagID := field.BindName()
		if err := v.BindPFlag(flagID, lookupFlag); err != nil {
			return failure.ToSystem(err, "v.BindPFlag failed for (%s)", flag)
		}
	}

	return nil
}

func ProcessCLI(v *viper.Viper, spec interface{}, prefix ...string) error {
	fields, err := Fields(spec, prefix...)
	if err != nil {
		return failure.Wrap(err, "Fields failed")
	}

	for _, field := range fields {
		env := field.EnvVariable()
		flag := field.CLIFlag()
		flagID := field.BindName()
		value := v.GetString(flagID)

		if value == "" {
			if v.InConfig(flagID) {
				data := v.Get(flagID)
				switch d := data.(type) {
				case map[string]interface{}:
					for k, v := range d {
						value += fmt.Sprintf("%s:%s,", k, v)
					}
					value = strings.TrimRight(value, ",")
				}
			} else {
				value = EnvVarOptional(env)
			}
		}

		// This will not happen if you use BindCLI because the default value is
		// always set. It is here just in case you are doing things manually
		if value == "" && field.IsDefault() {
			value = field.DefaultValue()
		}

		if value == "" && !field.IsDefault() {
			if field.IsRequired() {
				return failure.Config("required key (field:%s,env:%s,cmds:%s) missing value", field.Name, env, flag)
			}
			continue
		}

		if err = ProcessField(value, field.ReflectValue); err != nil {
			return failure.Wrap(err, "ProcessField failed (%s)", field.Name)
		}
	}

	return nil
}

func ProcessEnv(spec interface{}, prefix ...string) error {
	fields, err := Fields(spec, prefix...)
	if err != nil {
		return failure.Wrap(err, "Fields failed")
	}

	for _, field := range fields {
		env := field.EnvVariable()
		if env == "" {
			return failure.System("env: is required but empty for (%s)", field.Name)
		}

		value, ok := os.LookupEnv(env)
		if !ok && field.IsDefault() {
			value = field.DefaultValue()
		}

		if !ok && !field.IsDefault() {
			if field.IsRequired() {
				return failure.Config("required key (%s,%s) missing value", field.Name, env)
			}
			continue
		}

		if err = ProcessField(value, field.ReflectValue); err != nil {
			return failure.Wrap(err, "ProcessField failed (%s)", field.Name)
		}
	}

	return nil
}

func ProcessParamStore(pstore ssmiface.SSMAPI, appTitle string, isEncrypted bool, spec interface{}, prefix ...string) (map[string]string, error) {
	if pstore == nil {
		return nil, failure.System("pstore is nil")
	}

	if appTitle == "" {
		return nil, failure.System("appTitle is empty")
	}

	fields, err := Fields(spec, prefix...)
	if err != nil {
		return nil, failure.Wrap(err, "Fields failed")
	}

	result := map[string]string{}

OUTER:
	for _, field := range fields {
		env := field.EnvVariable()
		pkey := field.ParamStoreKey()

		if env == "-" || pkey == "-" {
			continue
		}

		if env == "" {
			return result, failure.System("env: is required but empty for (%s)", field.Name)
		}

		for _, ev := range excludedVars {
			if env == ev {
				continue OUTER
			}
		}

		key := PStoreKey(field, appTitle, env)
		in := ssm.GetParameterInput{
			Name:           aws.String(key),
			WithDecryption: aws.Bool(isEncrypted),
		}

		if field.IsDefault() {
			continue
		}

		out, err := pstore.GetParameter(&in)
		if err != nil {
			return result, failure.ToSystem(err, "pstore.API.GetParameter failed for (%s, %s)", field.Name, key)
		}

		if out == nil || out.Parameter == nil {
			return result, failure.System("pstore.API.GetParameter returned nil (%s, %s)", field.Name, key)
		}

		var value string
		if out.Parameter.Value != nil {
			value = *out.Parameter.Value
		}

		result[env] = value
	}

	return result, nil
}

func PStoreKey(field Field, appTitle, env string) string {
	var key string
	pkey := field.ParamStoreKey()
	switch {
	case pkey != "":
		key = pkey
	case field.IsGlobalParamStore():
		key = fmt.Sprintf("/%s/%s", GlobalParamStoreKey, env)
	default:
		key = fmt.Sprintf("/%s/%s", appTitle, env)
	}

	return key
}

func CollectParamStoreFromEnv(appTitle string, spec interface{}, prefix ...string) (map[string]string, error) {
	if appTitle == "" {
		return nil, failure.System("appTitle is empty")
	}

	fields, err := Fields(spec, prefix...)
	if err != nil {
		return nil, failure.Wrap(err, "Fields failed")
	}

	result := map[string]string{}

OUTER:
	for _, field := range fields {
		env := field.EnvVariable()
		pkey := field.ParamStoreKey()

		if env == "-" || pkey == "-" {
			continue
		}

		if env == "" {
			return result, failure.System("env: is required but empty for (%s)", field.Name)
		}

		for _, ev := range excludedVars {
			if env == ev {
				continue OUTER
			}
		}

		key := fmt.Sprintf("/%s/%s", appTitle, env)
		if pkey != "" {
			key = field.ParamStoreKey()
		}

		value, ok := os.LookupEnv(env)
		if !ok && field.IsDefault() {
			value = field.DefaultValue()
		}

		if !ok && !field.IsDefault() {
			if field.IsRequired() {
				return result, failure.Config("required key (%s,%s) missing value", field.Name, env)
			}
		}

		result[key] = value
	}

	return result, nil
}

func EnvReport(spec interface{}, prefix ...string) (map[string]string, error) {
	fields, err := Fields(spec, prefix...)
	if err != nil {
		return nil, failure.Wrap(err, "Fields failed")
	}

	result := map[string]string{}

OUTER:
	for _, field := range fields {
		env := field.EnvVariable()
		if env == "-" {
			continue
		}

		for _, ev := range excludedVars {
			if env == ev {
				continue OUTER
			}
		}

		if env == "" {
			return result, failure.System("env: is required but empty for (%s)", field.Name)
		}

		value, ok := os.LookupEnv(env)
		if !ok && field.IsDefault() {
			value = field.DefaultValue()
		}

		result[env] = value
	}

	return result, nil
}

func EnvToMap(spec interface{}, prefix ...string) (map[string]string, error) {
	fields, err := Fields(spec, prefix...)
	if err != nil {
		return nil, failure.Wrap(err, "Fields failed")
	}

	result := map[string]string{}

OUTER:
	for _, field := range fields {
		env := field.EnvVariable()
		if env == "-" {
			continue
		}

		for _, ev := range excludedVars {
			if env == ev {
				continue OUTER
			}
		}

		if env == "" {
			return result, failure.System("env: is required but empty for (%s)", field.Name)
		}

		value, ok := os.LookupEnv(env)
		if !ok && field.IsDefault() {
			value = field.DefaultValue()
		}

		if !ok && !field.IsDefault() {
			if field.IsRequired() {
				return result, failure.Config("required key (%s,%s) missing value", field.Name, env)
			}
		}

		result[env] = value
	}

	return result, nil
}

func EnvNames(spec interface{}, prefix ...string) ([]string, error) {
	var names []string

	fields, err := Fields(spec, prefix...)
	if err != nil {
		return nil, failure.Wrap(err, "Fields failed")
	}

OUTER:
	for _, field := range fields {
		env := field.EnvVariable()
		if env == "-" {
			continue
		}

		for _, ev := range excludedVars {
			if env == ev {
				continue OUTER
			}
		}
		names = append(names, env)
	}

	return names, nil
}

// EnvVar ensures the variable you are looking for is set. If you don't care
// about that use EnvVarOptional instead
func EnvVar(key string) (string, error) {
	value, ok := os.LookupEnv(key)
	if !ok {
		return value, failure.Config("env var (%s) is not set", key)
	}

	return value, nil
}

// EnvVarStrict ensures the variable is set and not empty
func EnvVarStrict(key string) (string, error) {
	value, err := EnvVar(key)
	if err != nil {
		return value, failure.Wrap(err, "EnvVar failed")
	}

	if value == "" {
		return value, failure.Config("env var (%s) is empty", key)
	}

	return value, nil
}

// EnvVarOptional is a wrapper around os.Getenv with the intent that by using
// this method you are declaring in code that you don't care about empty
// env vars. This is better than just using os.Getenv because that intent
// is not conveyed. So this simple wrapper has the purpose of reveal intent
// and not wrapping for the sake of wrapping
func EnvVarOptional(key string) string {
	return os.Getenv(key)
}
