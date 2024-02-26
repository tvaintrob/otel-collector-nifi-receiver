package translator

// ProvenanceEvent is a struct that represents a single provenance event
type ProvenanceEvent struct {
	EventId             string              `json:"eventId,omitempty"`
	EventOrdinal        int64               `json:"eventOrdinal,omitempty"`
	EventType           ProvenanceEventType `json:"eventType,omitempty"`
	TimestampMillis     int64               `json:"timestampMillis,omitempty"`
	DurationMillis      int64               `json:"durationMillis,omitempty"`
	LineageStart        int64               `json:"lineageStart,omitempty"`
	Details             string              `json:"details,omitempty"`
	ComponentId         string              `json:"componentId,omitempty"`
	ComponentType       string              `json:"componentType,omitempty"`
	ComponentName       string              `json:"componentName,omitempty"`
	ProcessGroupId      string              `json:"processGroupId,omitempty"`
	ProcessGroupName    string              `json:"processGroupName,omitempty"`
	EntityId            string              `json:"entityId,omitempty"`
	EntityType          string              `json:"entityType,omitempty"`
	EntitySize          int64               `json:"entitySize,omitempty"`
	PreviousEntitySize  int64               `json:"previousEntitySize,omitempty"`
	UpdatedAttributes   map[string]string   `json:"updatedAttributes,omitempty"`
	PreviousAttributes  map[string]string   `json:"previousAttributes,omitempty"`
	ActorHostname       string              `json:"actorHostname,omitempty"`
	ContentURI          string              `json:"contentURI,omitempty"`
	PreviousContentURI  string              `json:"previousContentURI,omitempty"`
	ParentIds           []string            `json:"parentIds,omitempty"`
	ChildIds            []string            `json:"childIds,omitempty"`
	Platform            string              `json:"platform,omitempty"`
	Application         string              `json:"application,omitempty"`
	RemoteIdentifier    string              `json:"remoteIdentifier,omitempty"`
	AlternateIdentifier string              `json:"alternateIdentifier,omitempty"`
	TransitUri          string              `json:"transitUri,omitempty"`
}

// ProvenanceEventType is a type that represents the type of a provenance event
type ProvenanceEventType string

const (
	ProvenanceEventTypeAddInfo            ProvenanceEventType = "ADDINFO"
	ProvenanceEventTypeAttributesModified ProvenanceEventType = "ATTRIBUTES_MODIFIED"
	ProvenanceEventTypeClone              ProvenanceEventType = "CLONE"
	ProvenanceEventTypeContentModified    ProvenanceEventType = "CONTENT_MODIFIED"
	ProvenanceEventTypeCreate             ProvenanceEventType = "CREATE"
	ProvenanceEventTypeDownload           ProvenanceEventType = "DOWNLOAD"
	ProvenanceEventTypeDrop               ProvenanceEventType = "DROP"
	ProvenanceEventTypeExpire             ProvenanceEventType = "EXPIRE"
	ProvenanceEventTypeFetch              ProvenanceEventType = "FETCH"
	ProvenanceEventTypeFork               ProvenanceEventType = "FORK"
	ProvenanceEventTypeJoin               ProvenanceEventType = "JOIN"
	ProvenanceEventTypeReceive            ProvenanceEventType = "RECEIVE"
	ProvenanceEventTypeRemoteInvocation   ProvenanceEventType = "REMOTE_INVOCATION"
	ProvenanceEventTypeReplay             ProvenanceEventType = "REPLAY"
	ProvenanceEventTypeRoute              ProvenanceEventType = "ROUTE"
	ProvenanceEventTypeSend               ProvenanceEventType = "SEND"
	ProvenanceEventTypeUnknown            ProvenanceEventType = "UNKNOWN"
)
