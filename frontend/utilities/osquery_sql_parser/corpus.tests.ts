/**
 * Regression test: every osquery SQL query shipped in this repo (standard
 * query library, CIS benchmark packs, dogfood gitops files) must parse.
 *
 * When a grammar change makes one of KNOWN_FAILURES parse, remove it from the
 * list so it is enforced from then on.
 */

import * as fs from "fs";
import * as path from "path";
import { loadAll } from "js-yaml";

import { astify } from ".";

const REPO_ROOT = path.resolve(__dirname, "..", "..", "..");

const QUERY_SOURCE_DIRS = [
  "docs/01-Using-Fleet/standard-query-library",
  "ee/cis",
  "it-and-security",
];

// Keyed by `${src}|${name}` — query names alone are not unique across the
// corpus (the CIS packs reuse names between OS versions).
const KNOWN_FAILURES: Record<string, string> = {
  "docs/01-Using-Fleet/standard-query-library/standard-query-library.yml|Detect active processes with Log4j running":
    "grammar does not yet support bare `count` as a column reference",
  "ee/cis/win-11-intune/l1_win11_intune.yaml|CIS - Ensure 'User Account Control Use Admin Approval Mode' is set to 'Enabled'":
    "the query itself is invalid SQL (unterminated string literal)",
  "it-and-security/lib/all/reports/dex-queries.yml|DEX - Application experience - Adoption gap":
    "grammar does not yet support ORDER BY ... NULLS FIRST/LAST",
};

const failureKey = ({ src, name }: { src: string; name: string }) =>
  `${src}|${name}`;

interface ICorpusQuery {
  src: string;
  name: string;
  query: string;
}

const findYamlFiles = (dir: string): string[] => {
  const files: string[] = [];
  fs.readdirSync(dir, { withFileTypes: true }).forEach((entry) => {
    const p = path.join(dir, entry.name);
    if (entry.isDirectory()) {
      files.push(...findYamlFiles(p));
    } else if (/\.ya?ml$/.test(entry.name)) {
      files.push(p);
    }
  });
  return files;
};

const collectQueries = (node: unknown, src: string, out: ICorpusQuery[]) => {
  if (!node || typeof node !== "object") return;
  if (!Array.isArray(node)) {
    const { query, name } = node as { query?: unknown; name?: unknown };
    if (typeof query === "string" && query.trim()) {
      out.push({ src, name: String(name ?? ""), query });
    }
  }
  Object.values(node).forEach((child) => collectQueries(child, src, out));
};

const loadCorpus = (): ICorpusQuery[] => {
  const corpus: ICorpusQuery[] = [];
  QUERY_SOURCE_DIRS.forEach((dir) => {
    findYamlFiles(path.join(REPO_ROOT, dir)).forEach((file) => {
      const src = path.relative(REPO_ROOT, file);
      loadAll(fs.readFileSync(file, "utf8"), (doc) =>
        collectQueries(doc, src, corpus)
      );
    });
  });
  return corpus;
};

describe("osquery_sql_parser repo query corpus", () => {
  const corpus = loadCorpus();

  it("finds a substantial corpus from every source (extractor sanity check)", () => {
    const countFor = (dir: string) =>
      corpus.filter(({ src }) => src.startsWith(dir)).length;
    expect(countFor("docs/")).toBeGreaterThan(50);
    expect(countFor("ee/cis/")).toBeGreaterThan(1000);
    expect(countFor("it-and-security/")).toBeGreaterThan(50);
  });

  it("parses every query in the repo", () => {
    const failures: string[] = [];
    corpus.forEach((entry) => {
      if (failureKey(entry) in KNOWN_FAILURES) return;
      try {
        astify(entry.query);
      } catch (err) {
        failures.push(`${entry.src} | ${entry.name} | ${err}`);
      }
    });
    expect(failures).toEqual([]);
  });

  it("still fails on the known failures (else remove them from the list)", () => {
    const stillFailing = corpus
      .filter((entry) => failureKey(entry) in KNOWN_FAILURES)
      .filter(({ query }) => {
        try {
          astify(query);
          return false;
        } catch {
          return true;
        }
      })
      .map(failureKey);

    expect(stillFailing.sort()).toEqual(Object.keys(KNOWN_FAILURES).sort());
  });
});
