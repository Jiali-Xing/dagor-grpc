package dagor

import (
	"math/rand"
	"sync"
	"time"
)

var debug = true

// Dagor is the DAGOR network.
type Dagor struct {
	nodeName       string
	uuid           string         // Only if nodeName is "Client"
	businessMap    map[string]int // Maps methodName to int B
	queuingThresh  time.Duration  // Overload control in milliseconds
	userPriority   sync.Map       // Concurrent map from user to priority
	thresholdTable sync.Map       // Concurrent map to keep B* and U* values for each downstream, key is method name
	// userPriority   map[string]int          // Map from user to priority
	// thresholdTable map[string]thresholdVal // Map to keep B* and U* values for each downstream, key is method name
	entryService                 bool     // Entry service for the DAGOR network
	isEnduser                    bool     // Is this node an end user?
	admissionLevel               sync.Map // Map to keep track of admission level B and U
	admissionLevelUpdateInterval time.Duration
	alpha                        float64
	beta                         float64
	Umax                         int
	Bmax                         int
	C                            sync.Map
	N                            int64 // Use int64 to be compatible with atomic operations
	// C is a two-dimensional array or a map that corresponds to the counters for each B, U pair.
	// You need to initialize this with the actual data structure you are using.

}

type thresholdVal struct {
	Bstar int
	Ustar int
}

type DagorParam struct {
	NodeName                     string
	UUID                         string
	BusinessMap                  map[string]int
	QueuingThresh                time.Duration
	EntryService                 bool
	IsEnduser                    bool
	AdmissionLevelUpdateInterval time.Duration
	Alpha                        float64
	Beta                         float64
	Umax                         int
	Bmax                         int
}

// NewDagorNode creates a new DAGOR node without a UUID.
func NewDagorNode(params *DagorParam) *Dagor {
	dagor := Dagor{
		nodeName:                     params.NodeName,
		uuid:                         params.UUID,
		businessMap:                  params.BusinessMap,
		queuingThresh:                params.QueuingThresh,
		userPriority:                 sync.Map{}, // Initialize as empty concurrent map
		thresholdTable:               sync.Map{}, // Initialize as empty concurrent map
		entryService:                 params.EntryService,
		isEnduser:                    params.IsEnduser,
		admissionLevel:               sync.Map{}, // Initialize as empty concurrent map
		admissionLevelUpdateInterval: params.AdmissionLevelUpdateInterval,
		alpha:                        params.Alpha,
		beta:                         params.Beta,
		Umax:                         params.Umax,
		Bmax:                         params.Bmax,
		// C:                            params.C,
	}
	dagor.admissionLevel.Store("B", 0)
	dagor.admissionLevel.Store("U", 0)
	rand.Seed(time.Now().UnixNano())

	// Initialize the C matrix with the initial counters for each B, U pair
	for B := 1; B <= params.Bmax; B++ {
		for U := 1; U <= params.Umax; U++ {
			dagor.C.Store([2]int{B, U}, 0)
		}
	}
	return &dagor
}