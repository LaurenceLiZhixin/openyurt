package site

import (
	"bytes"
	"context"
	"fmt"
	"github.com/openyurtio/openyurt/pkg/yurthub/proxy/remote"
	"k8s.io/apimachinery/pkg/util/uuid"
	apirequest "k8s.io/apiserver/pkg/endpoints/request"
	"k8s.io/client-go/rest"
	"net/http"
	"sync"
)

type watchCtx struct {
	ctx  context.Context
	info *apirequest.RequestInfo
	lb   remote.LoadBalancer

	subscriberWriters     map[string]http.ResponseWriter
	subscriberWritersLock sync.Mutex

	rspWriter *proxyResponseWriter
}

func newWatchCtx(info *apirequest.RequestInfo, lb remote.LoadBalancer) *watchCtx {
	return &watchCtx{
		ctx:               context.Background(),
		info:              info,
		lb:                lb,
		subscriberWriters: make(map[string]http.ResponseWriter, 0),
	}
}

func (w *watchCtx) runWatch() error {
	restReq := w.getRestRequestFromInfo()
	// create watch request
	req, err := http.NewRequest(http.MethodGet, restReq.URL().String()+"?watch=true", nil)
	if err != nil {
		return err
	}
	req.RequestURI = restReq.URL().String() + "?watch=true"

	w.rspWriter = newProxyResponseWriter(w.info)
	go func() {
		w.lb.ServeHTTP(w.rspWriter, req)
	}()

	for {
		select {
		case <-w.ctx.Done():
			return nil
		case buf := <-w.rspWriter.outputChan:
			w.subscriberWritersLock.Lock()
			func() {
				defer func() {
					recover()
					w.subscriberWritersLock.Unlock()
				}()
				for _, rw := range w.subscriberWriters {
					if _, err := rw.Write(buf.Bytes()); err != nil {
						fmt.Println("Write Error", err.Error())
					}
				}
			}()
		}
	}
}

func (w *watchCtx) RunSubscribe(rw http.ResponseWriter, ctx context.Context) {
	rw.Header().Set("Transfer-Encoding", "chunked")
	rw.WriteHeader(http.StatusOK)
	uid := uuid.NewUUID()

	w.subscriberWritersLock.Lock()
	w.subscriberWriters[string(uid)] = rw
	w.subscriberWritersLock.Unlock()

	defer func() {
		w.subscriberWritersLock.Lock()
		delete(w.subscriberWriters, string(uid))
		w.subscriberWritersLock.Unlock()
	}()

	<-ctx.Done()
}

func (w *watchCtx) getRestRequestFromInfo() rest.Request {
	restRequest := rest.NewRequest(&rest.RESTClient{})
	restRequest.Prefix("/api/v1")
	if w.info.Namespace != "" {
		restRequest.Namespace(w.info.Namespace)
	}
	if w.info.Name != "" {
		restRequest.Name(w.info.Name)
	}
	if w.info.Resource != "" {
		restRequest.Resource(w.info.Resource)
	}
	if w.info.Subresource != "" {
		restRequest.SubResource(w.info.Subresource)
	}

	//restRequest.Timeout()  // todo
	return *restRequest
}

type proxyResponseWriter struct {
	statusCode int
	header     http.Header
	outputChan chan *bytes.Buffer
	info       *apirequest.RequestInfo
}

func newProxyResponseWriter(info *apirequest.RequestInfo) *proxyResponseWriter {
	return &proxyResponseWriter{
		header:     make(http.Header),
		outputChan: make(chan *bytes.Buffer),
		info:       info,
	}
}

func (p *proxyResponseWriter) Header() http.Header {
	return p.header
}

func (p *proxyResponseWriter) Write(data []byte) (int, error) {
	p.outputChan <- bytes.NewBuffer(data)
	return len(data), nil
}

func (p *proxyResponseWriter) WriteHeader(statusCode int) {
	p.statusCode = statusCode
}

var _ http.ResponseWriter = &proxyResponseWriter{}
