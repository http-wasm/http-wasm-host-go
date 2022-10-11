package httpwasm

import (
	"context"
	"log"
	"os"
	"strconv"

	"mosn.io/api"

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

func factoryCreator(config map[string]interface{}) (api.StreamFilterChainFactory, error) {
	p := config["path"].(string)
	code, err := os.ReadFile(p)
	if err != nil {
		return nil, err
	}
	ctx := context.Background()
	rt, err := internalhandler.NewRuntime(ctx, code, host{}, httpwasm.Logger(func(ctx context.Context, s string) {
		log.Println(s)
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

	receiverFilterHandler api.StreamReceiverFilterHandler
	senderFilterHandler   api.StreamSenderFilterHandler

	features handler.Features

	respStatus int
	respBody   []byte

	nextCalled bool
	ch         chan error
}

func (f *filter) OnReceive(ctx context.Context, _ api.HeaderMap, _ api.IoBuffer, _ api.HeaderMap) api.StreamFilterStatus {
	ctx = context.WithValue(ctx, filterKey{}, f)

	go func() {
		f.ch <- f.rt.Handle(ctx)
	}()

	err := <-f.ch
	if err != nil {
		// TODO(anuraaga): Log error in a mosn way properly
		panic(err)
	}

	if f.nextCalled {
		return api.StreamFilterContinue
	}

	f.receiverFilterHandler.SendHijackReplyWithBody(f.respStatus, nil, string(f.respBody))
	return api.StreamFilterStop
}

func (f *filter) SetReceiveFilterHandler(handler api.StreamReceiverFilterHandler) {
	f.receiverFilterHandler = handler
}

func (f *filter) OnDestroy() {
}

func (f *filter) Append(ctx context.Context, headers api.HeaderMap, buf api.IoBuffer, trailers api.HeaderMap) api.StreamFilterStatus {
	ctx = context.WithValue(ctx, filterKey{}, f)

	if !f.nextCalled {
		// Response already sent from receiver.
		return api.StreamFilterStop
	}

	f.ch <- nil
	err := <-f.ch
	if err != nil {
		// TODO(anuraaga): Log error in a mosn way properly
		panic(err)
	}

	if f.respStatus != 0 {
		headers.Set(":status", strconv.Itoa(f.respStatus))
	}

	if f.respBody != nil {
		buf.Reset()
		_ = buf.Append(f.respBody)
	}

	return api.StreamFilterContinue
}

func (f *filter) SetSenderFilterHandler(handler api.StreamSenderFilterHandler) {
	f.senderFilterHandler = handler
}

type filterKey struct{}

func (s *filter) enableFeatures(features handler.Features) {
	s.features = s.features.WithEnabled(features)
}

func filterFromContext(ctx context.Context) *filter {
	return ctx.Value(filterKey{}).(*filter)
}
