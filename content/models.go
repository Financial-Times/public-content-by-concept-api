package content

type contentList []content

type content struct {
	ID     string `json:"id"`
	APIURL string `json:"apiUrl"`
}
