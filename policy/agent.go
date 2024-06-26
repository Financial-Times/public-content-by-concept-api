package policy

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/Financial-Times/go-logger/v2"
	transactionidutils "github.com/Financial-Times/transactionid-utils-go"
)

const (
	PublicationPolicyKey = "is_authorized_for_publication"
	OpaPolicyPath        = "public_content_by_concept/is_authorized_for_publication"
)

type Result struct {
	IsAuthorizedForPublication bool     `json:"is_authorized_for_publication"`
	AddFilterByPublication     bool     `json:"add_filter_by_publication"`
	Publications               []string `json:"xpolicy_publications"`
	Reasons                    []string `json:"reasons"`
}

func IsAuthorizedPublication(n http.Handler, w http.ResponseWriter, req *http.Request, log *logger.UPPLogger, r Result) {
	transID := transactionidutils.GetTransactionIDFromRequest(req)
	logEntry := log.WithTransactionID(transID)
	if r.IsAuthorizedForPublication {
		n.ServeHTTP(w, req)
	} else {
		if r.AddFilterByPublication {
			logEntry.Infof("Adding filter for publications: %s", r.Publications)
			req.URL.RawQuery = req.URL.RawQuery + fmt.Sprintf("&publication=%s", strings.Join(r.Publications, ","))
			n.ServeHTTP(w, req)
		} else {
			logEntry.Infof("Request is forbidden due to missing or non-matching access policies: %s", r.Reasons)
			http.Error(w, "Forbidden", http.StatusForbidden)
		}
	}
}
