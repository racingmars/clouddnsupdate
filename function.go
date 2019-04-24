package clouddnsupdate

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"strings"

	"google.golang.org/api/dns/v1"
)

var noChangeNeeded = errors.New("Hostname already set to requested IP address")

var validIPAddressRegex = regexp.MustCompile(`^(([0-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5])\.){3}([0-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5])$`)
var validHostnameRegex = regexp.MustCompile(`^(([a-zA-Z0-9]|[a-zA-Z0-9][a-zA-Z0-9\-]*[a-zA-Z0-9])\.)*([A-Za-z0-9]|[A-Za-z0-9][A-Za-z0-9\-]*[A-Za-z0-9])$`)

// Update is a cloud function that allows DynDNS-like functionality
func Update(w http.ResponseWriter, r *http.Request) {
	expectedUser := os.Getenv("MRW_USERNAME")
	expectedPass := os.Getenv("MRW_PASSWORD")
	project := os.Getenv("MRW_PROJECT")
	zone := os.Getenv("MRW_ZONE")
	allowedDomain := os.Getenv("MRW_DOMAIN")
	if expectedUser == "" || expectedPass == "" || project == "" || zone == "" || allowedDomain == "" {
		fmt.Println("Not configured. Set MRW_USERNAME, MRW_PASSWORD, MRW_PROJECT, MRW_ZONE, MRW_DOMAIN")
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "911")
		return
	}

	user, pw, ok := r.BasicAuth()
	if !ok {
		fmt.Println("Received unauthenticated request.")
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprintf(w, "badauth")
		return
	}

	if strings.ToLower(user) != strings.ToLower(expectedUser) ||
		pw != expectedPass {
		fmt.Println("Bad username/password.")
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprintf(w, "badauth")
		return
	}

	hostname := r.FormValue("hostname")
	myip := r.FormValue("myip")

	if !validHostnameRegex.MatchString(hostname) {
		fmt.Println("Hostname didn't match regex:", hostname)
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "notfqdn")
		return
	}

	if !validIPAddressRegex.MatchString(myip) {
		fmt.Println("IP didn't match regex:", myip)
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "notip")
		return
	}

	if !strings.HasSuffix(hostname, allowedDomain) {
		fmt.Println("Hostname isn't in allowed domain")
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "nohost")
		return
	}

	hostname = hostname + "."
	err := updateDNS(project, zone, hostname, myip)
	if err == noChangeNeeded {
		fmt.Printf("No need to change %s to %s\n", hostname, myip)
		fmt.Fprintf(w, "nochg")
		return
	} else if err != nil {
		fmt.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "911")
		return
	}

	fmt.Fprintf(w, "good")

	fmt.Printf("Successfully updated %s to %s\n", hostname, myip)
}

func updateDNS(project, zone, hostname, ip string) error {
	ctx := context.Background()
	dnsService, err := dns.NewService(ctx)
	if err != nil {
		return err
	}

	listcall := dnsService.ResourceRecordSets.List(project, zone)
	listresp, err := listcall.Do()
	if err != nil {
		return err
	}
	var foundRec *dns.ResourceRecordSet
	for i := range listresp.Rrsets {
		if listresp.Rrsets[i].Name == hostname {
			foundRec = listresp.Rrsets[i]
			break
		}
	}

	if foundRec != nil {
		if foundRec.Rrdatas[0] == ip {
			return noChangeNeeded
		}
	}

	change := &dns.Change{}
	rrs := &dns.ResourceRecordSet{}
	rrs.Name = hostname
	rrs.Rrdatas = []string{ip}
	rrs.Ttl = 60
	rrs.Type = "A"
	if foundRec != nil {
		change.Deletions = append(change.Deletions, foundRec)
	}
	change.Additions = append(change.Additions, rrs)
	changeCall := dnsService.Changes.Create(project, zone, change)
	_, err = changeCall.Do()
	if err != nil {
		return err
	}
	return nil
}
