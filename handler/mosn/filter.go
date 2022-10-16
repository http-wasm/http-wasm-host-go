package wasm

import (
	"context"
	"errors"
	"os"
	"runtime"

	"mosn.io/api"
	"mosn.io/mosn/pkg/log"
	mosnhttp "mosn.io/mosn/pkg/protocol/http"

	httpwasm "github.com/http-wasm/http-wasm-host-go"
	"github.com/http-wasm/http-wasm-host-go/api/handler"
	internalhandler "github.com/http-wasm/http-wasm-host-go/internal/handler"
)

func init() {
	// There's no API to configure a StreamFilter without using the global registry.
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
	runtime.SetFinalizer(rt, func(rt *internalhandler.Runtime) {
		rt.Close(context.Background())
	})
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

	respHeaders api.HeaderMap
	statusCode  int
	respBody    []byte

	nextCalled bool
	ch         chan error
}

func (f *filter) OnReceive(ctx context.Context, headers api.HeaderMap, body api.IoBuffer, _ api.HeaderMap) api.StreamFilterStatus {
	ctx = context.WithValue(ctx, filterKey{}, f)

	f.reqHeaders = headers
	f.reqBody = body

	go func() {
		// Handle dispatches to wazero which recovers any panics in host
		// functions to an error return. Hence, we don't need to recover here.
		f.ch <- f.rt.Handle(ctx)
	}()

	// Wait for the guest, running in a goroutine, to signal for us to continue. This will either be
	// within an invocation of next() or when returning from the guest if next() was not called.
	err := <-f.ch

	if err != nil {
		log.Proxy.Errorf(ctx, "wasm error: %v", err)
	}

	if f.nextCalled {
		return api.StreamFilterContinue
	}

	// TODO(anuraaga): All mosn filter examples pass the request headers when sending a hijack reply. Trying to send
	// f.respHeaders causes the hijack to be ignored. Figure out why.
	var statusCode int
	if resp, ok := f.respHeaders.(mosnhttp.ResponseHeader); ok {
		statusCode = resp.StatusCode()
	} else {
		statusCode = f.statusCode
	}
	if respBody := f.respBody; len(respBody) > 0 {
		f.receiverFilterHandler.SendHijackReplyWithBody(statusCode, headers, string(respBody))
	} else {
		f.receiverFilterHandler.SendHijackReply(statusCode, headers)
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

	// The guest called next, and as we have the upstream response now, we can resume it by
	// signaling the channel.
	f.ch <- nil

	// The channel will return when the guest completes.
	err := <-f.ch
	if err != nil {
		log.Proxy.Errorf(ctx, "wasm error: %v", err)
		return api.StreamFilterContinue
	}

	// TODO(anuraaga): Optimize
	buf.Reset()
	_ = buf.Append(f.respBody)

	return api.StreamFilterContinue
}

func (f *filter) SetSenderFilterHandler(handler api.StreamSenderFilterHandler) {
}

type filterKey struct{}

func (f *filter) enableFeatures(features handler.Features) {
	f.features = f.features.WithEnabled(features)
}

func filterFromContext(ctx context.Context) *filter {
	return ctx.Value(filterKey{}).(*filter)
}

// writerFunc implements io.Writer with a func.
type writerFunc func(p []byte) (n int, err error)

func (f writerFunc) Write(p []byte) (n int, err error) {
	return f(p)
}

func (f *filter) WriteRequestBody(p []byte) (n int, err error) {
	n = len(p)
	err = f.reqBody.Append(p)
	return
}

func (f *filter) WriteResponseBody(p []byte) (n int, err error) {
	n = len(p)
	f.respBody = append(f.respBody, p...)
	return
}
