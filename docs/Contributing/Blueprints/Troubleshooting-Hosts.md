# Fleet "Host Troubleshooting"

## Osquery worker/watcher

TODO/WIP: Write some docs around how osquery executes queries and the watcher/worker processes.

## Fleet's *detection* of `distributed/write` issues

Expected/Happy-path scenario:

1. Osquery host requests queries via the `distributed/read` request.
2. Fleet returns queries to execute (query set depends on configured intervals for details, policies, labels).
3. Osquery host executes the queries sequentially.
4. Osquery host sends the results of the queries via the `distributed/write` request.

We've seen the following scenario (A) on some hosts/deployments:

1. Osquery host requests queries via the `distributed/read` request.
2. Fleet returns queries to execute (query set depends on configured intervals for details, policies, labels).
3. Osquery start executing the queries sequentially.
4. Some query exceeds a performance threshold and the worker process is killed by the watcher process (e.g. memory or CPU intensive query).
5. Osquery restarts and performs step (1) over and over.

From the Fleet UI, the host is shown as "online" (because of the successful `distributed/read`
request in step 1 and also due to periodic `config` requests), but the "Last fetched" shown in the Hosts table (corresponding to
"detail_updated_at") is the time it last executed a successful `distributed/write`.

We would like Fleet to:
1. *detect* such problematic queries and show them to the user/admin.
2. *automatic troubleshooting* to allow the other non-problematic queries to eventually run on
the host and shed some light on the problematic queries.

We've also seen the following scenario (B) on some hosts/deployments:

1. Osquery host requests queries via `distributed/read`.
2. Fleet returns queries to execute (query set depends on configured intervals for details, policies, labels).
3. Osquery host executes the queries sequentially.
4. Osquery fails to send the `distributed/write` request, e.g. because:
    - proxies rejecting requests due to body size (which could happen if a lot of results are returned).
    - temporary network errors (`distributed/read` succeeded, but `distributed/write` failed because of network partition).

One way to detect these two scenarios (A) and (B) would be to check the following when executing the "distributed/read" request:

```go
// detectHostNotResponding returns whether the host hasn't been submitting results for sent queries.
func (svc *Service) detectHostNotResponding(host *fleet.Host) bool {
	interval := svc.config.Osquery.DetailUpdateInterval
	if host.DistributedInterval > interval {
		interval = host.DistributedInterval
	}
	// The following means Fleet hasn't received the distributed/request from the host
	// "in a while", thus we assume the host is having some issue in executing the
	// queries or sending the results.
	return svc.clock.Now() > (host.DetailUpdatedAt + 2 * interval)
} 
```

Notes: 
- `Service.detectHostNotResponding` will allow us to detect such hosts (TBD: Where to store and show such info).
- We use `2 * interval`, because of the artificial jitter added to the intervals in Fleet.
- Default value for:
    - host.DistributedInterval is usually 10s.
    - svc.config.Osquery.DetailUpdateInterval is usually 1h.

## Automatic Troubleshooting

Once Fleet detects that a host is not responding the `distributed/read` with its corresponding
`distributed/write`, we need a way to reduce the impact of the sent queries on such host.
For instance, if a single query is the problematic one, then it doesn't make sense for all the other queries to miss execution on the host.

### Host Vitals and Detail Queries

What we call "Host Vitals" in Fleet could be split into two groups of queries:
- Queries that update the `hosts` table (aka detail queries): 
    - `network_interface`
    - `os_version`
    - `osquery_flags`
    - `osquery_info`
    - `system_info`
    - `uptime`
    - `disk_space_unix|disk_space_windows`
- Queries that update tables that link to `hosts`:
    - `mdm`
    - `munki_info`
    - `google_chrome_profiles`
    - `orbit_info`
    - `software_macos|software_linux|software_windows`
    - `users`
    - `scheduled_query_stats`
	
### Algorithm

- The algorithm will assume the detail queries (first group shown above) will never have any performance issues.
Therefore we use them as the proof that a `distributed/write` request succeeded (by reading and updating
`host.details_updated_at`).
- The algorithm runs in the `distributed/read` and `distributed/write` requests
(and we cannot assume that the server that processes the read request will be the same that
processes the write request).
- The algorithm sends a "trouble query" as a "request stamp" in the `distributed/read`.
This way Fleet can tie the queries sent in the `distributed/read` to the results received in the
`distributed/write`. (Such stamp contains the hash of the queries.)
- State storage for each host with issues:
	- To not generate more load we can define a new column in `hosts` called something like
	`trouble_queries`. Given that we load the host by selecting `hosts` on every request, this
	should cause no extra performance penalty on every request).
	- Alternatively, we could use Redis to store such state, but it would add a Redis read request
	on every `distributed/read` request (NOTE: we are already doing a Redis request on the
	`distributed/read`, for retrieving live queries).
- We need to use and store the hashes of the queries because queries can change and new queries can be
added in between read and write requests (e.g. policy queries are editable by users and new ones
can be assigned to hosts).
- The algorithm performs a sort of binary search of the problematic query. TODO(lucas): Determine
how to support more than one query being problematic. With algorithm shown below, Fleet will ping pong between two
halfs if both have a problematic query.
- The added performance penalty of the algorithm is in `distributed/read`, there's now an extra
write to `hosts` table (only on hosts with issues).

Following is the rough/untested pseudo-code for the `distributed/read` and `distributed/write` to
implement the binary search. The algorithm keeps a JSON marshalled `TroubleQueriesState` for each problematic host in the `hosts` table.
	
#### `distributed/read`

```go
type TroubleQueriesState struct {
	BatchIndex int
	BatchSize  int
	// Queries holds the suspicious queries (currently in search for culprits).
	Queries    []QueryWithHash
	// Denylisted holds the queries that have been confirmed to cause issues in the host.
	Denylisted []QueryWithHash
}

type QueryWithHash struct {
	Query string
	Hash  string
}

func determineQueriesToSend(host *fleet.Host, queriesToSend map[string]string) map[string]string {
	hostNotResponding := svc.detectHostNotResponding(host)

	// TODO(lucas): Check host.TroubleQueries.Denylisted, and not include these in the response.

	switch {

	case !hostNotResponding && host.TroubleQueries == "":

		//
		// Nothing to do, host is responding and there are no "trouble queries".
		//

		return queriesToSend

	case hostNotResponding && host.TroubleQueries == "":

		//
		// This is the first time Fleet detects the host has issues.
		//
		
		// We assume hostDetailQueries never have performance issues.
		nonDetailQueries := excludeQueries(queriesToSend, hostDetailQueries)
		
		queriesWithHash := withHash(nonDetailQueries)
		host.TroubleQueries = TroubleQueriesState{
			BatchIndex: 0,
			BatchSize: len(queriesToSend)/2,
			Queries: queriesWithHash,
		}
		svc.ds.UpdateHost(ctx, host)

		queriesToSend := excludeQueries(queriesToSend, host.TroubleQueries.Queries)
		troubleQueries := nonDetailQueries[0:len(queriesToSend)/2]
		addQueries(queriesToSend, troubleQueries) // send first half of trouble queries
		
		// generateTroubleQuery generates a string of the form: SELECT "q0:hash(q0),q1:hash(q1),..."
		// which serves as an indicator of the sent queries on distributed/write.
		queriesToSend["trouble_queries"] = generateTroubleQuery(queriesWithHash[0:len(queriesWithHash/2)])
		return queriesToSend

	case !hostNotResponding && host.TroubleQueries != "":

		//
		// This means the host is in "troubleshooting mode" but it has responded to some queries.
		//
	
		queriesToSend := excludeQueries(queriesToSend, host.TroubleQueries.Queries)
		troubleQueries := queriesToSend[
			host.TroubleQueries.BatchSize*host.TroubleQueries.BatchIndex:
			host.TroubleQueries.BatchSize*host.TroubleQueries.BatchIndex+1,
		]
		addQueries(queriesToSend, troubleQueries)
		
		// generateTroubleQuery generates a string of the form: SELECT "q0:hash(q0),q1:hash(q1),..."
		// which serves as an indicator of the sent queries on distributed/write.
		queriesToSend["trouble_queries"] = generateTroubleQuery(troubleQueries)
		return queriesToSend

	default: // hostNotResponding && host.TroubleQueries != "":
		
		//
		// This means the host is in "troubleshooting mode" and hasn't responded to first half of trouble queries.
		//
		if host.TroubleQueries.BatchIndex == 0 && host.TroubleQueries.BatchSize == 0 {
			// We've found a query that kills the host, thus we move it from host.TroubleQueries.Queries to host.TroubleQueries.Denylisted
			foundOneTroubleQuery(host)
			host.TroubleQueries.BatchSize = len(host.TroubleQueries.Queries)/2
		} else {
			host.TroubleQueries.BatchIndex = 0
			host.TroubleQueries.BatchSize -= 1
		}
		svc.ds.UpdateHost(ctx, host)
		
		queriesToSend := excludeQueries(queriesToSend, host.TroubleQueries)
		troubleQueries := host.TroubleQueries[len(host.TroubleQueries)/2: len(host.TroubleQueries)] // send second half of trouble queries
		addQueries(queriesToSend, troubleQueries)

		// generateTroubleQuery generates a string of the form: SELECT "q0:hash(q0),q1:hash(q1),..."
		// which serves as an indicator of the sent queries on distributed/write.
		queriesToSend["trouble_queries"] = generateTroubleQuery(troubleQueries)
		return queriesToSend
	}
}
```

#### `distributed/write`
	
```go
func updateTroubleHost(host *fleet.Host, results fleet.OsqueryDistributedQueryResults) bool {
	ok, troubleQueryResults := results["trouble_queries"]
	if !ok {
		return false // nothing to do
	}
	// Remove queries that are confirmed to be working and update batch index and size.
	host.TroubleQueries = host.TroubleQueries[(host.TroubleQueries.BatchIndex+1)*host.TroubleQueries.BatchSize:]
	host.TroubleQueries.BatchIndex = 0
	host.TroubleQueries.BatchSize /= 2
}

//
// In SubmitDistributedQueryResults, we must make sure to update host in the database if updateTroubleHost returns true.
//
```


## Osquery future improvements

TODO/WIP: Distributed queries retries suggested by Zach.
Currently considering this as the proper solution given the complexity added to Fleet in the above algorithm.

## Misc Findings / Questions

### Need for Lighter Queries?

- Multiple smaller queries vs giant software query. Makes sense if a worker can be killed because of
  a long running query or a memory/CPU intensive query. If so, software queries split into multiple queries can reduce the chance of error.