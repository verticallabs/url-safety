# url safety app

NB: still learning golang, so I'm sure some of this could be more idiomatic 

## USAGE (assumes working go env)

1. Fire up cassandra using `docker-compose`
2. Create a database

```
docker exec -it cassandra_1 /usr/bin/cqlsh
create table urls(hostname ascii, rest ascii, safe boolean, updated timestamp, PRIMARY KEY(hostname, query));	
```

3. Seed the db (currently set to 4m records)

```
go run seed.go
```

4. Run the server

```
go run server.go
```

5. Curl server 

```
time curl "localhost:3000/urlinfo/1/www.siteJ.com?a=b"
{"safe":true}
real	0m0.025s
user	0m0.004s
sys	0m0.007s
```

## Benchmarking

Using https://github.com/cmpxchg16/gobench

```
$ ~/go/bin/gobench -u http://localhost:3000/urlinfo/1/www.siteJ.com?a=b -k=true -c 50 -t 5
Dispatching 50 clients
Waiting for results...

Requests:                            35052 hits
Successful requests:                 35052 hits
Network failed:                          0 hits
Bad requests failed (!2xx):              0 hits
Successful requests rate:             7010 hits/sec
Read throughput:                    911976 bytes/sec
Write throughput:                   793847 bytes/sec
Test time:                               5 sec
```

# NOTES BELOW
-----
 
Write a small web service, in the language/framework your choice,
that responds to GET requests where the caller passes in a URL and
the service responds with some information about that URL. The GET
requests look like this:
 
        GET /urlinfo/1/{hostname_and_port}/{original_path_and_query_string}
 
The caller wants to know if it is safe to access that URL or not.
As the implementer you get to choose the response format and
structure. These lookups are blocking users from accessing the URL
until the caller receives a response from your service.
 
Give some thought to the following:
 
  * The size of the URL list could grow infinitely, how might you
  scale this beyond the memory capacity of this VM? Bonus if you
  implement this.
 
  * The number of requests may exceed the capacity of this VM, how
  might you solve that? Bonus if you implement this.
 
  * What are some strategies you might use to update the service
  with new URLs? Updates may be as much as 5 thousand URLs a day
  with updates arriving every 10 minutes.


## overview

- handle web request
- check cache in case we've already crawled it
- if yes, respond to link to safety assessment resource (with completed status) and respond to request
- if no, return link to assessment resource which currently has incomplete status
- crawl site (run safety assessment algorithm)
- when assessment is completed, update assessment resource

## storage/caching

- persistence required
- very read-heavy
- SQL, NoSQL, K/V stores might all be plausibly used, at extreme scale something like Cassandra might be a good choice
- at scale it would need sharding 
- hostname is an obvious sharding key, or potentially hostname + path, but really any N letters of the target URL should work if the sharding gets extreme.
- also need a request cache and good caching strategy

## thoughts/questions around design

- known: unbounded url storage (well, bounded by number of existing urls and length constraints potentially)
- known: 5000 new urls per day at 10 min intervals
- there is potentially some optimization and interesting decisions around matching specific urls without query strings, ie if url with query string X is not safe, likely the same url with query string Y is not safe either.
- is it safe to assume that we're only assessing the safety of the URL for GET requests? ie no other verbs
- what are the length limits on the target URL if any?  
- how many requests expected per day (maximum)?
- how long is the longest we can reasonably block clients by not giving them results before the service is useless? 
- what is the target response time? obviously < 1ms is ideal, but what is an acceptable target time?
- how complicated is the algorithm to decide whether the site is safe? 
- how long does the assessment take? is it fast enough to plausibly return results fast enough for the service to respond? 
- how long can we hold client request open for?
- how do we deal with sites that are slow to respond? can we cut them off and call them unsafe after X seconds?
- overview is currently based on the assumption that the assessment is slow enough that you don't want to hold the request open in the case of requests for previously uncrawled sites.  this may be false.
- how long do we cache results for?  there is always a possibility of the site being changed immediately after the crawl, so there is always! a possibility of false positives.
- the caching/storage strategy might be complicated if requests are clustered around specific sharding keys.  
- it looks like currently the return value from this query is going to be a boolean: safe/not safe.  what is the likelihood of expanding the service at a later date to a more complex response, either with more gradations of safety (uncertain/probably safe, etc), more details about the site (meta information on the crawl time?).  if no chance of this, the storage strategy could be essentially two lists.
- trie data structure seems like a good match for this as well, so you could maximize speed/storage space - just storing a safe/unsafe indicator at the end, if you want to go with a custom storage implementation.
- ultimately you could improve this service at scale by using a bot to crawl and index like the google search engine, but that seems a little out of scope :P

- thought: implementing this as a proxy rather than a safe/unsafe assessment service would save clients of the service an extra request in the case that responses are extremely time critical, but obviously that balloons the storage requirements

## thoughts/questions around implementation

- json seems like an obvious choice unless we have a specific userbase that wants SOAP/XML or some other format
- can i assume this the actual assessment is a near-instant black box function run against the GET results from the URL? if not, what are the assessment specs?
- VM was mentioned in the question but not expanded on.  is there a particular container/VM format that is expected as part of the output of this question? or are you just looking for a functional web service?

## response questions

### Also quite likely.  What other URL normalization might you need to do?
- convert host to lowercase
- capitalize escape sequences
- sort query parameters
- remove duplicate query params

### Any thoughts on how to constrain the persistence size of the URLs?
- Hmm, you could think about hashing them, but you'd have to deal with collisions somehow, and the hashed value wouldn't be useful for analytics, etc.  If you needed to go this way, you could look at storing multiple hashes to resolve collisions (at the expense of speed, obviously) and store the url in some other bigger, slower, eventually consistent data storage for analytics. 
- Using trie would cut down storage requirements significantly
- In a relational db, you could split the hostnames into a separate table and avoid duplication there at the expense of retrieval speed
- You could use a compression algorithm to compress the url before storing
- There are only ~80-90 allowable chars in a url, so you could use short chars

To store all the URLs, assuming a maximum of 2k chars for URLs (URL length is theoretically unbounded, but I would expect 2k to catch 99.99% of urls - more analysis would be required if you need to support unbounded lengths)

20 million URLs
20m * 2k (url chars can all fit in ASCII)
40 000 000 000 bytes
40 TB without using any other compression strategies  


### What other tradeoffs would you me making here? (besides space if the service was a proxy
- clients would potentially be getting out of date responses
- this also makes the retrieving the page a requirement, which is not necessarily the case if your assessment strategy is just to test the host against a list of hostnames
- it would increase the load on your http servers significantly since the amount of data they would have to push would increase by a factor of ~100-10000.

### Cache invalidation
- ideally you'd expire the cache on a regular basis depending on the update frequency of the target page
- in the real world i wouldn't expect there to be too many sites that go back and forth between safe and unsafe and it might not be a bad idea to penalize unsafe sites anyway, so I think you could probably use large cache expiry times like 24hrs or potentially longer.

### Response time targets?
Re: https://www.nngroup.com/articles/response-times-3-important-limits/
The service should aim for < 100ms, ideally sub 50ms, but obviously the faster the better.

In the real world, real-time retrieval and analysis time for the target page is essentially only bounded by timeouts, so you'd never implement that (you'd be using a pre-crawled DB), which means you'd probably want a third designation of "unknown", although you could argue that unknown => unsafe (or safe depending on how good your data is).  Since we're not going to do any real-time retrieval, I'm going to drop the concept of a "progress" resource.
