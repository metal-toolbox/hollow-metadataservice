package cmd

import (
	"context"
	"net/url"
	"text/template"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.org/x/oauth2/clientcredentials"

	"go.hollow.sh/toolbox/ginjwt"

	"go.hollow.sh/metadataservice/internal/httpsrv"
	"go.hollow.sh/metadataservice/internal/lookup"
)

const (
	defaultDBMaxOpenConns    int           = 25
	defaultDBMaxIdleConns    int           = 25
	defaultDBConnMaxLifetime time.Duration = 5 * 60 * time.Second
)

// serveCmd represents the serve command
var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "starts the metadata server",
	Run: func(cmd *cobra.Command, args []string) {
		serve(cmd.Context())
	},
}

func init() {
	rootCmd.AddCommand(serveCmd)
	serveCmd.Flags().String("listen", "0.0.0.0:8000", "address on which to listen")
	viperBindFlag("listen", serveCmd.Flags().Lookup("listen"))

	// Tracing flags
	serveCmd.Flags().Bool("tracing", false, "enable tracing support")
	viperBindFlag("tracing.enabled", serveCmd.Flags().Lookup("tracing"))

	serveCmd.Flags().String("tracing-provider", "jaeger", "tracing provider to use")
	viperBindFlag("tracing.provider", serveCmd.Flags().Lookup("tracing-provider"))

	serveCmd.Flags().String("tracing-endpoint", "", "endpoint where traces are sent")
	viperBindFlag("tracing.endpoint", serveCmd.Flags().Lookup("tracing-endpoint"))

	serveCmd.Flags().String("tracing-environment", "production", "environment value in traces")
	viperBindFlag("tracing.environment", serveCmd.Flags().Lookup("tracing-environment"))

	// DB flags
	serveCmd.Flags().Int("db-conns-max-open", defaultDBMaxOpenConns, "max number of open database connections")
	viperBindFlag("db.connections.max_open", serveCmd.Flags().Lookup("db-conns-max-open"))

	serveCmd.Flags().Int("db-conns-max-idle", defaultDBMaxIdleConns, "max number of idle database connections")
	viperBindFlag("db.connections.max_idle", serveCmd.Flags().Lookup("db-conns-max-idle"))

	serveCmd.Flags().Duration("db-conns-max-lifetime", defaultDBConnMaxLifetime, "max database connections lifetime in seconds")
	viperBindFlag("db.connections.max_lifetime", serveCmd.Flags().Lookup("db-conns-max-lifetime"))

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

	// Lookup Service Flags
	serveCmd.Flags().Bool("lookup-enabled", false, "Use the lookup client to attempt to fetch metadata or userdata from an upstream source when it is not cached locall for the instance")
	viperBindFlag("lookup.enabled", serveCmd.Flags().Lookup("lookup-enabled"))

	serveCmd.Flags().String("lookup-base-url", "", "A base url (like 'https://metadata-lookup-service.tld/api/v1/') to use when fetching metadata or userdata from an upstream source")
	viperBindFlag("lookup.baseurl", serveCmd.Flags().Lookup("lookup-base-url"))

	serveCmd.Flags().String("lookup-oidc-issuer", "", "OIDC JWT issuer to the lookup service")
	viperBindFlag("lookup.oidc.issuer", serveCmd.Flags().Lookup("lookup-oidc-issuer"))

	serveCmd.Flags().String("lookup-oidc-client-id", "", "OIDC Client ID to use by the lookup service client for auth token exchange")
	viperBindFlag("lookup.oidc.clientid", serveCmd.Flags().Lookup("lookup-oidc-client-id"))

	serveCmd.Flags().String("lookup-oidc-client-secret", "", "OIDC Client Secret to use by the lookup service client for auth token exchange")
	viperBindFlag("lookup.oidc.clientsecret", serveCmd.Flags().Lookup("lookup-oidc-client-secret"))

	serveCmd.Flags().String("lookup-oidc-aud", "", "OIDC JWT audience for lookup service")
	viperBindFlag("lookup.oidc.audience", serveCmd.Flags().Lookup("lookup-oidc-aud"))

	serveCmd.Flags().StringSlice("lookup-oidc-scopes", []string{"metadata:read:metadata", "metadata:read:userdata"}, "OIDC JWT scopes for lookup service")
	viperBindFlag("lookup.oidc.scopes", serveCmd.Flags().Lookup("lookup-oidc-scopes"))

	// Misc serve flags
	serveCmd.Flags().StringSlice("gin-trusted-proxies", []string{}, "Comma-separated list of IP addresses, like `\"192.168.1.1,10.0.0.1\"`. When running the Metadata Service behind something like a reverse proxy or load balancer, you may need to set this so that gin's `(*Context).ClientIP()` method returns a value provided by the proxy in a header like `X-Forwarded-For`.")
	viperBindFlag("gin.trustedproxies", serveCmd.Flags().Lookup("gin-trusted-proxies"))

	serveCmd.Flags().String("phone-home-url", "", "An optional golang template string used to build a URL which instances can use as part of a 'phone home' process. This template string will be evaluated against the instance metadata, and appended as a 'phone_home_url' field on the metadata document served to instances. If no template string is specified, the 'phone_home_url' field will not be added to the metadata document.")
	viperBindFlag("metadata.phone_home_url", serveCmd.Flags().Lookup("phone-home-url"))

	serveCmd.Flags().String("user-state-url", "", "An optional golang template string used to build a URL which instances can use for sending user state events. This template string will be evaluated against the instance metadata, and appended as a 'user_state_url' field on the metadata document served to instances. If no template string is specified, the 'user_state_url' field will not be added to the metadata document.")
	viperBindFlag("metadata.user_state_url", serveCmd.Flags().Lookup("user-state-url"))
}

func serve(ctx context.Context) {
	db := initTracingAndDB()

	logger.Infow("starting metadata server", "address", viper.GetString("listen"))

	lookupClient, err := getLookupClient(ctx)
	if err != nil {
		logger.Fatalw("error getting lookup service client", "error", err)
	}

	hs := &httpsrv.Server{
		Logger: logger.Desugar(),
		Listen: viper.GetString("listen"),
		Debug:  viper.GetBool("logging.debug"),
		DB:     db,
		AuthConfig: ginjwt.AuthConfig{
			Enabled:       viper.GetBool("oidc.enabled"),
			Audience:      viper.GetString("oidc.audience"),
			Issuer:        viper.GetString("oidc.issuer"),
			JWKSURI:       viper.GetString("oidc.jwksuri"),
			LogFields:     viper.GetStringSlice("oidc.log"), // TODO: We don't seem to be grabbing this from config?
			RolesClaim:    viper.GetString("oidc.claims.roles"),
			UsernameClaim: viper.GetString("oidc.claims.username"),
		},
		TrustedProxies: viper.GetStringSlice("gin.trustedproxies"),
		LookupEnabled:  viper.GetBool("lookup.enabled"),
		LookupClient:   lookupClient,
		TemplateFields: getTemplateFields(),
	}

	if err := hs.Run(); err != nil {
		logger.Fatalw("failed starting metadata server", "error", err)
	}
}

func getLookupClient(ctx context.Context) (*lookup.ServiceClient, error) {
	if viper.GetBool("lookup.enabled") {
		oauthConfig := clientcredentials.Config{
			ClientID:       viper.GetString("lookup.oidc.clientid"),
			ClientSecret:   viper.GetString("lookup.oidc.clientsecret"),
			TokenURL:       viper.GetString("lookup.oidc.issuer"),
			Scopes:         viper.GetStringSlice("lookup.oidc.scopes"),
			EndpointParams: url.Values{"audience": []string{viper.GetString("lookup.oidc.audience")}},
		}

		return lookup.NewClient(logger.Desugar(), viper.GetString("lookup.basepath"), oauthConfig.Client(ctx))
	}

	return nil, nil
}

func getTemplateFields() map[string]template.Template {
	templates := make(map[string]template.Template)

	phoneHomeURL := viper.GetString("metadata.phone_home_url")
	userStateURL := viper.GetString("metadata.user_state_url")

	if len(phoneHomeURL) > 0 {
		phoneHomeTempl, err := template.New("phoneHomeURL").Parse(phoneHomeURL)
		if err != nil {
			logger.Fatalf("failed to parse phone home URL template (%s)", phoneHomeURL, "error", err)
		}

		templates["phone_home_url"] = *phoneHomeTempl
	}

	if len(userStateURL) > 0 {
		userStateTempl, err := template.New("userStateURL").Parse(userStateURL)
		if err != nil {
			logger.Fatalf("failed to parse user state URL template (%s)", userStateURL, "error", err)
		}

		templates["user_state_url"] = *userStateTempl
	}

	return templates
}
