package cmd

import (
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"go.hollow.sh/toolbox/version"
	"go.uber.org/zap"

	homedir "github.com/mitchellh/go-homedir"
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

	rootCmd.PersistentFlags().Bool("debug", false, "enable debug logging")
	viperBindFlag("logging.debug", rootCmd.PersistentFlags().Lookup("debug"))

	rootCmd.PersistentFlags().Bool("pretty", false, "enable pretty (human readable) logging output")
	viperBindFlag("logging.pretty", rootCmd.PersistentFlags().Lookup("pretty"))

	rootCmd.PersistentFlags().String("db-uri", "postgresql://root@localhost:26257/metadataservice?sslmode=disable", "URI for database connection")
	viperBindFlag("db.uri", rootCmd.PersistentFlags().Lookup("db-uri"))
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

	setupLogging()

	if err == nil {
		logger.Infow("using config file", "file", viper.ConfigFileUsed())
	}
}

func setupLogging() {
	cfg := zap.NewProductionConfig()
	if viper.GetBool("logging.pretty") {
		cfg = zap.NewDevelopmentConfig()
	}

	if viper.GetBool("logging.debug") {
		cfg.Level = zap.NewAtomicLevelAt(zap.DebugLevel)
	} else {
		cfg.Level = zap.NewAtomicLevelAt(zap.InfoLevel)
	}

	l, err := cfg.Build()
	if err != nil {
		panic(err)
	}

	logger = l.Sugar().With("app", "metadataservice", "version", version.Version())
	defer logger.Sync() //nolint:errcheck
}

// viperBindFlag provides a wrapper around the viper bindings that handles error checks
func viperBindFlag(name string, flag *pflag.Flag) {
	err := viper.GetViper().BindPFlag(name, flag)
	if err != nil {
		panic(err)
	}
}
