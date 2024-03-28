package policy

import (
	"fmt"
	"github.com/Financial-Times/go-logger/v2"
	"net/http"
	"strings"
)

const (
	PublicationPolicyKey = "is_authorized_for_publication"
	OpaPolicyPath        = "public_content_by_concept/is_authorized_for_publication"
)

type Result struct {
	IsAuthorizedForPublication bool     `json:"is_authorized_for_publication"`
	AddFilterByPublication     bool     `json:"add_filter_by_publication"`
	Publications               []string `json:"xpolicy_publications"`
}

func IsAuthorizedPublication(n http.Handler, w http.ResponseWriter, req *http.Request, log *logger.UPPLogger, r Result) {

	if r.IsAuthorizedForPublication {
		n.ServeHTTP(w, req)
	} else {
		if r.AddFilterByPublication {
			log.Infof("Adding filter for publications: %s", r.Publications)
			req.URL.RawQuery = req.URL.RawQuery + fmt.Sprintf("&publication=%s", strings.Join(r.Publications, ","))
			n.ServeHTTP(w, req)
		} else {

			http.Error(w, "Forbidden", http.StatusForbidden)
		}
	}
}
