package main

import (
    "fmt"
    "github.com/gocql/gocql"
    _ "net/url"
    "log"
    "time"
    "math/rand"
    "os"
    "strconv"
    "unicode/utf8"
)

var cluster *gocql.ClusterConfig
var session *gocql.Session
var legitChars string = "abcdefghijklmnopqrstuvwxyz1234567890-"
var legitCharCount = utf8.RuneCountInString(legitChars)

//there's probably an easier and more idiomatic way to do this in go...
type Info struct {
	Hostname string
	Rest string
}

//create host from int seed
func generateHostname(seed int) (name string) {
  name = "www.site"

  for seed > 0 {
  	char := seed % legitCharCount
  	name += string(legitChars[char])
  	seed /= legitCharCount
  }

  name += ".com"

  return
}

//generate a bunch of simple variants exercising query/path functionality
func generateUrls(startAt int, length int) ([]Info) {
	var data = make([]Info, length);

	for i := 0; i<length/5; i++ {
		hostname := generateHostname(startAt + i)
		data[i*5] = Info{ Hostname: hostname, Rest: "path" }
		data[i*5+1] = Info{ Hostname: hostname, Rest: "?a=b" }
		data[i*5+2] = Info{ Hostname: hostname, Rest: "path?a=b" }
		data[i*5+3] = Info{ Hostname: hostname, Rest: "complex/path" }
		data[i*5+4] = Info{ Hostname: hostname, Rest: "complex/path?complex=yes&query=yes" }
	}
	return data
}

func main() {

	//init db
	cluster = gocql.NewCluster(os.GetEnv("CASSANDRA_IP"))
	cluster.Keyspace = "url-safety"
	session, _ = cluster.CreateSession()

	//init rand
	rand.Seed(time.Now().UnixNano());

	//clean the table
	err := session.Query("TRUNCATE urls").Exec()
	if(err != nil) {
		log.Fatal("Truncate: ", err)
	}

	//generate fake data
	count := 4000000;
	chunk := 100;
	statement := "INSERT INTO urls(hostname, rest, safe, updated) VALUES(?, ?, ?, ?)"

	for i := 0; i<count/chunk; i++ {
		if(i * chunk % 100000 == 0) {
			fmt.Println(i * chunk)
		}
		batch := gocql.NewBatch(gocql.LoggedBatch)
		data := generateUrls(i * chunk, chunk);

		for j:= 0; j<chunk; j++ {
			//fmt.Println(data[j])
			batch.Query(statement, data[j].Hostname, data[j].Rest, rand.Float64() > 0.5, time.Now())
		}
		err := session.ExecuteBatch(batch)
		if(err != nil) {
			log.Fatal("ExecuteBatch: ", err)
		}
	}

	fmt.Println("Generated " + strconv.Itoa(count) + " urls")

    //clean up session
    defer session.Close()
}
