Using `curl` and `jq` to interact with the fleet API.

First, create a `env` file with the following contents:

```
export SERVER_URL=https://localhost:8080 # your fleet server url and port
export CURL_FLAGS='-k -s' # set insecure flag
export TOKEN=eyJhbGciOi... # your api token
```

Next set the `FLEET_ENV_PATH` to point to the `env` file. This will let the scripts in the `fleet/` folder source the env file.

# Examples

```
export FLEET_ENV_PATH=./path/to/env/file/fleet_env

# get my user info
./tools/api/fleet/me
{
  "user": {
    "created_at": "2018-04-10T02:07:46Z",
    "updated_at": "2018-04-10T02:07:46Z",
    "id": 1,
    "name": "admin",
    "email": "admin@acme.co",
    "admin": true,
    "enabled": true,
    "force_password_reset": false,
    "gravatar_url": "",
    "sso_enabled": false
  }
}

# list queries
./tools/api/fleet/queries/list
{
  "queries": []
}

# use jq to filter a specific query and get the id
./tools/api/fleet/queries/list | jq '.queries[]|select(.name == "osquery_info")|.id'
2

# create a query
./tools/api/fleet/queries/create 'system_info' 'SELECT * FROM system_info;'
{
  "query": {
    "created_at": "0001-01-01T00:00:00Z",
    "updated_at": "0001-01-01T00:00:00Z",
    "id": 4,
    "name": "system_info",
    "description": "",
    "query": "SELECT * FROM system_info;",
    "saved": true,
    "author_id": 1,
    "author_name": "admin",
    "packs": []
  }
}

# add query with id=4 to pack with id=2
./tools/api/fleet/schedule/add_query_to_pack 2 4

# get scheduled queries in a pack
./tools/api/fleet/packs/scheduled 2 | jq '.scheduled[]|{"name": .name, "schedule_id": .id, "query_id": .query_id}'

# run a live queries on hosts (queries with id=1 and id=2 on hosts with id=3 and id=4)
./tools/api/fleet/queries/run "[1,2]" "[3,4]"
```

> The following examples are made obsolete with the vuln endpoint https://fleetdm.com/docs/rest-api/rest-api#vulnerabilities


Bash Script - Pulls all hosts based on software _name_ for your Fleet instance, uses jq. Helps if wanting to track down a particular software and see what hosts might be affected.

`./name.sh api_token software_title_id base_url`

```
#!/bin/bash

# Check if we have the correct number of arguments
if [ "$#" -ne 3 ]; then
    echo "Usage: $0 <api_token> <software_title_id> <base_url>"
    exit 1
fi

# Read arguments
API_TOKEN="$1"
SOFTWARE_TITLE_ID="$2"
BASE_URL="$3"

# Get the version IDs for the software title
VERSION_IDS=$(curl -s "${BASE_URL}/software/titles/${SOFTWARE_TITLE_ID}" \
  -H 'accept: application/json, text/plain, */*' \
  -H "authorization: Bearer ${API_TOKEN}" \
  --compressed | jq '.software_title.versions[].id')

# Define a jq filter for deduplicating hosts by id
jq_filter='[.[] | {id: .id, hostname: .hostname}] | unique_by(.id)'

# Make a temporary file to hold all host entries
tmpfile=$(mktemp)

# Loop through each version ID and get the hosts
for version_id in $VERSION_IDS; do
  # Fetch hosts for the current version ID
  curl -s "${BASE_URL}/hosts?software_version_id=${version_id}" \
    -H 'accept: application/json, text/plain, */*' \
    -H "authorization: Bearer ${API_TOKEN}" \
    --compressed | jq '.hosts[]' >> "$tmpfile"
done

# Deduplicate hosts by id and convert to a JSON array
jq -s "$jq_filter" "$tmpfile"

# Remove the temporary file
rm "$tmpfile"
```

Some quick Python to pull all Vuln software per host 
Might be better to do this _backwards_ by host instead of by the software. Attempting to use parallel threading to make it run faster, only helps a little.
can adjust `structure` to display what info you want.


```
import requests
import time
import json
from concurrent.futures import ThreadPoolExecutor

# Define the base URL for the API
BASE_URL = "https://dogfood.fleetdm.com/api/latest" #change to your base url

# The headers for the HTTP requests, including the Authorization Bearer token
HEADERS = {
    'Authorization': 'Bearer TOKEN',  # Add your API token
    'Content-Type': 'application/json'
}

# Initialize counters for API calls and hits
api_calls_counter = 0
hits_counter = 0

# Initialize a cache to store hosts for software versions
version_hosts_cache = {}


# Function to get all the software titles with vulnerabilities
def get_all_vulnerable_software_titles():
    global api_calls_counter
    endpoint = (f"{BASE_URL}/fleet/software/titles?scope=software-titles&page=0&per_page=1000&order_direction=desc&order_key=hosts_count&vulnerable=true") #note that this is set to 1k to try and capture "all" but might need to adjust
    response = requests.get(endpoint, headers=HEADERS)
    api_calls_counter += 1

    if response.status_code == 200:
        data = response.json()
        return data.get('software_titles', [])
    else:
        raise Exception(f"Error fetching software titles: {response.text}")


# Function to get the hosts for a software version with caching
def get_hosts_for_software_version(version_id):
    global api_calls_counter
    global hits_counter

    # Check the cache first
    if version_id in version_hosts_cache:
        return version_hosts_cache[version_id]

    # If not cached, make the request
    endpoint = f"{BASE_URL}/fleet/hosts?software_version_id={version_id}"
    response = requests.get(endpoint, headers=HEADERS)
    api_calls_counter += 1

    if response.status_code == 200:
        hosts = response.json().get('hosts', [])
        hits_counter += len(hosts)
        # Cache the result
        version_hosts_cache[version_id] = hosts
        return hosts
    else:
        raise Exception(f"Error fetching hosts for software version {version_id}: {response.text}")


# Function to fetch hosts for all vulnerable software versions in parallel using threading
def fetch_hosts_in_parallel(software_versions):
    with ThreadPoolExecutor(max_workers=10) as executor:
        future_to_version_id = {executor.submit(get_hosts_for_software_version, v['id']): v['id'] for v in
                                software_versions}
        for future in future_to_version_id:
            future.result()  # We wait for each call to complete here. The results are stored in the cache.


# Main function to build the desired structure
def build_structure():
    global api_calls_counter
    global hits_counter

    api_calls_counter = 0
    hits_counter = 0

    software_titles = get_all_vulnerable_software_titles()
    vulnerable_versions = [version for software in software_titles for version in software.get('versions', []) if
                           version.get('vulnerabilities')]
    fetch_hosts_in_parallel(vulnerable_versions)  # Fetch all hosts in parallel

    structure = {}
    for software in software_titles:
        for version in software.get('versions', []):
            if version.get('vulnerabilities'):
                version_id = version['id']
                hosts = version_hosts_cache.get(version_id, [])
                for host in hosts:
                    host_id = host['id']
                    if host_id not in structure:
                        structure[host_id] = {
                            "hostname": host['hostname'],
                            "software": []
                        }
                    structure[host_id]['software'].append({
                        "version_id": str(version_id),
                        "software_id": str(software['id']),
                        "name": software['name']
                    })

   
    return structure


# Run the main function and print results
if __name__ == "__main__":
    start_time = time.time()
    final_structure = build_structure()
    print(json.dumps(final_structure, indent=2))
    #print(f"Total time taken: {time.time() - start_time:.2f} seconds")
    #print(f"API Calls: {api_calls_counter}")
    #print(f"total number of software vulns: {hits_counter}")
```
