import SQLiteAsyncESMFactory from "wa-sqlite/dist/wa-sqlite-async.mjs";

import * as SQLite from "wa-sqlite";

import VirtualDatabase from "./db";

let NODE_KEY = "";

const request = async (path: string, body: Record<string, any>) => {
  const { fleet_url } = await chrome.storage.managed.get({
    fleet_url: "https://fleet.loophole.site",
  });

  const response = await fetch(new URL(path, fleet_url), {
    method: "POST",
    body: JSON.stringify(body),
  });
  const response_body = await response.json();

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

  console.log(response);
};

(async () => {
  const module = await SQLiteAsyncESMFactory();
  const sqlite3 = SQLite.Factory(module);
  const db = await sqlite3.open_v2(":memory:");

  const virtual = new VirtualDatabase(sqlite3, db);

  const os_version = await virtual.query("SELECT * FROM os_version");
  const system_info = await virtual.query("SELECT * FROM system_info");

  try {
    const node_key = await enroll({
      os_version: os_version[0],
      system_info: system_info[0],
    });
    NODE_KEY = node_key;
    console.log("got node key: ", node_key);
  } catch (err) {
    console.error("enroll failed: " + err);
  }

  await sqlite3.close(db);
})();

setInterval(live_query, 10 * 1000);
