package baseapp

import (
	"fmt"

	gogogrpc "github.com/gogo/protobuf/grpc"
	abci "github.com/tendermint/tendermint/abci/types"
	"google.golang.org/grpc"
	"google.golang.org/grpc/encoding"
	"google.golang.org/grpc/encoding/proto"

	"github.com/cosmos/cosmos-sdk/client/grpc/reflection"
	"github.com/cosmos/cosmos-sdk/client/grpc/simulate"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

var protoCodec = encoding.GetCodec(proto.Name)

// GRPCQueryRouter routes ABCI Query requests to GRPC handlers
type GRPCQueryRouter struct {
	routes            map[string]GRPCQueryHandler
	interfaceRegistry codectypes.InterfaceRegistry
	serviceData       []serviceData
}

// serviceData represents a gRPC service, along with its handler.
type serviceData struct {
	serviceDesc *grpc.ServiceDesc
	handler     interface{}
}

var _ gogogrpc.Server = &GRPCQueryRouter{}

// NewGRPCQueryRouter creates a new GRPCQueryRouter
func NewGRPCQueryRouter() *GRPCQueryRouter {
	return &GRPCQueryRouter{
		routes: map[string]GRPCQueryHandler{},
	}
}

// GRPCQueryHandler defines a function type which handles ABCI Query requests
// using gRPC
type GRPCQueryHandler = func(ctx sdk.Context, req abci.RequestQuery) (abci.ResponseQuery, error)

// Route returns the GRPCQueryHandler for a given query route path or nil
// if not found
func (qrt *GRPCQueryRouter) Route(path string) GRPCQueryHandler {
	handler, found := qrt.routes[path]
	if !found {
		return nil
	}
	return handler
}

// RegisterService implements the gRPC Server.RegisterService method. sd is a gRPC
// service description, handler is an object which implements that gRPC service
func (qrt *GRPCQueryRouter) RegisterService(sd *grpc.ServiceDesc, handler interface{}) {
	// adds a top-level query handler based on the gRPC service name
	for _, method := range sd.Methods {
		fqName := fmt.Sprintf("/%s/%s", sd.ServiceName, method.MethodName)
		methodHandler := method.Handler

		qrt.routes[fqName] = func(ctx sdk.Context, req abci.RequestQuery) (abci.ResponseQuery, error) {
			// call the method handler from the service description with the handler object,
			// a wrapped sdk.Context with proto-unmarshaled data from the ABCI request data
			res, err := methodHandler(handler, sdk.WrapSDKContext(ctx), func(i interface{}) error {
				err := protoCodec.Unmarshal(req.Data, i)
				if err != nil {
					return err
				}
				if qrt.interfaceRegistry != nil {
					return codectypes.UnpackInterfaces(i, qrt.interfaceRegistry)
				}
				return nil
			}, nil)
			if err != nil {
				return abci.ResponseQuery{}, err
			}

			// proto marshal the result bytes
			resBytes, err := protoCodec.Marshal(res)
			if err != nil {
				return abci.ResponseQuery{}, err
			}

			// return the result bytes as the response value
			return abci.ResponseQuery{
				Height: req.Height,
				Value:  resBytes,
			}, nil
		}
	}

	qrt.serviceData = append(qrt.serviceData, serviceData{
		serviceDesc: sd,
		handler:     handler,
	})
}

// SetInterfaceRegistry sets the interface registry for the router. This will
// also register the interface reflection gRPC service.
func (qrt *GRPCQueryRouter) SetInterfaceRegistry(interfaceRegistry codectypes.InterfaceRegistry) {
	qrt.interfaceRegistry = interfaceRegistry

	// Once we have an interface registry, we can register the interface
	// registry reflection gRPC service.
	reflection.RegisterReflectionServiceServer(
		qrt,
		reflection.NewReflectionServiceServer(interfaceRegistry),
	)
}

// RegisterSimulateService registers the simulate service on the gRPC router.
func (qrt *GRPCQueryRouter) RegisterSimulateService(
	simulateFn simulate.SimulateFn,
	interfaceRegistry codectypes.InterfaceRegistry,
	pubkeyCodec cryptotypes.PublicKeyCodec,
) {
	simulate.RegisterSimulateServiceServer(
		qrt,
		simulate.NewSimulateServer(simulateFn, qrt.interfaceRegistry, pubkeyCodec),
	)
}
