import { Client } from "pg";
import { from as copyFrom } from "pg-copy-streams";
import { Readable } from "stream";
import { pipeline } from "stream/promises";

const DB_TABLE = "usage_data";

const INTEGER_COLUMNS = new Set([
  "numHostsEnrolled",
  "numUsers",
  "numTeams",
  "numQueries",
  "numPolicies",
  "numLabels",
  "numWeeklyActiveUsers",
  "numHostsNotResponding",
  "numHostsABMPending",
  "numSoftwareVersions",
  "numHostSoftwares",
  "numSoftwareTitles",
  "numHostSoftwareInstalledPaths",
  "numSoftwareCPEs",
  "numSoftwareCVEs",
  "numHostsFleetDesktopEnabled",
]);

const JSON_COLUMNS = new Set([
  "hostsEnrolledByOperatingSystem",
  "hostsEnrolledByOrbitVersion",
  "hostsEnrolledByOsqueryVersion",
]);

function colTypeFor(col: string): string {
  if (col === "updatedTime") return "TIMESTAMP";
  if (INTEGER_COLUMNS.has(col)) return "INTEGER";
  if (JSON_COLUMNS.has(col)) return "JSONB";
  if (
    col.endsWith("Enabled") ||
    col.endsWith("Disabled") ||
    col.endsWith("Configured")
  )
    return "BOOLEAN";
  return "TEXT";
}

function csvEscape(field: string): string {
  if (field.includes(",") || field.includes('"') || field.includes("\n")) {
    return '"' + field.replace(/"/g, '""') + '"';
  }
  return field;
}

interface DataclipJSON {
  fields: string[];
  values: unknown[][];
}

async function loadData(): Promise<DataclipJSON> {
  const dataclipUrl = process.env.DATACLIP_URL;
  if (!dataclipUrl) {
    throw new Error("DATACLIP_URL environment variable is required");
  }
  const herokuToken = process.env.HEROKU_API_TOKEN;
  if (!herokuToken) {
    throw new Error("HEROKU_API_TOKEN environment variable is required");
  }
  console.log(`Fetching data from dataclip URL...`);
  const res = await fetch(dataclipUrl, {
    headers: {
      Accept: "application/json",
      Authorization: `Bearer ${herokuToken}`,
    },
  });
  if (!res.ok) {
    throw new Error(`Dataclip fetch failed: ${res.status} ${res.statusText}`);
  }
  return (await res.json()) as DataclipJSON;
}

async function main() {
  // pg Client reads PGHOST, PGPORT, PGDATABASE, PGUSER, PGPASSWORD automatically

  // --- 1. Load JSON data ---
  let data: DataclipJSON;
  try {
    data = await loadData();
  } catch (err) {
    console.error(`Error loading data: ${err}`);
    process.exit(1);
  }

  const header = data.fields;
  const dataRows = data.values;
  console.log(`Loaded ${dataRows.length} rows with ${header.length} columns.`);

  // --- 2. Cleanse data and build CSV ---
  console.log("Cleansing data...");

  const timeIndex = header.indexOf("updatedTime");
  const integerIndices = new Map<string, number>();
  for (const col of INTEGER_COLUMNS) {
    const idx = header.indexOf(col);
    if (idx !== -1) integerIndices.set(col, idx);
  }
  const jsonIndices = new Map<string, number>();
  for (const col of JSON_COLUMNS) {
    const idx = header.indexOf(col);
    if (idx !== -1) jsonIndices.set(col, idx);
  }

  const csvLines: string[] = [header.map(csvEscape).join(",")];

  for (const row of dataRows) {
    if (row.length !== header.length) {
      console.log(`Warning: Skipping malformed row with ${row.length} columns.`);
      continue;
    }

    const csvRow: string[] = row.map((val, i) => {
      if (val === null || val === undefined) return "";

      // Clean datetime field
      if (i === timeIndex && typeof val === "string" && val.includes("+00")) {
        return val.replace("+00", "");
      }

      // Clean integer fields (floats like 14.0 -> 14)
      if (integerIndices.has(header[i])) {
        const num = Number(val);
        if (!isNaN(num)) return Math.trunc(num).toString();
        console.log(`Warning: Could not convert '${val}' to int for column '${header[i]}'.`);
        return String(val);
      }

      // Serialize JSON fields
      if (jsonIndices.has(header[i]) && typeof val === "object") {
        return JSON.stringify(val);
      }

      return String(val);
    });

    csvLines.push(csvRow.map(csvEscape).join(","));
  }

  const cleanedCsv = csvLines.join("\n") + "\n";
  console.log("Cleansing complete.");

  // --- 3. Import into PostgreSQL ---
  const client = new Client();
  try {
    console.log("Connecting to database...");
    await client.connect();
    console.log("Connection successful.");

    // Ensure table exists
    console.log(`Ensuring table '${DB_TABLE}' exists...`);
    const ddlColumns = header
      .map((col) => `"${col}" ${colTypeFor(col)}`)
      .join(", ");
    await client.query(
      `CREATE TABLE IF NOT EXISTS ${DB_TABLE} (${ddlColumns});`
    );

    // Add any new columns that don't yet exist in the table
    const { rows } = await client.query(
      "SELECT column_name FROM information_schema.columns WHERE table_name = $1",
      [DB_TABLE]
    );
    const existingColumns = new Set(
      rows.map((r: { column_name: string }) => r.column_name)
    );
    for (const col of header) {
      if (!existingColumns.has(col)) {
        const colType = colTypeFor(col);
        console.log(
          `Adding new column '${col}' (${colType}) to table '${DB_TABLE}'...`
        );
        await client.query(
          `ALTER TABLE ${DB_TABLE} ADD COLUMN "${col}" ${colType};`
        );
      }
    }

    // Clear existing data before import to avoid duplicates across runs
    await client.query(`TRUNCATE ${DB_TABLE};`);

    // Bulk import via COPY
    console.log(`Importing data into table '${DB_TABLE}'...`);
    const quotedHeader = header.map((col) => `"${col}"`).join(",");
    const copyStream = client.query(
      copyFrom(
        `COPY ${DB_TABLE} (${quotedHeader}) FROM STDIN WITH (FORMAT CSV, HEADER)`
      )
    );
    await pipeline(Readable.from([cleanedCsv]), copyStream);
    console.log(`${copyStream.rowCount} rows imported successfully.`);

    // Create customers view used by Grafana dashboards.
    // Organizations with multiple environments get labeled by host count:
    // the largest is "(Prod)", a single smaller one is "(Stage)", and
    // additional ones are numbered "(Stage 2)", "(Stage 3)", etc.
    await client.query(`
      CREATE OR REPLACE VIEW customers AS
      WITH latest AS (
        SELECT "anonymousIdentifier", organization,
          MAX("numHostsEnrolled") AS max_hosts
        FROM ${DB_TABLE}
        GROUP BY "anonymousIdentifier", organization
      ),
      ranked AS (
        SELECT *,
          ROW_NUMBER() OVER (PARTITION BY organization ORDER BY max_hosts DESC) AS env_rank,
          COUNT(*) OVER (PARTITION BY organization) AS env_count
        FROM latest
      )
      SELECT "anonymousIdentifier",
        CASE
          WHEN env_count = 1 THEN organization
          WHEN env_rank = 1 THEN organization || ' (Prod)'
          WHEN env_count = 2 THEN organization || ' (Stage)'
          ELSE organization || ' (Stage ' || (env_rank - 1) || ')'
        END AS organization
      FROM ranked;
    `);
    console.log("Created customers view.");
  } catch (err) {
    console.error(`Database error: ${err}`);
    process.exit(1);
  } finally {
    await client.end();
    console.log("Database connection closed.");
  }
}

main().then(() => console.log("Process finished."));
