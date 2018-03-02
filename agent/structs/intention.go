package structs

import (
	"time"
)

const (
	// IntentionWildcard is the wildcard value.
	IntentionWildcard = "*"
)

// Intention defines an intention for the Connect Service Graph. This defines
// the allowed or denied behavior of a connection between two services using
// Connect.
type Intention struct {
	// ID is the UUID-based ID for the intention, always generated by Consul.
	ID string

	// SourceNS, SourceName are the namespace and name, respectively, of
	// the source service. Either of these may be the wildcard "*", but only
	// the full value can be a wildcard. Partial wildcards are not allowed.
	// The source may also be a non-Consul service, as specified by SourceType.
	//
	// DestinationNS, DestinationName is the same, but for the destination
	// service. The same rules apply. The destination is always a Consul
	// service.
	SourceNS, SourceName           string
	DestinationNS, DestinationName string

	// SourceType is the type of the value for the source.
	SourceType IntentionSourceType

	// Action is whether this is a whitelist or blacklist intention.
	Action IntentionAction

	// DefaultAddr, DefaultPort of the local listening proxy (if any) to
	// make this connection.
	DefaultAddr string
	DefaultPort int

	// Meta is arbitrary metadata associated with the intention. This is
	// opaque to Consul but is served in API responses.
	Meta map[string]string

	// CreatedAt and UpdatedAt keep track of when this record was created
	// or modified.
	CreatedAt, UpdatedAt time.Time `mapstructure:"-"`

	RaftIndex
}

// IntentionAction is the action that the intention represents. This
// can be "allow" or "deny" to whitelist or blacklist intentions.
type IntentionAction string

const (
	IntentionActionAllow IntentionAction = "allow"
	IntentionActionDeny  IntentionAction = "deny"
)

// IntentionSourceType is the type of the source within an intention.
type IntentionSourceType string

const (
	// IntentionSourceConsul is a service within the Consul catalog.
	IntentionSourceConsul IntentionSourceType = "consul"
)

// Intentions is a list of intentions.
type Intentions []*Intention

// IndexedIntentions represents a list of intentions for RPC responses.
type IndexedIntentions struct {
	Intentions Intentions
	QueryMeta
}

// IndexedIntentionMatches represents the list of matches for a match query.
type IndexedIntentionMatches struct {
	Matches []Intentions
	QueryMeta
}

// IntentionOp is the operation for a request related to intentions.
type IntentionOp string

const (
	IntentionOpCreate IntentionOp = "create"
	IntentionOpUpdate IntentionOp = "update"
	IntentionOpDelete IntentionOp = "delete"
)

// IntentionRequest is used to create, update, and delete intentions.
type IntentionRequest struct {
	// Datacenter is the target for this request.
	Datacenter string

	// Op is the type of operation being requested.
	Op IntentionOp

	// Intention is the intention.
	Intention *Intention

	// WriteRequest is a common struct containing ACL tokens and other
	// write-related common elements for requests.
	WriteRequest
}

// RequestDatacenter returns the datacenter for a given request.
func (q *IntentionRequest) RequestDatacenter() string {
	return q.Datacenter
}

// IntentionMatchType is the target for a match request. For example,
// matching by source will look for all intentions that match the given
// source value.
type IntentionMatchType string

const (
	IntentionMatchSource      IntentionMatchType = "source"
	IntentionMatchDestination IntentionMatchType = "destination"
)

// IntentionQueryRequest is used to query intentions.
type IntentionQueryRequest struct {
	// Datacenter is the target this request is intended for.
	Datacenter string

	// IntentionID is the ID of a specific intention.
	IntentionID string

	// Match is non-nil if we're performing a match query. A match will
	// find intentions that "match" the given parameters. A match includes
	// resolving wildcards.
	Match *IntentionQueryMatch

	// Options for queries
	QueryOptions
}

// RequestDatacenter returns the datacenter for a given request.
func (q *IntentionQueryRequest) RequestDatacenter() string {
	return q.Datacenter
}

// IntentionQueryMatch are the parameters for performing a match request
// against the state store.
type IntentionQueryMatch struct {
	Type    IntentionMatchType
	Entries []IntentionMatchEntry
}

// IntentionMatchEntry is a single entry for matching an intention.
type IntentionMatchEntry struct {
	Namespace string
	Name      string
}

// IntentionPrecedenceSorter takes a list of intentions and sorts them
// based on the match precedence rules for intentions. The intentions
// closer to the head of the list have higher precedence. i.e. index 0 has
// the highest precedence.
type IntentionPrecedenceSorter Intentions

func (s IntentionPrecedenceSorter) Len() int { return len(s) }
func (s IntentionPrecedenceSorter) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func (s IntentionPrecedenceSorter) Less(i, j int) bool {
	a, b := s[i], s[j]

	// First test the # of exact values in destination, since precedence
	// is destination-oriented.
	aExact := s.countExact(a.DestinationNS, a.DestinationName)
	bExact := s.countExact(b.DestinationNS, b.DestinationName)
	if aExact != bExact {
		return aExact > bExact
	}

	// Next test the # of exact values in source
	aExact = s.countExact(a.SourceNS, a.SourceName)
	bExact = s.countExact(b.SourceNS, b.SourceName)
	return aExact > bExact
}

// countExact counts the number of exact values (not wildcards) in
// the given namespace and name.
func (s IntentionPrecedenceSorter) countExact(ns, n string) int {
	// If NS is wildcard, it must be zero since wildcards only follow exact
	if ns == IntentionWildcard {
		return 0
	}

	// Same reasoning as above, a wildcard can only follow an exact value
	// and an exact value cannot follow a wildcard, so if name is a wildcard
	// we must have exactly one.
	if n == IntentionWildcard {
		return 1
	}

	return 2
}
