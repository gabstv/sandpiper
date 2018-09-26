## Sandpiper
```
# 100 concurrent users under 1 minute
siege -c 100 -t 1m http://localhost:8080/test

Transactions:		       11814 hits
Availability:		      100.00 %
Elapsed time:		       59.93 secs
Data transferred:	        0.06 MB
Response time:		        0.00 secs
Transaction rate:	      197.13 trans/sec
Throughput:		        0.00 MB/sec
Concurrency:		        0.42
Successful transactions:       11814
Failed transactions:	           0
Longest transaction:	        0.04
Shortest transaction:	        0.00

# 100 concurrent users under 10 minutes
siege -c 100 -t 10m http://localhost:8080/test

Transactions:		      119206 hits
Availability:		      100.00 %
Elapsed time:		      599.77 secs
Data transferred:	        0.57 MB
Response time:		        0.00 secs
Transaction rate:	      198.75 trans/sec
Throughput:		        0.00 MB/sec
Concurrency:		        0.27
Successful transactions:      119206
Failed transactions:	           0
Longest transaction:	        0.04
Shortest transaction:	        0.00

```

## NGINX
```
# 100 concurrent users under 1 minute
siege -c 100 -t 1m http://localhost:8080/test

Transactions:		       11836 hits
Availability:		      100.00 %
Elapsed time:		       59.87 secs
Data transferred:	        0.06 MB
Response time:		        0.00 secs
Transaction rate:	      197.70 trans/sec
Throughput:		        0.00 MB/sec
Concurrency:		        0.35
Successful transactions:       11836
Failed transactions:	           0
Longest transaction:	        0.04
Shortest transaction:	        0.00

# 100 concurrent users under 10 minutes
siege -c 100 -t 10m http://localhost:8080/test

Transactions:		      118577 hits
Availability:		      100.00 %
Elapsed time:		      599.90 secs
Data transferred:	        0.57 MB
Response time:		        0.00 secs
Transaction rate:	      197.66 trans/sec
Throughput:		        0.00 MB/sec
Concurrency:		        0.70
Successful transactions:      118577
Failed transactions:	           0
Longest transaction:	        0.55
Shortest transaction:	        0.00

```

```bash
# start nginx
nginx -c ~/GO/src/github.com/gabstv/sandpiper/benchmarks/nginx.conf
# stop nginx
nginx -s quit
```