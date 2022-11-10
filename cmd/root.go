package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"go.infratographer.com/x/goosex"
	"go.infratographer.com/x/loggingx"
	"go.infratographer.com/x/versionx"
	"go.uber.org/zap"

	homedir "github.com/mitchellh/go-homedir"

	dbm "go.hollow.sh/metadataservice/db"
	"go.hollow.sh/metadataservice/internal/config"
)

var (
	cfgFile string
	logger  *zap.SugaredLogger
)

var rootCmd = &cobra.Command{
	Use:   "metadataservice",
	Short: "Instance Metadata Service for Hollow ecosystem",
}

// Execute adds all child commands to the root command and sets flags as
// appropriate. This is called by main.main(). It only needs to happen once to
// the rootCmd.
func Execute() {
	cobra.CheckErr(rootCmd.Execute())
}

func init() {
	cobra.OnInitialize(initConfig)
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.metadataservice.yml")

	// Logging flags
	loggingx.MustViperFlags(rootCmd.PersistentFlags())

	// Register version command
	versionx.RegisterCobraCommand(rootCmd, func() { versionx.PrintVersion(logger) })

	// Setup migrate command
	goosex.RegisterCobraCommand(rootCmd, func() {
		goosex.SetBaseFS(dbm.Migrations)
		goosex.SetDBURI(config.AppConfig.CRDB.URI)
		goosex.SetLogger(logger)
	})
}

// initConfig reads in config file and ENV variables if set
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := homedir.Dir()
		cobra.CheckErr(err)

		// search config in home directory with name ".metadataservice" (without extension)
		viper.AddConfigPath(home)
		viper.SetConfigName(".metadataservice")
	}

	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.SetEnvPrefix("metadataservice")
	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, reat it in.
	err := viper.ReadInConfig()

	setupAppConfig()

	// setupLogging()
	logger = loggingx.InitLogger("metadataservice", config.AppConfig.Logging)

	if err == nil {
		logger.Infow("using config file", "file", viper.ConfigFileUsed())
	}
}

// setupAppConfig loads our config.AppConfig struct with the values bound by
// viper. Then, anywhere we need these values, we can just return to AppConfig
// instead of performing viper.GetString(...), viper.GetBool(...), etc.
func setupAppConfig() {
	err := viper.Unmarshal(&config.AppConfig)
	if err != nil {
		fmt.Printf("unable to decode app config: %s", err)
		os.Exit(1)
	}
}

// viperBindFlag provides a wrapper around the viper bindings that handles error checks
func viperBindFlag(name string, flag *pflag.Flag) {
	err := viper.BindPFlag(name, flag)
	if err != nil {
		panic(err)
	}
}
