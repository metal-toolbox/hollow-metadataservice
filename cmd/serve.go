package cmd

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"go.hollow.sh/toolbox/ginjwt"

	"go.hollow.sh/metadataservice/internal/httpsrv"
)

// TODO: Uncomment once db stuff is set up
// const (
// 	defaultDBMaxOpenConns int = 25
// 	defaultDBMaxIdleConns int = 25
// )

// serveCmd represents the serve command
var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "starts the metadata server",
	Run: func(cmd *cobra.Command, args []string) {
		serve()
	},
}

func init() {
	rootCmd.AddCommand(serveCmd)
	serveCmd.Flags().String("listen", "0.0.0.0:8000", "address on which to listen")
	viperBindFlag("listen", serveCmd.Flags().Lookup("listen"))

	// TODO:
	// Tracing flags
	// DB flags

	// OIDC Flags
	serveCmd.Flags().Bool("oidc", true, "use oidc auth")
	viperBindFlag("oidc.enabled", serveCmd.Flags().Lookup("oidc"))

	serveCmd.Flags().String("oidc-aud", "", "expected audient on OIDC JWT")
	viperBindFlag("oidc.audience", serveCmd.Flags().Lookup("oidc-aud"))

	serveCmd.Flags().String("oidc-issuer", "", "expected issuer of OIDC JWT")
	viperBindFlag("oidc.issuer", serveCmd.Flags().Lookup("oidc-issuer"))

	serveCmd.Flags().String("oidc-jwksuri", "", "URI for JWKS listing for JWTs")
	viperBindFlag("oidc.jwksuri", serveCmd.Flags().Lookup("oidc-jwksuri"))

	serveCmd.Flags().String("oidc-roles-claim", "claim", "field containing the permissions of an OIDC JWT")
	viperBindFlag("oidc.claims.roles", serveCmd.Flags().Lookup("oidc-roles-claim"))

	serveCmd.Flags().String("oidc-username-claim", "", "additional fields to output in logs from the JWT token, ex (email)")
	viperBindFlag("oidc.claims.username", serveCmd.Flags().Lookup("oidc-username-claim"))
}

func serve() {
	// TODO: set up DB
	logger.Infow("starting metadata server", "address", viper.GetString("listen"))

	hs := &httpsrv.Server{
		Logger: logger.Desugar(),
		Listen: viper.GetString("listen"),
		Debug:  viper.GetBool("logging.debug"),
		AuthConfig: ginjwt.AuthConfig{
			Enabled:       viper.GetBool("oidc.enabled"),
			Audience:      viper.GetString("oidc.audience"),
			Issuer:        viper.GetString("oidc.issuer"),
			JWKSURI:       viper.GetString("oidc.jwksuri"),
			LogFields:     viper.GetStringSlice("oidc.log"), // TODO: We don't seem to be grabbing this from config?
			RolesClaim:    viper.GetString("oidc.claims.roles"),
			UsernameClaim: viper.GetString("oidc.claims.username"),
		},
	}

	if err := hs.Run(); err != nil {
		logger.Fatalw("failed starting metadata server", "error", err)
	}
}
