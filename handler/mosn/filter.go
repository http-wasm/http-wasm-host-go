package mosn

import (
	"context"
	"errors"
	"os"
	"strconv"

	"mosn.io/api"
	"mosn.io/mosn/pkg/log"

	httpwasm "github.com/http-wasm/http-wasm-host-go"
	"github.com/http-wasm/http-wasm-host-go/api/handler"
	internalhandler "github.com/http-wasm/http-wasm-host-go/internal/handler"
)

func init() {
	api.RegisterStream("httpwasm", factoryCreator)
}

var _ api.StreamFilterFactoryCreator = factoryCreator
var _ api.StreamFilterChainFactory = (*filterFactory)(nil)
var _ api.StreamSenderFilter = (*filter)(nil)
var _ api.StreamReceiverFilter = (*filter)(nil)

var errNoPath = errors.New("path is not set or is not a string")

func factoryCreator(config map[string]interface{}) (api.StreamFilterChainFactory, error) {
	p, ok := config["path"].(string)
	if !ok {
		return nil, errNoPath
	}
	conf, _ := config["config"].(string)
	code, err := os.ReadFile(p)
	if err != nil {
		return nil, err
	}
	ctx := context.Background()
	rt, err := internalhandler.NewRuntime(ctx, code, host{},
		httpwasm.GuestConfig([]byte(conf)),
		httpwasm.Logger(func(ctx context.Context, s string) {
			log.Proxy.Infof(ctx, "wasm: %s", s)
		}))
	if err != nil {
		return nil, err
	}

	return &filterFactory{
		rt: rt,
	}, nil
}

type filterFactory struct {
	rt *internalhandler.Runtime
}

func (f filterFactory) CreateFilterChain(context context.Context, callbacks api.StreamFilterChainFactoryCallbacks) {
	fr := &filter{
		rt: f.rt,
		ch: make(chan error, 1),
	}
	callbacks.AddStreamReceiverFilter(fr, api.BeforeRoute)
	callbacks.AddStreamSenderFilter(fr, api.BeforeSend)
}

type filter struct {
	rt *internalhandler.Runtime

	features handler.Features

	receiverFilterHandler api.StreamReceiverFilterHandler

	reqHeaders api.HeaderMap
	reqBody    api.IoBuffer

	respStatus  int
	respHeaders api.HeaderMap
	respBody    []byte

	nextCalled bool
	ch         chan error
}

func (f *filter) OnReceive(ctx context.Context, headers api.HeaderMap, body api.IoBuffer, _ api.HeaderMap) api.StreamFilterStatus {
	ctx = context.WithValue(ctx, filterKey{}, f)

	f.reqHeaders = headers
	f.reqBody = body

	go func() {
		f.ch <- f.rt.Handle(ctx)
	}()

	err := <-f.ch
	if err != nil {
		log.Proxy.Errorf(ctx, "wasm error: %v", err)
	}

	if f.nextCalled {
		return api.StreamFilterContinue
	}

	// TODO(anuraaga): All mosn filter examples pass the request headers when sending a hijack reply. Trying to send
	// f.respHeaders causes the hijack to be ignored. Figure out why.
	if respBody := f.respBody; len(respBody) > 0 {
		f.receiverFilterHandler.SendHijackReplyWithBody(f.respStatus, headers, string(respBody))
	} else {
		f.receiverFilterHandler.SendHijackReply(f.respStatus, headers)
	}
	return api.StreamFilterStop
}

func (f *filter) SetReceiveFilterHandler(handler api.StreamReceiverFilterHandler) {
	f.receiverFilterHandler = handler
}

func (f *filter) OnDestroy() {
}

func (f *filter) Append(ctx context.Context, headers api.HeaderMap, buf api.IoBuffer, trailers api.HeaderMap) api.StreamFilterStatus {
	if !f.nextCalled {
		// TODO(anuraaga): All mosn filter examples pass the request headers when sending a hijack reply. We replace
		// with response headers here until fixing that.
		// There is no headers.Clear() for some reason.
		headers.Range(func(key, value string) bool {
			headers.Del(key)
			return true
		})
		if f.respHeaders != nil {
			f.respHeaders.Range(func(key, value string) bool {
				headers.Set(key, value)
				return true
			})
		}
		return api.StreamFilterStop
	}

	ctx = context.WithValue(ctx, filterKey{}, f)

	if f.respHeaders != nil {
		f.respHeaders.Range(func(key, value string) bool {
			headers.Set(key, value)
			return true
		})
	}
	f.respHeaders = headers
	f.respBody = buf.Bytes()

	f.ch <- nil
	err := <-f.ch
	if err != nil {
		log.Proxy.Errorf(ctx, "wasm error: %v", err)
		return api.StreamFilterContinue
	}

	// BUG: mosn doesn't use psuedoheaders.
	if f.respStatus != 0 {
		headers.Set(":status", strconv.Itoa(f.respStatus))
	}

	// TODO(anuraaga): Optimize
	buf.Reset()
	_ = buf.Append(f.respBody)

	return api.StreamFilterContinue
}

func (f *filter) SetSenderFilterHandler(handler api.StreamSenderFilterHandler) {
}

type filterKey struct{}

func (s *filter) enableFeatures(features handler.Features) {
	s.features = s.features.WithEnabled(features)
}

func filterFromContext(ctx context.Context) *filter {
	return ctx.Value(filterKey{}).(*filter)
}
