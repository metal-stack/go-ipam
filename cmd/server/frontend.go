package main

import (
	"log/slog"
	"net/http"
	"strings"

	"connectrpc.com/connect"
	"github.com/donseba/go-htmx"
	apiv1 "github.com/metal-stack/go-ipam/api/v1"
	"github.com/metal-stack/go-ipam/pkg/service"
	"google.golang.org/protobuf/encoding/protojson"
)

type App struct {
	htmx        *htmx.HTMX
	log         *slog.Logger
	mux         *http.ServeMux
	ipamService *service.IPAMService
}

func NewFrontend(log *slog.Logger, ipamService *service.IPAMService, mux *http.ServeMux) *App {
	return &App{
		htmx:        htmx.New(),
		log:         log,
		mux:         mux,
		ipamService: ipamService,
	}
}

func (a *App) Serve() {
	a.mux.Handle("/", http.HandlerFunc(a.Home))
	a.mux.Handle("/prefix", http.HandlerFunc(a.Create))
}

func (a *App) Home(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "static/index.html")
}

func (a *App) Create(w http.ResponseWriter, r *http.Request) {
	a.log.Debug("create called", "header", r.Header, "body", r.Body)
	cidr := strings.ToLower(r.PostFormValue("cidr"))
	if cidr == "" {
		_, _ = w.Write([]byte("Please enter a cidr."))
		return
	}
	ctx := r.Context()
	resp, err := a.ipamService.CreatePrefix(ctx, connect.NewRequest(&apiv1.CreatePrefixRequest{
		Cidr: cidr,
	}))

	if err != nil {
		a.log.Error("unable to create prefix", "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}

	json, err := protojson.Marshal(resp.Msg.Prefix)
	if err != nil {
		a.log.Error("unable to marshall prefix", "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}

	// TODO return json and use mustache template on the frontend
	_, _ = w.Write(json)
}
