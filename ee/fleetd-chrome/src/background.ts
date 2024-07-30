import VirtualDatabase from "./db";
import {E_ALREADY_LOCKED, Mutex, tryAcquire, withTimeout} from 'async-mutex';

// ENV Vars
declare var FLEET_URL: string;
declare var FLEET_ENROLL_SECRET: string;

// Memory leaks have been observed in WASM sqlite implementation due to thrown errors by Javascript,
// which may be caused by improper SQL syntax or insufficient error handling in table implementations.
// Handling this error at the top level will prevent the extension from becoming non-functional.
const MEMORY_RUNTIME_ERROR_MESSAGE = "memory access out of bounds";

// TODO: Globals should probably be cleaned up into a class encapsulating state.
let DATABASE: VirtualDatabase;

interface requestArgs {
  path: string;
  body?: Record<string, any>;
  reenroll?: boolean;
}

const request = async ({ path, body = {} }: requestArgs): Promise<any> => {
  const { fleet_url } = await chrome.storage.managed.get({
    fleet_url: FLEET_URL,
  });

  const target = new URL(path, fleet_url);

  const options = {
    method: "POST",
    body: JSON.stringify(body),
  };
  console.debug("Request:", target, options);
  let response: Response;
  let response_body: { node_invalid: any; error: string };
  try {
    response = await fetch(target, options);
    response_body = await response.json();
  } catch (err) {
    if (response && !response.ok) {
      let text = ""
      try {
        text = await response.text()
      } catch (e) {
        // ignore, since we already know response is not ok
      }
      throw new Error(`Failed to fetch ${target}: ${response.status} ${response.statusText} ${text}`);
    }
    console.warn(`Failed to fetch ${target}: ${err}`);
    throw new Error(`${path} request failed`);
  }
  console.debug("Response:", response, "JSON:", response_body);

  if (response_body.node_invalid) {
    // QUESTION: Is it acceptable design for us to be modifying the storage state in this function?
    // Should the only side effect be the network request?
    await clearNodeKey();
    throw new NodeInvalidError(response_body.error);
  }
  if (!response.ok) {
    throw new Error(`${path} request failed: ${response_body.error}`);
  }

  return response_body;
};

const authenticatedRequest = async ({
  path,
  body = {},
  reenroll = true,
}: requestArgs): Promise<any> => {
  const node_key = await getNodeKey();
  if (!node_key) {
    console.warn(`node key empty in ${path} request`);
  }

  try {
    return await request({path, body: {...body, node_key}});
  } catch (err) {
    // Reenroll if it's a node_invalid issue (and we haven't already tried a reenroll), otherwise
    // rethrow the error.
    if (err instanceof NodeInvalidError && reenroll) {
      await enroll();
      // Prevent infinite recursion by disabling reenroll on the retry.
      return await authenticatedRequest({ path, body, reenroll: false });
    }
    throw err;
  }
};

const enroll = async () => {
  const os_version = (await DATABASE.query("SELECT * FROM os_version")).data;
  const system_info = (await DATABASE.query("SELECT * FROM system_info")).data;
  const host_details = {
    os_version: os_version[0],
    system_info: system_info[0],
  };

  const { enroll_secret } = await chrome.storage.managed.get({
    enroll_secret: FLEET_ENROLL_SECRET,
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
    path: "/api/v1/osquery/enroll",
    body: enroll_request,
  });

  const { node_key } = response_body;
  if (node_key === "") {
    throw new Error("server returned empty node key without error");
  }
  await setNodeKey(node_key);
};

const live_query = async () => {
  const response = await authenticatedRequest({
    path: "/api/v1/osquery/distributed/read",
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
        const discovery_result = (await DATABASE.query(query_discovery_sql))
          .data;
        if (discovery_result.length == 0) {
          // Discovery queries that return no results mean skip running the query.
          continue;
        }
      } catch (err) {
        if (err.message === MEMORY_RUNTIME_ERROR_MESSAGE) {
          // We completely cancel the live query if wa-sqlite hit a memory error to prevent returning bogus data.
          throw err
        }
        // Discovery queries failing is typical -- they are often used to "discover" whether the
        // tables exist.
        console.debug(
          `Discovery (${query_name} sql: "${query_discovery_sql}") failed: ${err}`
        );
        results[query_name] = null;
        statuses[query_name] = 1;
        messages[query_name] = err.toString();
        continue;
      }
    }

    // Run the actual query if discovery passed.
    const query_sql = response.queries[query_name];
    try {
      const query_result = await DATABASE.query(query_sql);
      results[query_name] = query_result.data;
      statuses[query_name] = 0;
      if (query_result.warnings && query_result.warnings.length !== 0) {
        statuses[query_name] = 1; // Set to show warnings in errors table and campaign.ts returned host_counts to +1 failing instead of +1 successful
        messages[query_name] = query_result.warnings; // Warnings array is concatenated in Table.ts xfilter
      }
    } catch (err) {
      if (err.message === MEMORY_RUNTIME_ERROR_MESSAGE) {
        // We completely cancel the live query if wa-sqlite hit a memory error to prevent returning bogus data.
        throw err
      }
      console.warn(`Query (${query_name} sql: "${query_sql}") failed: ${err}`);
      results[query_name] = null;
      statuses[query_name] = 1;
      messages[query_name] = err.toString();
    }
  }

  const live_query_result_request = {
    queries: results,
    statuses,
    messages,
  };

  await authenticatedRequest({
    path: "/api/v1/osquery/distributed/write",
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

const main = async () => {
  console.debug("main");

  // @ts-expect-error @types/chrome doesn't yet have navigator.userAgentData.
  const platform = navigator.userAgentData.platform;
  const { installType } = await chrome.management.getSelf();
  if (platform !== "Chrome OS" && installType !== "development") {
    console.error("Refusing to run on non Chrome OS with managed install!");
    return;
  }

  if (!DATABASE) {
    await initDB();
  }

  const node_key = await getNodeKey();
  if (!node_key) {
    await enroll();
  }
  await live_query();
  //await sqlite3.close(db);
};

const initDB = async () => {
  DATABASE = await VirtualDatabase.init();
  globalThis.DB = DATABASE;
}

class NodeInvalidError extends Error {
  constructor(message: string) {
    super(`request failed with node_invalid: ${message}`);
    this.name = "NodeInvalidError";
  }
}

// We use a mutex to ensure that only one instance of main is running at a time.
const mutexWithTimeout = withTimeout(new Mutex(), 60 * 1000) // 60 second timeout
async function runExclusive(callback: () => Promise<void>) {
  await tryAcquire(mutexWithTimeout).runExclusive(callback)
}

// QUESTION maybe we should use one of the persistence mechanisms described in
// https://stackoverflow.com/a/66618269/491710? The "offscreen API" mechanism might be useful. On
// the other hand, this seems to work decently well and adding the complexity might not be worth it.

// This is a bit funky here. We want the main loop to run every 10 seconds, but we have to be
// careful that we clear the old timeouts because of the alarm triggering that causes an additional
// call to mainLoop. If we don't clear the timeout, we'll start getting more and more calls to
// mainLoop each time the alarm fires.
let mainTimeout: ReturnType<typeof setTimeout>;
const mainLoop = async () => {
  try {
    await runExclusive(main);
  } catch (err) {
    if (err === E_ALREADY_LOCKED) {
      console.info("'main' mutex already locked, skipping run")
      return
    }
    console.error(err);
    if (err.message === MEMORY_RUNTIME_ERROR_MESSAGE) {
      console.info("Restarting DB after wa-sqlite RuntimeError")
      await initDB();
    }
  }
  clearTimeout(mainTimeout);
  mainTimeout = setTimeout(mainLoop, 10 * 1000);
};
mainLoop();

// This alarm is used to ensure the extension "wakes up" at least once every minute. Otherwise
// Chrome could shut it down in the background.
const MAIN_ALARM = "main";
chrome.alarms.create(MAIN_ALARM, { periodInMinutes: 1 });
chrome.alarms.onAlarm.addListener(async ({ name }) => {
  console.debug(`alarm ${name} $fired`);
  switch (name) {
    case MAIN_ALARM:
      await mainLoop();
      break;
    default:
      console.error(`unknown alarm ${name}`);
  }
});
