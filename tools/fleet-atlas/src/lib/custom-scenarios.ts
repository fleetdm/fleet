import Database from "better-sqlite3";
import path from "node:path";
import type { Scenario, ScenarioTag, Step } from "@/components/scenarios/scenarios";

const DB_PATH = path.join(process.cwd(), ".fleet-atlas.db");

let db: Database.Database | null = null;

function getDb(): Database.Database {
  if (db) return db;
  db = new Database(DB_PATH);
  db.pragma("journal_mode = WAL");
  db.exec(`
    CREATE TABLE IF NOT EXISTS custom_scenarios (
      slug TEXT PRIMARY KEY,
      label TEXT NOT NULL,
      summary TEXT NOT NULL,
      overview TEXT NOT NULL,
      takeaway TEXT NOT NULL,
      tags_json TEXT NOT NULL,
      steps_json TEXT NOT NULL,
      created_at INTEGER NOT NULL
    );
  `);
  return db;
}

export function slugify(label: string): string {
  return label
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, "-")
    .replace(/^-+|-+$/g, "")
    .slice(0, 60);
}

export type CustomScenarioInput = {
  label: string;
  summary: string;
  overview: string;
  takeaway: string;
  tags: ScenarioTag[];
  steps: Step[];
};

export function listCustomScenarios(): Scenario[] {
  const rows = getDb()
    .prepare(
      "SELECT slug, label, summary, overview, takeaway, tags_json, steps_json FROM custom_scenarios ORDER BY created_at DESC",
    )
    .all() as Array<{
    slug: string;
    label: string;
    summary: string;
    overview: string;
    takeaway: string;
    tags_json: string;
    steps_json: string;
  }>;
  return rows.map((r) => ({
    slug: r.slug,
    label: r.label,
    summary: r.summary,
    overview: r.overview,
    takeaway: r.takeaway,
    tags: JSON.parse(r.tags_json) as ScenarioTag[],
    steps: JSON.parse(r.steps_json) as Step[],
  }));
}

export function getCustomScenario(slug: string): Scenario | undefined {
  const row = getDb()
    .prepare(
      "SELECT slug, label, summary, overview, takeaway, tags_json, steps_json FROM custom_scenarios WHERE slug = ?",
    )
    .get(slug) as
    | {
        slug: string;
        label: string;
        summary: string;
        overview: string;
        takeaway: string;
        tags_json: string;
        steps_json: string;
      }
    | undefined;
  if (!row) return undefined;
  return {
    slug: row.slug,
    label: row.label,
    summary: row.summary,
    overview: row.overview,
    takeaway: row.takeaway,
    tags: JSON.parse(row.tags_json) as ScenarioTag[],
    steps: JSON.parse(row.steps_json) as Step[],
  };
}

export function createCustomScenario(input: CustomScenarioInput): string {
  const base = slugify(input.label) || "scenario";
  let slug = base;
  let n = 1;
  while (
    getDb()
      .prepare("SELECT 1 FROM custom_scenarios WHERE slug = ?")
      .get(slug) !== undefined
  ) {
    n += 1;
    slug = `${base}-${n}`;
  }
  getDb()
    .prepare(
      "INSERT INTO custom_scenarios (slug, label, summary, overview, takeaway, tags_json, steps_json, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)",
    )
    .run(
      slug,
      input.label,
      input.summary,
      input.overview,
      input.takeaway,
      JSON.stringify(input.tags),
      JSON.stringify(input.steps),
      Date.now(),
    );
  return slug;
}

export function deleteCustomScenario(slug: string): void {
  getDb().prepare("DELETE FROM custom_scenarios WHERE slug = ?").run(slug);
}
