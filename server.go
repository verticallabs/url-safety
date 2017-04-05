package main

import (
    "fmt"
    "github.com/gocql/gocql"
    "log"
    "net/http"
    "net/url"
    "strings"
    "time"
    "os"
    //"net/http/httputil"
)

var prefix string = "/urlinfo/1/"
var cluster *gocql.ClusterConfig
var session *gocql.Session

// func dumpRequest(w http.ResponseWriter, r *http.Request) {
// 	req, err := httputil.DumpRequest(r, true)
//     if err != nil {
//         log.Fatal("DumpRequest: ", err)
//     }

//     fmt.Printf("%s", req)
//     fmt.Fprintf(w, "%s", req)
// }


func isUrlSafe(h string, r string) bool {
	var hostname, rest string
	var safe bool
	var updated time.Time

	if err := session.Query("SELECT * FROM urls WHERE hostname = ? and rest = ?", h, r).Scan(&hostname, &rest, &safe, &updated); err != nil {
		return false
	}

	return safe
}

func normalizeRest(path string, query string) string {
	values, _ := url.ParseQuery(query)
	normalizedQuery := values.Encode()
	return path + "?" + normalizedQuery
}

func normalizeHostname(h string) string {
	return strings.ToLower(h)
}

//breaking url up into "hostname" and "rest"
func extractHostnameAndRest(targetURL string) (hostname string, rest string) {
	firstSlash := strings.Index(targetURL, "/")
	question := strings.Index(targetURL, "?")

	var path, query string
	if(firstSlash > -1) {
		hostname = targetURL[0:firstSlash]

		if(question > -1) {
			path = targetURL[firstSlash+1:question]
			query = targetURL[question+1:]
		} else {
			path = targetURL[firstSlash+1:]
			query = ""
		}
	} else if (question > -1) { //no path
		hostname = targetURL[0:question]
		path = ""
		query = targetURL[question+1:]
	} else { //no path, no query
		hostname = targetURL
		path = ""
		query = ""
	}

	//fmt.Printf("%s %s %s\n", hostname, path, query)
	hostname = normalizeHostname(hostname)
	rest = normalizeRest(path, query)

	return
}

func renderResponse(w http.ResponseWriter, safe bool) {
	json := "{\"safe\":"
	if(safe) {
		json += "true}"
	} else {
		json += "false}"
	}

    fmt.Fprintf(w, "%s", json)
}

func requestHandler(w http.ResponseWriter, r *http.Request) {
	targetURL := r.URL.String()[len(prefix):]
	hostname, rest := extractHostnameAndRest(targetURL)

	//fmt.Printf("%s %s\n", hostname, rest)
	safe := isUrlSafe(hostname, rest)

	renderResponse(w, safe);
}

func main() {

	//init db
	cluster = gocql.NewCluster(os.Getenv('CASSANDRA_IP'))
	cluster.Keyspace = "url-safety"
	session, _ = cluster.CreateSession()

	//setup http handler
    http.HandleFunc(prefix, requestHandler)
    err := http.ListenAndServe(":8080", nil)
    if err != nil {
        log.Fatal("ListenAndServe: ", err)
    }

    //clean up session
    defer session.Close()
}
