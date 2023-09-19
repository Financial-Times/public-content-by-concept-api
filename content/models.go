package content

const (
	ThingsPrefix = "http://www.ft.com/things/"
)

type Content struct {
	ID                      string `json:"id"`
	APIURL                  string `json:"apiUrl"`
	ContentPublication      string `json:"contentPublication,omitempty"`
	RelationshipPublication string `json:"relationshipPublication,omitempty"`
	ConceptPublication      string `json:"conceptPublication,omitempty"`
}
