package dagor

import (
	"context"
	"math/rand"
	"runtime/metrics"
	"strconv"
	"sync/atomic"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

var currentThresholdB int = 0 // Initialize with default value
var currentThresholdU int = 0 // Initialize with default value
var currentThresholdBVal interface{}
var currentThresholdUVal interface{}

func (d *Dagor) UnaryInterceptorServer(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	md, _ := metadata.FromIncomingContext(ctx)
	methodNames, methodExists := md["method"]
	// Ensure method name is present
	if !methodExists || len(methodNames) == 0 {
		return nil, status.Errorf(codes.InvalidArgument, "Method name not provided in metadata")
	}
	userIDs, userIDExists := md["user-id"]
	var B, U int
	var err error

	// if this is an entry service, B and U are not in metadata
	if d.entryService {
		methodName := methodNames[0]
		if businessValue, exists := d.businessMap[methodName]; exists {
			B = businessValue
		} else {
			return nil, status.Errorf(codes.Internal, "Business value for method %s not found", methodName)
		}
		if userIDExists && len(userIDs) > 0 {
			userID := userIDs[0]
			if val, ok := d.userPriority.Load(userID); ok {
				U = val.(int)
			} else {
				U = rand.Intn(100) // Assign a random int for U
				d.userPriority.Store(userID, U)
			}
		}
		logger("[Entry service] assigned B: %d, U: %d", d.nodeName, B, U)
	} else {
		BValues, BExists := md["B"]
		UValues, UExists := md["U"]

		methodName := methodNames[0]
		// if no B or U in metadata, this is an entry service
		if !BExists || !UExists {
			// mark this node as entry service
			d.entryService = true
			logger("Node %s is assigned as an entry service", d.nodeName)
		}

		// Assign B based on method from businessMap or metadata
		if !BExists || len(BValues) == 0 {
			if businessValue, exists := d.businessMap[methodName]; exists {
				B = businessValue
			} else {
				return nil, status.Errorf(codes.Internal, "Business value for method %s not found", methodName)
			}
			logger("B value not provided in metadata, assigned B: %d", B)
		} else {
			B, err = strconv.Atoi(BValues[0])
			if err != nil {
				return nil, status.Errorf(codes.InvalidArgument, "Invalid B value: %v", BValues[0])
			}
			logger("B value provided in metadata: %d", B)
		}

		// Assign U based on user-id from userPriority or metadata
		if !UExists || len(UValues) == 0 {
			if userIDExists && len(userIDs) > 0 {
				userID := userIDs[0]
				if val, ok := d.userPriority.Load(userID); ok {
					U = val.(int)
				} else {
					U = rand.Intn(100) // Assign a random int for U
					d.userPriority.Store(userID, U)
				}
			} else {
				return nil, status.Errorf(codes.InvalidArgument, "User ID not provided in metadata")
			}
			logger("U value not provided in metadata, assigned U: %d", U)
		} else {
			U, err = strconv.Atoi(UValues[0])
			if err != nil {
				return nil, status.Errorf(codes.InvalidArgument, "Invalid U value: %v", UValues[0])
			}
			logger("U value provided in metadata: %d", U)
		}
	}
	// Retrieve current thresholds from admissionLevel
	currentThresholdBVal, _ := d.admissionLevel.Load("B")
	currentThresholdUVal, _ := d.admissionLevel.Load("U")
	currentThresholdB := currentThresholdBVal.(int) // Assert the type to int
	currentThresholdU := currentThresholdUVal.(int) // Assert the type to int

	// If the request's B and U don't meet the threshold, drop the request
	if B < currentThresholdB || U < currentThresholdU {
		logger("Request B, U %d, %d values are below the threshold %d, %d", B, U, currentThresholdB, currentThresholdU)
		return nil, status.Errorf(codes.ResourceExhausted, "Request B, U values are below the threshold")
	}

	// Modify ctx with the B and U
	ctx = metadata.AppendToOutgoingContext(ctx, "B", strconv.Itoa(B), "U", strconv.Itoa(U))

	// Handle the request
	resp, err := handler(ctx, req)
	if err != nil {
		return nil, err
	}

	// Attach B* and U* to the response metadata
	newMD := metadata.Pairs("B*", strconv.Itoa(currentThresholdB), "U*", strconv.Itoa(currentThresholdU))
	logger("Attached B*, U* to the response metadata: B*=%d, U*=%d", currentThresholdB, currentThresholdU)
	grpc.SendHeader(ctx, newMD)

	return resp, nil
}

// func (d *Dagor) UnaryServerInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
// 	md, _ := metadata.FromIncomingContext(ctx)
// 	bValues := md.Get("B")
// 	uValues := md.Get("U")

// 	// Assuming single values for B and U
// 	if len(bValues) > 0 && len(uValues) > 0 {
// 		b, _ := strconv.Atoi(bValues[0])
// 		u, _ := strconv.Atoi(uValues[0])

// 		// If the request's B and U don't meet the threshold, drop the request
// 		if b < currentThresholdB || u < currentThresholdU {
// 			return nil, grpc.Errorf(codes.ResourceExhausted, "Request B, U values are below the threshold")
// 		}
// 	}

// 	resp, err := handler(ctx, req)

// 	// Attach B* and U* to the response metadata
// 	newMD := metadata.Pairs("B*", strconv.Itoa(currentThresholdB), "U*", strconv.Itoa(currentThresholdU))
// 	grpc.SendHeader(ctx, newMD)

// 	return resp, err
// }

// overloadDetection is a function that detects overload and updates the threshold
func (d *Dagor) UpdateAdmissionLevel() {
	var prevHist *metrics.Float64Histogram
	for range time.Tick(d.admissionLevelUpdateInterval) {
		// get the current histogram
		currHist := readHistogram()

		if prevHist == nil {
			// directly go to next iteration
			prevHist = currHist
			continue
		}
		gapLatency := maximumQueuingDelayms(prevHist, currHist)

		// Load the current threshold values for B and U
		currentThresholdBVal, _ := d.admissionLevel.Load("B")
		currentThresholdUVal, _ := d.admissionLevel.Load("U")
		currentThresholdB := currentThresholdBVal.(int)
		currentThresholdU := currentThresholdUVal.(int)

		// update the threshold
		foverload := gapLatency > float64(d.queuingThresh.Milliseconds())
		Bstar, Ustar := d.CalculateAdmissionLevel(foverload)

		// Update the admission level with the new values
		d.admissionLevel.Store("B", Bstar)
		d.admissionLevel.Store("U", Ustar)

		logger("Updated threshold B, U: %d, %d", currentThresholdB, currentThresholdU)

		// Update prevHist for the next iteration
		prevHist = currHist
	}
}

// Assuming the constants alpha, beta, Bmax, Umax, and the initial N are defined elsewhere
// and the sync.Map C is a part of the Dagor struct initialized appropriately

func (d *Dagor) ResetHistogram() {
	// Reset the N to 0
	d.UpdateN(0)
	// Reset the C matrix which holds the admitted request counters
	d.C.Range(func(key, value interface{}) bool {
		d.C.Store(key, 0)
		return true
	})
}

func (d *Dagor) UpdateHistogram(r int) {
	// Update the C matrix with the new histogram value
	// The implementation details of this will depend on how you're tracking requests
}

// CalculateAdmissionLevel adjusts the B and U based on the overload flag
func (d *Dagor) CalculateAdmissionLevel(foverload bool) (int, int) {
	Nexp := atomic.LoadInt64(&d.N)

	// Adjust Nexp based on overload
	if foverload {
		Nexp = int64((1 - d.alpha) * float64(Nexp))
	} else {
		Nexp = int64((1 + d.beta) * float64(Nexp))
	}

	Bstar, Ustar := 0, 0
	// Nprefix int64
	Nprefix := int64(0)

	// Iterate over the range of B and U values
	for B := 1; B <= d.Bmax; B++ {
		for U := 1; U <= d.Umax; U++ {
			// Retrieve the count for this B, U combination from the C matrix
			val, _ := d.C.Load([2]int{B, U})
			Nprefix += val.(int64)

			if Nprefix > Nexp {

				return Bstar, Ustar
			}
			Bstar, Ustar = B, U
		}
	}
	// If the loop completes without returning, update the admission level with the max values
	return Bstar, Ustar
}
