import SQLiteAsyncESMFactory from "wa-sqlite/dist/wa-sqlite-async.mjs";

import * as SQLite from "wa-sqlite";

import VirtualDatabase from "./db";

// Globals should probably be cleaned up into a class encapsulating state.
let DATABASE: VirtualDatabase;

interface requestArgs {
  path: string;
  body: Record<string, any>;
  reenroll?: boolean;
}
const request = async ({ path, body, reenroll = true }: requestArgs) => {
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

  if (response_body.node_invalid) {
    await clearNodeKey();
    if (reenroll) {
      try {
        await enroll();
      } catch (err) {
        throw new NodeInvalidError(`reenroll failed: ${err}`);
      }
      return await request({ path, body, reenroll: false });
    } else {
      throw new NodeInvalidError(response_body.error);
    }
  }
  if (!response.ok) {
    throw new Error(`${path} request failed: ${response_body.error}`);
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
const enroll = async () => {
  const os_version = await DATABASE.query("SELECT * FROM os_version");
  const system_info = await DATABASE.query("SELECT * FROM system_info");
  const host_details = {
    os_version: os_version[0],
    system_info: system_info[0],
  };

  const { enroll_secret } = await chrome.storage.managed.get({
    enroll_secret: "Y3nHXcZUBcFJc/7e6V6k1z7RG22rrAnQ", // + "bad",
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
  const response_body = await request({
    path: "/api/osquery/enroll",
    body: enroll_request,
    reenroll: false,
  });

  const { node_key } = response_body;
  if (node_key === "") {
    throw new Error("server returned empty node key without error");
  }
  await setNodeKey(node_key);
};

const live_query = async () => {
  const node_key = await getNodeKey();
  const live_query_request = { node_key };
  const response = await request({
    path: "/api/osquery/distributed/read",
    body: live_query_request,
  });

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
    node_key,
    queries: results,
    statuses,
    messages,
  };

  const result_response = await request({
    path: "/api/osquery/distributed/write",
    body: live_query_result_request,
  });
};

const getNodeKey = async () => {
  const { node_key } = await chrome.storage.local.get("node_key");
  return node_key;
};

const clearNodeKey = async () => {
  await chrome.storage.local.remove("node_key");
};

const setNodeKey = async (node_key: string) => {
  await chrome.storage.local.set({ node_key });
};

(async () => {
  const module = await SQLiteAsyncESMFactory();
  const sqlite3 = SQLite.Factory(module);
  const db = await sqlite3.open_v2(":memory:");

  const virtual = new VirtualDatabase(sqlite3, db);
  DATABASE = virtual;

  const node_key = await getNodeKey();
  if (!node_key) {
    await enroll();
  }

  //await sqlite3.close(db);
})();

class NodeInvalidError extends Error {
  constructor(message: string) {
    super(`request failed with node_invalid: ${message}`);
    this.name = "NodeInvalidError";
  }
}

// Run a live_query routine every 10s.
setInterval(live_query, 10 * 1000);
