package http2

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gig/orion-go-sdk/interfaces"

	"github.com/gig/orion-go-sdk/env"

	"github.com/gig/orion-go-sdk/transport"
	"github.com/go-chi/chi"
	"golang.org/x/net/http2"
)

// Prepares a transport for http2
func PrepareHttp2Transport() *http2.Transport {
	return &http2.Transport{
		TLSClientConfig:    TLSConfig(),
		DisableCompression: true,
		AllowHTTP:          false,
	}
}

// Transport object
type Transport struct {
	codec        interfaces.Codec
	client       http.Client
	server       http.Server
	host         string
	port         int
	gatewayUrl   string
	root         chi.Router
	closeHandler func()
	open         bool
	poolLimit    chan struct{}
}

var _ interfaces.Transport = (*Transport)(nil)
var _ interfaces.Register = (*Transport)(nil)

// New returns client for HTTP2 messaging
func New(options ...transport.Option) *Transport {
	optionsObj := transport.Options{}

	for _, o := range options {
		o(&optionsObj)
	}

	root := chi.NewRouter()

	// Default pool limit of 1:
	poolLimit := 1
	if optionsObj.PoolThreadSize > 1 {
		poolLimit = optionsObj.PoolThreadSize
	}

	return &Transport{client: http.Client{
		Transport: PrepareHttp2Transport(),
	}, server: http.Server{
		Addr:         ":" + strconv.Itoa(optionsObj.Http2Port),
		ReadTimeout:  1 * time.Minute,
		WriteTimeout: 1 * time.Minute,
		TLSConfig:    TLSConfig(),
	},
		gatewayUrl: env.Get("GATEWAY_URL", "https://localhost"),
		root:       root,
		host:       optionsObj.URL,
		port:       optionsObj.Http2Port,
		poolLimit:  make(chan struct{}, poolLimit),
		codec:      optionsObj.Codec,
	}

}

// Listen to nats
func (t *Transport) Listen(callback func()) {
	crtFilename := env.Get("ORION_DEFAULT_SSL_CERT", "")
	keyFilename := env.Get("ORION_DEFAULT_SSL_KEY", "")

	done := make(chan struct{})
	go func() {
		t.server.Handler = t.root
		t.open = true
		if err := t.server.ListenAndServeTLS(crtFilename, keyFilename); err != nil {
			t.open = false
			if t.closeHandler != nil {
				t.closeHandler()
			}
			done <- struct{}{}
		}
	}()
	callback()
	<-done
}

// Publish to topic
func (t *Transport) Publish(route string, data []byte) error {
	buffer := bytes.NewBuffer(data)

	req, err := http.NewRequest("POST", t.gatewayUrl+route, buffer)
	if err != nil {
		return err
	}

	req.Header.Set("scheme", "https")
	req.Header.Set("Content-Type", "application/json")

	_, err = t.client.Do(req)

	return err
}

// Subscribe for a topic given a handler. For http2, this method can't fail
func (t *Transport) Subscribe(topic string, group string, handler func([]byte)) error {
	route := "/" + group + "/" + strings.ReplaceAll(topic, ":", "/")
	t.root.Post(route, func(writer http.ResponseWriter, request *http.Request) {
		defer request.Body.Close()
		data, err := ioutil.ReadAll(request.Body)
		if err != nil {
			_, _ = writer.Write([]byte(err.Error()))
			writer.WriteHeader(500)
			return
		}

		_, _ = writer.Write([]byte("OK"))
		writer.WriteHeader(200)

		// Execute the logic concurrently, and finish the POST without waiting for the answer.
		go func() {
			t.poolControl(func() {
				handler(data)
			})
		}()
	})
	return nil
}

// SubscribeForRawMsg for topic
func (t *Transport) SubscribeForRawMsg(topic string, group string, handler func(interface{})) error {
	// TODO: will not implement
	return nil
}

// Handle path
func (t *Transport) Handle(path string, group string, handler func([]byte, func(interfaces.Response))) error {
	route := strings.ReplaceAll(path, ".", "/")
	route = strings.ReplaceAll(route, "//", "/")

	route = "/" + group + "/" + route

	t.root.HandleFunc(route, func(writer http.ResponseWriter, request *http.Request) {
		bodyData, err := ioutil.ReadAll(request.Body)

		if err != nil {
			defer request.Body.Close()
			_, _ = writer.Write([]byte(err.Error()))
			writer.WriteHeader(500)
			return
		}

		t.poolControl(func() {
			handler(bodyData, func(res interfaces.Response) {
				defer request.Body.Close()

				encoded, err := t.codec.Encode(res)

				if err != nil {
					writer.WriteHeader(500)
				}

				_, err = writer.Write(encoded)
				if err != nil {
					writer.WriteHeader(500)
				}

				if err = res.GetError(); err != nil {
					writer.WriteHeader(500)
				}
			})
		})
	})

	return nil
}

// Request path
func (t *Transport) Request(path string, payload []byte, timeOut int) ([]byte, error) {
	route := "/" + strings.ReplaceAll(path, ":", "/")
	buffer := bytes.NewBuffer(payload)

	req, err := http.NewRequest("POST", t.gatewayUrl+route, buffer)
	if err != nil {
		return nil, err
	}

	req.Header.Set("scheme", "https")
	// TODO: coupled to the codec
	req.Header.Set("Content-Type", "application/json")

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeOut)*time.Millisecond)
	defer cancel()

	res, err := t.client.Do(req.WithContext(ctx))

	if err != nil {
		return nil, err
	}

	defer res.Body.Close()
	responseBytes, err := ioutil.ReadAll(res.Body)

	return responseBytes, err
}

// Close connection
func (t *Transport) Close() {
	t.server.Close()
	t.client.CloseIdleConnections()
}

// OnClose adds a handler to NATS close event
func (t *Transport) OnClose(handler interface{}) {
	if callback, ok := handler.(func()); ok {
		t.closeHandler = callback
	}
}

// IsOpen returns whether the transport connection is open and ready to be used.
func (t *Transport) IsOpen() bool {
	return t.open
}

type TraefikPayload struct {
	Name    string
	Tags    []string
	Address string
	Port    int
	Check   struct {
		Args                           []string
		Interval                       string
		Timeout                        string
		DeregisterCriticalServiceAfter string
	}
}

func (t *Transport) Register(serviceName string, instanceName string, prefixList []string) error {
	registryUrl := env.Get("CONSUL_API_URL", "https://localhost:8501")

	payload := TraefikPayload{
		Name: instanceName,
		Tags: []string{
			"traefik.http.routers.router_" + serviceName + ".entrypoints=websecure",
			"traefik.http.routers.router_" + serviceName + ".rule=PathPrefix(`/" + serviceName + "/`)",
			"traefik.http.routers.router_" + serviceName + ".service=" + serviceName,
			"traefik.http.routers.router_" + serviceName + ".tls=true",
			"traefik.http.services." + serviceName + ".loadbalancer.server.scheme=https",
		},
		Address: t.host,
		Port:    t.port,
		Check: struct {
			Args                           []string
			Interval                       string
			Timeout                        string
			DeregisterCriticalServiceAfter string
		}{
			//Args: []string{"sh", "-c", "exit 0"},
			Args:                           []string{"sh", "-c", "(curl -ks https://" + t.host + ":" + strconv.Itoa(t.port) + "/" + instanceName + "/healthcheck 2>&1 || echo \"CRIT\") | awk '$1 ~ /\"summary\":\"OK\"/ {print; exit 0} {exit 2}'"},
			Interval:                       "500ms",
			Timeout:                        "200ms",
			DeregisterCriticalServiceAfter: "1s",
		},
	}

	for _, prefix := range prefixList {
		nPrefix := strings.ReplaceAll(prefix, ".", "_")
		nPrefix = strings.ReplaceAll(nPrefix, "/", "_")

		routerName := "router_" + nPrefix
		payload.Tags = append(payload.Tags, "traefik.http.routers."+routerName+".rule=PathPrefix(`/"+nPrefix+"/`)")
		payload.Tags = append(payload.Tags, "traefik.http.routers."+routerName+".service="+serviceName)
		payload.Tags = append(payload.Tags, "traefik.http.routers."+routerName+".tls=true")
	}

	data, err := json.Marshal(payload)

	if err != nil {
		return err
	}

	buffer := bytes.NewReader(data)

	req, err := http.NewRequest("PUT", registryUrl+"/v1/agent/service/register?replace-existing-checks=true", buffer)

	if err != nil {
		return err
	}

	req.Header.Set("Content.Type", "application/json")
	req.ContentLength = int64(len(data))

	_, err = t.client.Do(req)

	return err
}

// Blocking pool control
func (t *Transport) poolControl(f func()) {
	t.poolLimit <- struct{}{}
	defer func() {
		if r := recover(); r != nil {
			// TODO: What to do with the error?
			fmt.Println(r)
		}
		<-t.poolLimit
	}()
	f()
}

func (t *Transport) GetRootRouter() chi.Router {
	return t.root
}
