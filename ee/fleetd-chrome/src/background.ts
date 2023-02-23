import SQLiteAsyncESMFactory from "wa-sqlite/dist/wa-sqlite-async.mjs";

import * as SQLite from "wa-sqlite";

import VirtualDatabase from "./db";

// Globals should probably be cleaned up into a class encapsulating state.
let NODE_KEY = "";
let DATABASE: VirtualDatabase;

const request = async (path: string, body: Record<string, any>) => {
  const { fleet_url } = await chrome.storage.managed.get({
    fleet_url: "https://fleet.loophole.site",
  });

  const target = new URL(path, fleet_url);
  const options = {
    method: "POST",
    body: JSON.stringify(body),
  };
  console.debug("Request:", target, options);
  const response = await fetch(target, options);
  const response_body = await response.json();
  console.debug("Response:", response, "JSON:", response_body);

  if (!response.ok) {
    throw new Error("request failed: " + response_body.error);
  }
  if (response_body.node_invalid) {
    throw new Error("request failed with node_invalid: " + response_body.error);
  }

  return response_body;
};

interface SystemInfo {
  hardware_serial: string;
  uuid: string;
}
interface EnrollDetails {
  system_info?: SystemInfo;
  os_version?: Map<string, any>;
}
const enroll = async (host_details: EnrollDetails) => {
  const { enroll_secret } = await chrome.storage.managed.get({
    enroll_secret: "Y3nHXcZUBcFJc/7e6V6k1z7RG22rrAnQ",
  });

  let host_identifier = host_details.system_info.hardware_serial;
  if (!host_identifier) {
    host_identifier = host_details.system_info.uuid;
  }

  const enroll_request = {
    enroll_secret,
    host_details,
    host_identifier,
  };
  const response_body = await request("/api/osquery/enroll", enroll_request);

  return response_body.node_key;
};

const live_query = async () => {
  const live_query_request = {
    node_key: NODE_KEY,
  };
  const response = await request(
    "/api/osquery/distributed/read",
    live_query_request
  );

  if (!response.queries || Object.keys(response.queries).length === 0) {
    // No queries were returned by the server. Nothing to do.
    return;
  }

  const results = {};
  const statuses = {};
  const messages = {};
  for (const query_name in response.queries) {
    // Run the discovery query to see if we should run the actual query.
    const query_discovery_sql = response.discovery[query_name];
    if (query_discovery_sql) {
      try {
        const discovery_result = await DATABASE.query(query_discovery_sql);
        if (discovery_result.length == 0) {
          // Discovery queries that return no results mean skip running the query.
          continue;
        }
      } catch (err) {
        // Discovery queries failing is typical -- they are often used to "discover" whether the
        // tables exist.
        console.debug(
          `Discovery (${query_name} sql: "${query_discovery_sql}") failed: ${err}`
        );
        continue;
      }
    }

    // Run the actual query if discovery passed.
    try {
      const query_result = await DATABASE.query(response.queries[query_name]);
      results[query_name] = query_result;
      statuses[query_name] = 0;
    } catch (err) {
      console.warn(
        `Query (${query_name} sql: "${query_discovery_sql}") failed: ${err}`
      );
      results[query_name] = null;
      statuses[query_name] = 1;
      messages[query_name] = err.toString();
    }
  }

  const live_query_result_request = {
    node_key: NODE_KEY,
    queries: results,
    statuses,
    messages,
  };

  const result_response = await request(
    "/api/osquery/distributed/write",
    live_query_result_request
  );
};

(async () => {
  const module = await SQLiteAsyncESMFactory();
  const sqlite3 = SQLite.Factory(module);
  const db = await sqlite3.open_v2(":memory:");

  const virtual = new VirtualDatabase(sqlite3, db);
  DATABASE = virtual;

  const os_version = await virtual.query("SELECT * FROM os_version");
  const system_info = await virtual.query("SELECT * FROM system_info");

  try {
    console.log("enrolling");
    const node_key = await enroll({
      os_version: os_version[0],
      system_info: system_info[0],
    });
    NODE_KEY = node_key;
    console.log("got node key: ", node_key);
  } catch (err) {
    console.error("enroll failed: " + err);
  }

  //await sqlite3.close(db);
})();

// Run a live_query routine every 10s.
setInterval(live_query, 10 * 1000);
