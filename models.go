package main

type ContentList []Content

type Content struct {
	ID     string `json:"id"`
	APIURL string `json:"apiUrl"`
}
