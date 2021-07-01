package api

import (
	"context"
	"github.com/kostiamol/centerms/proto"
	"net/http"
	"time"

	"github.com/kostiamol/centerms/store/model"
	"google.golang.org/grpc"

	"github.com/kostiamol/centerms/metric"

	"github.com/kostiamol/centerms/log"

	"github.com/kostiamol/centerms/svc"

	"fmt"

	"github.com/gorilla/mux"
	"github.com/rs/cors"
)

type (
	// CfgProvider is a contract for the configuration provider.
	CfgProvider interface {
		GetInitCfg(m *model.Meta) (*model.Cfg, error)
		GetCfg(devID string, t model.Type) (*model.Cfg, error)
		SetCfg(devID string, t model.Type, c *model.Cfg) error
	}

	// DataProvider is a contract for the data provider.
	DataProvider interface {
		GetDevsData() ([]model.Data, error)
		GetDevData(devID string) (*model.Data, error)
		SaveData(*model.Data) error
	}

	// Publisher .
	Publisher interface {
		Publish(msg interface{}, channel string) (int64, error)
	}

	// Cfg is used to initialize an instance of api.
	Cfg struct {
		Log                log.Logger
		Ctrl               svc.Ctrl
		Metric             *metric.Metric
		PubChan            chan<- *model.Cfg
		PortRPC            uint32
		PortREST           uint32
		CfgProvider        CfgProvider
		DataProvider       DataProvider
		Retry              time.Duration
		PublicKey          string
		PrivateKey         string
		TerminationTimeout time.Duration
	}

	// api includes both rest and grpc.
	api struct {
		proto.UnimplementedCenterServiceServer
		log                log.Logger
		ctrl               svc.Ctrl
		metric             *metric.Metric
		pubChan            chan<- *model.Cfg
		portRPC            uint32
		portREST           uint32
		cfgProvider        CfgProvider
		dataProvider       DataProvider
		retry              time.Duration
		router             *mux.Router
		token              *tokenValidator
		publicKey          string
		privateKey         string
		terminationTimeout time.Duration
		httpServer         *http.Server
		grpcServer         *grpc.Server
	}
)

// New creates and initializes a new instance of api.
func New(c *Cfg) *api { // nolint
	return &api{
		log:                c.Log.With("component", "api"),
		ctrl:               c.Ctrl,
		metric:             c.Metric,
		pubChan:            c.PubChan,
		portRPC:            c.PortRPC,
		portREST:           c.PortREST,
		cfgProvider:        c.CfgProvider,
		dataProvider:       c.DataProvider,
		retry:              c.Retry,
		publicKey:          c.PublicKey,
		privateKey:         c.PrivateKey,
		terminationTimeout: c.TerminationTimeout,
	}
}

// Run launches the service by running goroutines for listening to the service termination and queries
// from the web client.
func (a *api) Run() {
	a.log.With("event", log.EventComponentStarted).
		Infof("rpc port [%d] rest port [%d]", a.portRPC, a.portREST)

	var err error
	if a.token, err = newTokenValidator(a.publicKey); err != nil {
		a.log.Errorf("func newTokenValidator: %s", err)
		a.ctrl.Terminate()
	}

	go a.listenToTermination()

	go a.serveRPC()

	c := cors.New(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowCredentials: true,
		AllowedMethods:   []string{"GET", "HEAD", "PATCH", "OPTIONS"},
		AllowedHeaders:   []string{"*"},
	})
	a.httpServer = &http.Server{
		Handler: c.Handler(a.router),
		Addr:    fmt.Sprintf(":%d", a.portREST),
	}
	a.router = mux.NewRouter()
	a.registerRoutes()
	a.serveHTTP()
}

func (a *api) listenToTermination() {
	<-a.ctrl.StopChan

	_ = a.log.Flush()

	a.grpcServer.GracefulStop()

	ctx, cancel := context.WithTimeout(context.Background(), a.terminationTimeout)
	defer cancel()
	a.httpServer.SetKeepAlivesEnabled(false)
	if err := a.httpServer.Shutdown(ctx); err != nil {
		a.log.Errorf("httpServer shutdown: %s", err)
	}

	a.log.With("event", log.EventComponentShutdown).Infof("")
}

func (a *api) registerRoutes() {
	middleware := []func(next http.HandlerFunc, name string, l log.Logger) http.HandlerFunc{
		requestLogger,
		a.token.validator,
		a.metric.TimeTracker,
	}

	a.registerRoute(http.MethodGet, "/health", a.health)
	a.registerRoute(http.MethodGet, "/metrics", a.metric.RouterHandlerHTTP())

	a.registerRoute(http.MethodGet, "/v1/token", getTokenHandler)

	a.registerRoute(http.MethodGet, "/v1/device", a.getDevsDataHandler, middleware...)
	a.registerRoute(http.MethodGet, "/v1/device/{dev_id}/data", a.getDevDataHandler, middleware...)
	a.registerRoute(http.MethodGet, "/v1/device/{dev_id}/cfg", a.getDevCfgHandler, middleware...)
	a.registerRoute(http.MethodPatch, "/v1/device/{dev_id}/cfg", a.patchDevCfgHandler, middleware...)
}

func (a *api) serveHTTP() {
	if err := a.httpServer.ListenAndServe(); err != nil {
		a.log.Errorf("func ListenAndServe: %s", err)
		a.ctrl.Terminate()
	}
}
