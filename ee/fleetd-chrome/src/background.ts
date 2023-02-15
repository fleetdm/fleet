// @ts-ignore
import SQLiteAsyncESMFactory from "wa-sqlite/dist/wa-sqlite-async.mjs";

import * as SQLite from "wa-sqlite";

import VirtualDatabase from "./db.js";

const enroll = async (host_details) => {
  const { fleet_url, enroll_secret } = await chrome.storage.managed.get({
    fleet_url: "https://fleet.loophole.site",
    enroll_secret: "Y3nHXcZUBcFJc/7e6V6k1z7RG22rrAnQ",
  });
  console.log("enrolling: ", fleet_url, enroll_secret);

  let host_identifier = host_details.system_info.hardware_serial;
  if (!host_identifier) {
    host_identifier = host_details.system_info.uuid;
  }

  const enroll_request = {
    enroll_secret,
    host_details,
    host_identifier,
  };
  const response = await fetch(new URL("/api/osquery/enroll", fleet_url), {
    method: "POST",
    body: JSON.stringify(enroll_request),
  });
  const response_body = await response.json();

  if (!response.ok) {
    throw new Error("enroll failed: " + response_body.error);
  }
  if (response_body.node_invalid) {
    throw new Error("enroll failed with node_invalid: " + response_body.error);
  }

  return response_body.node_key;
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
    console.log("got node key: ", node_key);
  } catch (err) {
    console.error("enroll failed: " + err);
  }

  await sqlite3.close(db);
})();
