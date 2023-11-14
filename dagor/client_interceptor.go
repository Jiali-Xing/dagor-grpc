package dagor

import (
	"context"
	"errors"
	"strconv"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type ThresholdTable struct {
	BStar int
	UStar int
}

func (d *Dagor) UnaryInterceptorClient(ctx context.Context, method string, req interface{}, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
	// if d.isEnduser, attach user id to metadata and send request
	if d.isEnduser {
		ctx = metadata.AppendToOutgoingContext(ctx, "user-id", d.uuid)
		err := invoker(ctx, method, req, reply, cc, opts...)
		if err != nil {
			logger("[End User] %s is an end user, req got error: %v", d.uuid, err)
			return err
		}
		logger("[End User] %s is an end user, req completed", d.uuid)
		return nil
	}

	// Extracting metadata
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return errors.New("could not retrieve metadata from context")
	}

	// Extracting method name and determining B value
	methodName, ok := md["method"]
	if !ok || len(methodName) == 0 {
		return errors.New("method name not found in metadata")
	}

	// Check if B and U are in the metadata
	BValues, BExists := md["B"]
	UValues, UExists := md["U"]

	if !BExists || !UExists {
		// if B or U not in metadata, this client is end user
		// mark this client as end user
		d.isEnduser = true
		logger("No B or U received. Client %s marked as an end user", d.uuid)
		// attach user id to metadata
		ctx = metadata.AppendToOutgoingContext(ctx, "user-id", d.uuid)
		err := invoker(ctx, method, req, reply, cc, opts...)
		if err != nil {
			return err
		}
		return nil
	}

	var B, U int
	// otherwise, this client is a DAGOR node in the service app
	B, _ = strconv.Atoi(BValues[0])
	U, _ = strconv.Atoi(UValues[0])
	// check if B and U against threshold table before sending sub-request
	// Thresholding

	val, ok := d.thresholdTable.Load(methodName[0])
	if ok {
		threshold := val.(thresholdVal)
		if B < threshold.Bstar || U < threshold.Ustar {
			logger("[Ratelimiting] B %d or U %d value below the threshold B* %d or U* %d, request dropped", B, U, threshold.Bstar, threshold.Ustar)
			return status.Errorf(codes.ResourceExhausted, "B or U value below the threshold, request dropped")
		}
		logger("[Ratelimiting] B %d and U %d values above the threshold B* %d and U* %d, request sent", B, U, threshold.Bstar, threshold.Ustar)
	} else {
		logger("[Ratelimiting] B* and U* values not found in the threshold table for method %s, request dropped", methodName[0])
		// return status.Errorf(codes.ResourceExhausted, "B* and U* values not found in the threshold table, request dropped")
	}
	// Modify ctx with the B and U
	ctx = metadata.AppendToOutgoingContext(ctx, "B", strconv.Itoa(B), "U", strconv.Itoa(U))

	// Invoking the gRPC call
	var header metadata.MD
	err := invoker(ctx, method, req, reply, cc, grpc.Header(&header))
	if err != nil {
		return err
	}

	// Store received B* and U* values from the header
	BstarValues := header.Get("B*")
	UstarValues := header.Get("U*")
	if len(BstarValues) > 0 && len(UstarValues) > 0 {
		Bstar, _ := strconv.Atoi(BstarValues[0])
		Ustar, _ := strconv.Atoi(UstarValues[0])
		d.thresholdTable.Store(methodName[0], thresholdVal{Bstar: Bstar, Ustar: Ustar})
		// d.thresholdTable[methodName[0]] = thresholdVal{Bstar: Bstar, Ustar: Ustar}
		logger("Received B* and U* values from the header: B*=%d, U*=%d", Bstar, Ustar)
	}

	return nil
}
