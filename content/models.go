package content

const (
	ThingsPrefix = "http://www.ft.com/things/"
)

type Content struct {
	ID     string `json:"id"`
	APIURL string `json:"apiUrl"`
}
