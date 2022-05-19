// Package cmd is responsible for the cli commands used to manage the
// prolog platform. We are using the cobra cli to model our command
// line interactions
package cmd

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/joho/godotenv"
	"github.com/mitchellh/go-homedir"
	rsbConf "github.com/rsb/conf"
	"github.com/rsb/failure"
	"github.com/rsb/prolog/app"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

var (
	cfgFile string
	build   = "develop"
)

// rootCmd is the base cli command
var rootCmd = &cobra.Command{
	Use:   "prolog",
	Short: "cli tool to help develop, deploy and test the prolog platform",
	Long: `prolog platform cli tool is used for the following:
- Aid in development of the distributed services
- API server management
- Admin tasks
- Testing 
- Deployment helpers
`,
	Version: build,
}

func init() {
	if err := godotenv.Load(); err != nil {
		log.Println("[init] no .env file used.")
	}
	cobra.OnInitialize(initConfig)
	template := `{{printf "%s: %s - version %s\n" .Name .Short .Version}}`
	rootCmd.SetVersionTemplate(template)

	replacer := strings.NewReplacer("-", "_")
	viper.SetEnvKeyReplacer(replacer)
	viper.SetEnvPrefix(strings.ToUpper(app.ServiceName))

	// rootCmd.AddCommand(apiCmd)
	// rootCmd.AddCommand(logFmtCmd)
	// rootCmd.AddCommand(auth0Cmd)
	// // api sub commands
	// apiCmd.AddCommand(serveCmd)
	//
	// // auth0 sub commands
	// auth0Cmd.AddCommand(genKeyCmd)
	// auth0Cmd.AddCommand(validateKeyCmd)

}

func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		root := app.RootDir()
		configName := "lola-platform"
		viper.AddConfigPath(root)
		viper.SetConfigType("toml")
		viper.SetConfigName(configName)
	}

	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		log.Println("cli-init,initConfig, config-file:", viper.ConfigFileUsed())
	}
}

func Execute(b string) {
	build = b
	cobra.CheckErr(rootCmd.Execute())
}

func processConfigCLI(v *viper.Viper, c interface{}) error {
	if err := rsbConf.ProcessCLI(v, c); err != nil {
		return failure.Wrap(err, "rsbConf.ProcessCLI failed")
	}

	return nil
}

func bindCLI(c *cobra.Command, v *viper.Viper, config interface{}) {
	if err := rsbConf.BindCLI(c, v, config); err != nil {
		err = failure.Wrap(err, "rsbConf.BindCLI failed")
		_, _ = fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
}

func failureExit(log *zap.SugaredLogger, err error, cat, msg string) {
	err = failure.Wrap(err, msg)
	if log == nil {
		_, _ = fmt.Fprintf(os.Stderr, "ERROR: (%s)", err.Error())
		os.Exit(1)
	}

	log.Errorw(cat, "ERROR", err)
	os.Exit(1)
}

// Filepath is used as a custom decoder which will take a configuration string
// and resolve a ~ to the absolute path of the home directory. If ~ is not
// present it treated as a normal path to a directory
type Filepath struct {
	Path string
}

func (d *Filepath) String() string {
	return d.Path
}

func (d *Filepath) IsEmpty() bool {
	return d.Path == ""
}

func (d *Filepath) Decode(v string) error {
	if v == "" {
		return failure.InvalidParam("directory can not be empty")
	}

	path, err := homedir.Expand(v)
	if err != nil {
		return failure.ToSystem(err, "homedir.Expand failed")
	}

	d.Path = path
	return nil
}
