export const instant = false;

type Link = { title: string; href: string; blurb: string };

const PREREQS: Link[] = [
  {
    title: "Go 1.26+",
    href: "https://go.dev/dl/",
    blurb:
      "The Fleet server, fleetd (orbit), and fleetctl are all Go. Match or exceed the `go` directive in go.mod.",
  },
  {
    title: "Node 24.x + Yarn",
    href: "https://nodejs.org/en/download",
    blurb:
      "The admin UI builds with webpack/Node. Yarn 1 (classic) is the lockfile Fleet commits. Don't use npm or pnpm or yarn 3+.",
  },
  {
    title: "Docker Desktop (or Rancher Desktop)",
    href: "https://www.docker.com/products/docker-desktop/",
    blurb:
      "Fleet's local dev stack runs MySQL, Redis, and a SAML IdP in docker-compose. You need a daemon.",
  },
  {
    title: "Make",
    href: "https://www.gnu.org/software/make/",
    blurb:
      "Every build + run command routes through the Makefile. macOS ships one with Xcode CLT; Windows developers typically use WSL.",
  },
];

const RUN_STACK: Link[] = [
  {
    title: "Clone the repo",
    href: "https://github.com/fleetdm/fleet",
    blurb:
      "git clone git@github.com:fleetdm/fleet.git. Then cd fleet. All commands below assume you're in the repo root.",
  },
  {
    title: "Install deps",
    href: "https://github.com/fleetdm/fleet/blob/main/docs/Contributing/getting-started/building-fleet.md",
    blurb:
      "make deps installs Go modules + runs yarn install. Re-run after pulling if anything's out of date.",
  },
  {
    title: "Spin up MySQL + Redis + dev services",
    href: "https://github.com/fleetdm/fleet/blob/main/docker-compose.yml",
    blurb:
      "docker-compose up -d brings up the dev dependency stack: MySQL on 3306, Redis on 6379, a SAML IdP, Minio (S3-compatible), and localstack.",
  },
  {
    title: "Build + run Fleet",
    href: "https://github.com/fleetdm/fleet/blob/main/docs/Contributing/getting-started/building-fleet.md",
    blurb:
      "make generate-dev builds frontend assets in watch mode. make serve builds the binaries and runs Fleet on port 8080. First-run: open http://localhost:8080 and create the first admin.",
  },
  {
    title: "Reset the DB",
    href: "https://github.com/fleetdm/fleet/blob/main/docs/Contributing/getting-started/testing-and-local-development.md",
    blurb:
      "make db-reset wipes and re-migrates the dev MySQL. Use this when a migration branch collides with yours.",
  },
];

const DB_INSPECTION: Link[] = [
  {
    title: "TablePlus",
    href: "https://tableplus.com/",
    blurb:
      "Mac-native MySQL GUI. Connection: host 127.0.0.1, port 3306, user fleet, password insecure, database fleet. Quick-open with ⌘T.",
  },
  {
    title: "DBeaver",
    href: "https://dbeaver.io/",
    blurb:
      "Cross-platform MySQL GUI, free. Same connection details as TablePlus. Heavier UI but runs on Linux + Windows.",
  },
  {
    title: "MySQL CLI",
    href: "https://dev.mysql.com/doc/refman/8.0/en/mysql.html",
    blurb:
      "mysql -h 127.0.0.1 -u fleet -pinsecure fleet. No GUI, but fast for ad-hoc SELECTs and sanity checks.",
  },
  {
    title: "docker exec into the container",
    href: "https://docs.docker.com/engine/reference/commandline/exec/",
    blurb:
      "docker-compose exec mysql mysql -ufleet -pinsecure fleet. Bypasses needing a MySQL client installed locally.",
  },
];

const REDIS_INSPECTION: Link[] = [
  {
    title: "redis-cli",
    href: "https://redis.io/docs/latest/develop/tools/cli/",
    blurb:
      "redis-cli -h 127.0.0.1 -p 6379. See cron locks (KEYS cron:*), live-query pubsub channels (PUBSUB CHANNELS), rate counters.",
  },
  {
    title: "RedisInsight",
    href: "https://redis.io/insight/",
    blurb:
      "Redis's official GUI. Useful when you need to watch keys change in real time (cron locks acquiring + releasing).",
  },
];

const TESTING: Link[] = [
  {
    title: "Quick Go tests (no DB)",
    href: "https://github.com/fleetdm/fleet/blob/main/docs/Contributing/getting-started/testing-and-local-development.md",
    blurb:
      "go test ./server/fleet/.... fast signal, no external deps. Run after every Go edit.",
  },
  {
    title: "MySQL integration",
    href: "https://github.com/fleetdm/fleet/blob/main/docs/Contributing/getting-started/testing-and-local-development.md",
    blurb:
      "MYSQL_TEST=1 go test ./server/datastore/mysql/.... requires the docker-compose stack running.",
  },
  {
    title: "Service + Redis integration",
    href: "https://github.com/fleetdm/fleet/blob/main/docs/Contributing/getting-started/testing-and-local-development.md",
    blurb:
      "MYSQL_TEST=1 REDIS_TEST=1 go test ./server/service/.... covers the service layer with real datastores.",
  },
  {
    title: "Frontend tests",
    href: "https://github.com/fleetdm/fleet/blob/main/frontend/README.md",
    blurb: "yarn test. Jest. Fast; run after FE edits.",
  },
  {
    title: "Linters",
    href: "https://github.com/fleetdm/fleet/blob/main/.golangci.yml",
    blurb:
      "make lint-go-incremental (changes since main) or make lint-go (full). make lint-js for the frontend.",
  },
];

type Gotcha = {
  title: string;
  symptom: string;
  fix: string;
};

const GOTCHAS: Gotcha[] = [
  {
    title: "\"Cannot connect to the Docker daemon\" from make serve",
    symptom:
      "make serve or docker-compose up fails immediately with a Docker socket error.",
    fix: "Docker Desktop (or Rancher Desktop) isn't running. Launch the app; wait for the whale icon to be solid; try again. make serve doesn't start Docker for you.",
  },
  {
    title: "Dev server port 8080 already in use",
    symptom:
      "make serve errors out with \"bind: address already in use.\"",
    fix: "Another Fleet instance, a stale run, or something else is holding 8080. lsof -i :8080 to see what's there. Kill it, or set FLEET_SERVER_ADDRESS=0.0.0.0:8081 (or similar) before make serve.",
  },
  {
    title: "MySQL migration order conflict with main",
    symptom:
      "make serve or tests fail with \"migration XX is older than migration YY that's already applied.\"",
    fix: "Another migration landed on main while your branch was open and yours now sorts before it. Use the /bump-migration skill or manually rename the file to a timestamp after the newest on main, then make db-reset.",
  },
  {
    title: "make db-reset doesn't clear a corrupted DB",
    symptom:
      "After make db-reset you still see weird data or constraint failures.",
    fix: "docker-compose down -v (the -v nukes volumes), then docker-compose up -d, then make db-reset. The volume survives docker-compose down without -v.",
  },
  {
    title: "Yarn installs classic refuse to run on newer Node",
    symptom:
      "yarn install crashes with a syntax error from node_modules or complains about engines.",
    fix: "Fleet uses yarn 1 (classic); the exact versions it expects live in the repo's `package.json` `engines` and `packageManager` fields. If you've got yarn 3+ installed globally (common from Corepack), run `corepack prepare yarn@$(node -e \"console.log(require('./package.json').packageManager.split('@')[1])\") --activate` in the fleet repo, or check `.nvmrc` / `package.json#engines` and pin to whatever that says. Avoids drift as Fleet bumps versions.",
  },
  {
    title:
      "Linter passes locally but CI fails (or vice versa)",
    symptom:
      "make lint-go is green but a PR check fires lint errors. Or the inverse.",
    fix: "Make sure you're running make lint-go (full) not make lint-go-incremental (changes since main only). Also check your Go version against go.mod; CI pins specific toolchains.",
  },
  {
    title: "TablePlus/DBeaver connection refused",
    symptom:
      "The GUI hangs on \"Testing connection\" or returns refused.",
    fix: "docker-compose ps to confirm the mysql service is up. If it's listed but you still can't connect, the dev password is literally the string insecure. No quotes, no secret-manager fetch. Host = 127.0.0.1 (not localhost in some tools' edge cases). Port = 3306.",
  },
  {
    title: "Frontend edits not reflected in the browser",
    symptom:
      "You edit a .tsx file, refresh, and nothing changes.",
    fix: "make generate-dev needs to be running in a separate terminal. It's the webpack watch. Without it, make serve ships the last compiled build and your edits never make it into the bundle. If generate-dev IS running and you still see stale UI: hard-refresh (⌘⇧R), then check the watch terminal for a compile error that silently kept the old bundle alive.",
  },
  {
    title: "Backend / API edits not reflected in responses",
    symptom:
      "You edit a .go file, hit the API, and the response or behavior is the same as before.",
    fix: "There's no hot reload for Go. make serve compiles once and runs that binary; subsequent edits are invisible until you restart. Stop make serve (Ctrl+C), re-run it, and your changes compile in. If the restart is instant but behavior still looks stale, confirm you're hitting the right instance. E.g. a background make serve from another terminal can still be holding the port.",
  },
  {
    title: "SAML IdP tests fail with connection errors",
    symptom:
      "Integration tests exercising SSO fail even though code looks right.",
    fix: "The dev docker-compose includes a SAML IdP container (test-saml-idp). It takes ~10 seconds after docker-compose up to be ready. Run docker-compose ps and confirm it's healthy before running SSO integration tests.",
  },
  {
    title: "First-run admin creation hangs",
    symptom:
      "You open http://localhost:8080, fill in the admin form, click Create, and it spins forever.",
    fix: "Check make serve's stdout. Usually a MySQL connection error. Docker Desktop restarts sometimes shuffle the mysql port; make db-reset will resync Fleet's view, and re-running docker-compose up -d after a restart gets the services back online.",
  },
];

const FURTHER: Link[] = [
  {
    title: "Building Fleet (contributing docs)",
    href: "https://github.com/fleetdm/fleet/blob/main/docs/Contributing/getting-started/building-fleet.md",
    blurb:
      "The canonical build + run guide. Covers platform-specific gotchas (Windows signing, macOS notarization).",
  },
  {
    title: "Testing + local development",
    href: "https://github.com/fleetdm/fleet/blob/main/docs/Contributing/getting-started/testing-and-local-development.md",
    blurb:
      "Every test env var, every suite, how to run a single test with -run.",
  },
  {
    title: "Fleet UI testing",
    href: "https://github.com/fleetdm/fleet/blob/main/docs/Contributing/guides/ui/fleet-ui-testing.md",
    blurb:
      "FE-specific testing strategy: Jest unit, React Testing Library, Storybook, Cypress.",
  },
  {
    title: "Engineering debugging playbook",
    href: "https://fleetdm.com/handbook/engineering/Debugging",
    blurb:
      "Fleet's house debugging patterns. Pairs with Atlas's scenarios + request tracer pages.",
  },
  {
    title: "Engineering handbook",
    href: "https://fleetdm.com/handbook/engineering",
    blurb:
      "Fleet's engineering rituals, review conventions, releases, deployment.",
  },
];

function CardList({ items }: { items: Link[] }) {
  return (
    <ul className="grid gap-3 md:grid-cols-2">
      {items.map((i) => (
        <li key={i.title}>
          <a
            href={i.href}
            target="_blank"
            rel="noopener"
            className="block h-full rounded border border-zinc-200 bg-white p-4 transition-colors hover:border-zinc-400 hover:bg-zinc-50 dark:border-zinc-800 dark:bg-zinc-950 dark:hover:border-zinc-600 dark:hover:bg-zinc-900"
          >
            <div className="text-sm font-semibold text-zinc-900 dark:text-zinc-100">
              {i.title} ↗
            </div>
            <p className="mt-1 text-xs text-zinc-600 dark:text-zinc-400">
              {i.blurb}
            </p>
          </a>
        </li>
      ))}
    </ul>
  );
}

function SectionHeader({ children }: { children: React.ReactNode }) {
  return (
    <h2 className="mb-4 text-sm font-semibold uppercase tracking-wide text-zinc-500">
      {children}
    </h2>
  );
}

export default function DevSetupPage() {
  return (
    <div className="mx-auto max-w-4xl px-10 py-10">
      <header className="mb-8 border-b border-zinc-200 pb-6 dark:border-zinc-800">
        <h1 className="text-2xl font-semibold text-zinc-900 dark:text-zinc-100">
          Spin up your dev environment
        </h1>
        <p className="mt-2 text-zinc-600 dark:text-zinc-400">
          The minimum setup for running Fleet locally, poking at MySQL and
          Redis, and running tests. Links go out to Fleet&apos;s canonical docs
          on fleetdm.com and the fleetdm/fleet repo; this page is the index so
          a new Fleetie knows where to click first.
        </p>
      </header>

      <section className="mb-10">
        <SectionHeader>Prerequisites</SectionHeader>
        <CardList items={PREREQS} />
      </section>

      <section className="mb-10">
        <SectionHeader>Clone, build, run</SectionHeader>
        <CardList items={RUN_STACK} />
      </section>

      <section className="mb-10">
        <SectionHeader>Look at the database</SectionHeader>
        <p className="mb-3 text-xs text-zinc-600 dark:text-zinc-400">
          Any of these connect to the dev MySQL after{" "}
          <code className="rounded bg-zinc-100 px-1 font-mono text-[11px] text-zinc-800 dark:bg-zinc-900 dark:text-zinc-200">
            docker-compose up
          </code>
          . Credentials come from{" "}
          <code className="rounded bg-zinc-100 px-1 font-mono text-[11px] text-zinc-800 dark:bg-zinc-900 dark:text-zinc-200">
            docker-compose.yml
          </code>{" "}
         . default user{" "}
          <code className="rounded bg-zinc-100 px-1 font-mono text-[11px] text-zinc-800 dark:bg-zinc-900 dark:text-zinc-200">
            fleet
          </code>
          , password{" "}
          <code className="rounded bg-zinc-100 px-1 font-mono text-[11px] text-zinc-800 dark:bg-zinc-900 dark:text-zinc-200">
            insecure
          </code>
          .
        </p>
        <CardList items={DB_INSPECTION} />
      </section>

      <section className="mb-10">
        <SectionHeader>Look at Redis</SectionHeader>
        <CardList items={REDIS_INSPECTION} />
      </section>

      <section className="mb-10">
        <SectionHeader>Run tests</SectionHeader>
        <CardList items={TESTING} />
      </section>

      <section className="mb-10">
        <SectionHeader>Common gotchas</SectionHeader>
        <p className="mb-3 text-xs text-zinc-600 dark:text-zinc-400">
          The things that trip up a first-day setup. Each entry has the symptom
          you&apos;ll see and the fastest fix.
        </p>
        <ul className="space-y-3">
          {GOTCHAS.map((g) => (
            <li
              key={g.title}
              className="rounded border border-zinc-200 bg-white p-4 dark:border-zinc-800 dark:bg-zinc-950"
            >
              <div className="text-sm font-semibold text-zinc-900 dark:text-zinc-100">
                {g.title}
              </div>
              <div className="mt-2 text-xs">
                <span className="font-semibold uppercase tracking-wide text-rose-700 dark:text-rose-400">
                  Symptom:{" "}
                </span>
                <span className="text-zinc-700 dark:text-zinc-300">
                  {g.symptom}
                </span>
              </div>
              <div className="mt-1 text-xs">
                <span className="font-semibold uppercase tracking-wide text-emerald-700 dark:text-emerald-400">
                  Fix:{" "}
                </span>
                <span className="text-zinc-700 dark:text-zinc-300">{g.fix}</span>
              </div>
            </li>
          ))}
        </ul>
      </section>

      <section className="mb-10">
        <SectionHeader>Go deeper</SectionHeader>
        <CardList items={FURTHER} />
      </section>

      <div className="rounded border border-zinc-200 bg-zinc-50 p-4 text-xs text-zinc-600 dark:border-zinc-800 dark:bg-zinc-900 dark:text-zinc-400">
        Something&apos;s out of date? This page is a flat set of pointers.
        Correct it where you&apos;re reading this (<code>src/app/dev-setup/page.tsx</code>) or point the next new person at the canonical Fleet docs.
      </div>
    </div>
  );
}
