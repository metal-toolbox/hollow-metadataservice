package httpsrv

import (
	"net/http"
	"os"
	"text/template"
	"time"

	"github.com/gin-contrib/cors"
	ginzap "github.com/gin-contrib/zap"
	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
	ginprometheus "github.com/zsais/go-gin-prometheus"
	"go.hollow.sh/toolbox/ginjwt"
	"go.hollow.sh/toolbox/version"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
	"go.opentelemetry.io/otel"
	"go.uber.org/zap"

	"go.hollow.sh/metadataservice/internal/lookup"
	v1api "go.hollow.sh/metadataservice/pkg/api/v1"
)

// Server contains the HTTP server configuration
type Server struct {
	Logger         *zap.Logger
	Listen         string
	Debug          bool
	DB             *sqlx.DB
	AuthConfig     ginjwt.AuthConfig
	TrustedProxies []string
	LookupEnabled  bool
	LookupClient   lookup.Client
	TemplateFields map[string]template.Template
}

var (
	readTimeout  = 30 * time.Second
	writeTimeout = 30 * time.Second
	corsMaxAge   = 12 * time.Hour
)

func (s *Server) setup() *gin.Engine {
	var (
		authMW *ginjwt.Middleware
		err    error
	)

	authMW, err = ginjwt.NewAuthMiddleware(s.AuthConfig)
	if err != nil {
		s.Logger.Sugar().Fatal("failed to initialize auth middleware", "error", err)
	}

	// Setup default gin router
	r := gin.New()

	// Set the trusted proxies, if they were specified by config
	if len(s.TrustedProxies) > 0 {
		err = r.SetTrustedProxies(s.TrustedProxies)
		if err != nil {
			s.Logger.Sugar().Fatal("failed to set gin trusted proxies", "error", err)
		}
	}

	r.Use(cors.New(cors.Config{
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD"},
		AllowHeaders:     []string{"Origin", "Content-Length", "Content-Type", "Authorization"},
		AllowAllOrigins:  true,
		AllowCredentials: true,
		MaxAge:           corsMaxAge,
	}))

	p := ginprometheus.NewPrometheus("gin")

	// Remove any params from the URL string to keep the number of labels down
	p.ReqCntURLLabelMappingFn = func(c *gin.Context) string {
		return c.FullPath()
	}

	p.Use(r)

	r.Use(ginzap.Logger(s.Logger.With(zap.String("component", "httpsrv")), ginzap.WithTimeFormat(time.RFC3339),
		ginzap.WithUTC(true),
		ginzap.WithCustomFields(
			func(c *gin.Context) zap.Field { return zap.String("jwt_subject", ginjwt.GetSubject(c)) },
			func(c *gin.Context) zap.Field { return zap.String("jwt_user", ginjwt.GetUser(c)) },
		),
	))
	r.Use(ginzap.RecoveryWithZap(s.Logger.With(zap.String("component", "httpsrv")), true))

	tp := otel.GetTracerProvider()
	if tp != nil {
		hostname, err := os.Hostname()
		if err != nil {
			hostname = "unknown"
		}

		r.Use(otelgin.Middleware(hostname, otelgin.WithTracerProvider(tp)))
	}

	// Version endpoint returns build information
	r.GET("/version", s.version)

	// Health endpoints
	r.GET("/healthz", s.livenessCheck)
	r.GET("/healthz/liveness", s.livenessCheck)
	r.GET("/healthz/readiness", s.readinessCheck)

	v1Rtr := v1api.Router{AuthMW: authMW, DB: s.DB, Logger: s.Logger, LookupEnabled: s.LookupEnabled, LookupClient: s.LookupClient, TemplateFields: s.TemplateFields}

	// Host our latest version of the API under / in addition to /api/v*
	latest := r.Group("/")
	{
		v1Rtr.Routes(latest)
	}

	v1 := r.Group(v1api.V1URI)
	{
		v1Rtr.Routes(v1)
	}

	ec2 := r.Group(v1api.V20090404URI)
	{
		v1Rtr.Ec2Routes(ec2)
	}

	r.NoRoute(func(c *gin.Context) {
		c.JSON(http.StatusNotFound, gin.H{"message": "invalid request - route not found"})
	})

	return r
}

// NewServer returns a configured server
func (s *Server) NewServer() *http.Server {
	if !s.Debug {
		gin.SetMode(gin.ReleaseMode)
	}

	return &http.Server{
		Handler:      s.setup(),
		Addr:         s.Listen,
		ReadTimeout:  readTimeout,
		WriteTimeout: writeTimeout,
	}
}

// Run will start the server listening on the specified address
func (s *Server) Run() error {
	if !s.Debug {
		gin.SetMode(gin.ReleaseMode)
	}

	return s.setup().Run(s.Listen)
}

// livenessCheck ensures that the server is up and responding
func (s *Server) livenessCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "UP",
	})
}

// readinessCheck ensures that the server is up and that we are able to process
// requests. Currently our only dependency is the DB so we just ensure that it
// is responding.
func (s *Server) readinessCheck(c *gin.Context) {
	if err := s.DB.PingContext(c.Request.Context()); err != nil {
		s.Logger.Sugar().Errorf("readiness check db ping failed", "err", err)
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status": "DOWN",
		})

		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "UP",
	})
}

// version returns the metadataservice build information
func (s *Server) version(c *gin.Context) {
	c.JSON(http.StatusOK, version.String())
}
