"use client";

import { useMemo, useState } from "react";
import {
  CODE_ANCHORS,
  FEATURE_MATRIX,
  countPremiumOnlyRows,
  countSharedRows,
  type TierSupport,
} from "@/components/tiers/tiers-data";
import TierQuiz from "@/components/tiers/TierQuiz";
import { editorLink } from "@/lib/config";

const SUPPORT_LABEL: Record<TierSupport, string> = {
  yes: "✓",
  no: "✗",
  limited: "limited",
};

const SUPPORT_CELL: Record<TierSupport, string> = {
  yes: "border-emerald-300 bg-emerald-50 text-emerald-800 dark:border-emerald-800 dark:bg-emerald-950/40 dark:text-emerald-200",
  no: "border-zinc-200 bg-white text-zinc-400 dark:border-zinc-800 dark:bg-zinc-950 dark:text-zinc-600",
  limited:
    "border-amber-300 bg-amber-50 text-amber-800 dark:border-amber-800 dark:bg-amber-950/40 dark:text-amber-200",
};

export default function TiersPage() {
  const [onlyFree, setOnlyFree] = useState(false);
  const [query, setQuery] = useState("");

  const categories = useMemo(() => {
    const q = query.trim().toLowerCase();
    return FEATURE_MATRIX.map((c) => ({
      ...c,
      rows: c.rows.filter((r) => {
        if (onlyFree && !(r.free === "yes" && r.premium === "yes")) return false;
        if (!q) return true;
        return (
          r.feature.toLowerCase().includes(q) ||
          (r.note?.toLowerCase().includes(q) ?? false) ||
          c.label.toLowerCase().includes(q)
        );
      }),
    })).filter((c) => c.rows.length > 0);
  }, [onlyFree, query]);

  const premiumOnly = countPremiumOnlyRows();
  const shared = countSharedRows();

  return (
    <div className="mx-auto max-w-4xl px-10 py-10">
      <header className="mb-8 border-b border-zinc-200 pb-6 dark:border-zinc-800">
        <h1 className="text-2xl font-semibold text-zinc-900 dark:text-zinc-100">
          Fleet Free vs Fleet Premium
        </h1>
        <p className="mt-2 text-zinc-600 dark:text-zinc-400">
          The tier distinction is cross-cutting: it shapes the <code>ee/</code>{" "}
          wrapper pattern, almost every scenario, and almost every{" "}
          &ldquo;can the customer actually do X?&rdquo; conversation. Of the
          features cataloged below, {shared} are available in both tiers and{" "}
          {premiumOnly} are premium-only.
        </p>
      </header>

      <section className="mb-10">
        <div className="mb-4 flex flex-wrap items-center gap-3">
          <input
            type="text"
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            placeholder="Search feature matrix..."
            className="flex-1 rounded border border-zinc-300 bg-white px-3 py-2 text-sm text-zinc-900 placeholder:text-zinc-400 focus:border-zinc-500 focus:outline-none dark:border-zinc-700 dark:bg-zinc-900 dark:text-zinc-100 dark:placeholder:text-zinc-500"
          />
          <label className="inline-flex cursor-pointer items-center gap-2 text-sm text-zinc-700 dark:text-zinc-300">
            <input
              type="checkbox"
              checked={onlyFree}
              onChange={(e) => setOnlyFree(e.target.checked)}
              className="h-4 w-4 accent-zinc-900 dark:accent-zinc-100"
            />
            Show free-only
          </label>
        </div>

        {categories.length === 0 ? (
          <p className="text-sm text-zinc-500">No matches.</p>
        ) : (
          categories.map((c) => (
            <div key={c.slug} className="mb-8">
              <div className="mb-2">
                <h2 className="text-lg font-semibold text-zinc-900 dark:text-zinc-100">
                  {c.label}
                </h2>
                <p className="mt-1 text-sm text-zinc-600 dark:text-zinc-400">
                  {c.blurb}
                </p>
              </div>
              <div className="overflow-hidden rounded border border-zinc-200 dark:border-zinc-800">
                <table className="w-full text-sm">
                  <thead>
                    <tr className="bg-zinc-50 text-left text-xs font-semibold uppercase tracking-wide text-zinc-500 dark:bg-zinc-900">
                      <th className="w-full px-4 py-2">Feature</th>
                      <th className="w-20 px-3 py-2 text-center">Free</th>
                      <th className="w-20 px-3 py-2 text-center">Premium</th>
                    </tr>
                  </thead>
                  <tbody>
                    {c.rows.map((r, i) => (
                      <tr
                        key={i}
                        className="border-t border-zinc-200 dark:border-zinc-800"
                      >
                        <td className="px-4 py-3">
                          <div className="text-zinc-900 dark:text-zinc-100">
                            {r.feature}
                          </div>
                          {r.note ? (
                            <div className="mt-0.5 text-xs text-zinc-500">
                              {r.note}
                            </div>
                          ) : null}
                        </td>
                        <td className="px-3 py-3 text-center align-top">
                          <span
                            className={`inline-flex min-w-10 justify-center rounded border px-2 py-0.5 text-xs font-semibold ${SUPPORT_CELL[r.free]}`}
                          >
                            {SUPPORT_LABEL[r.free]}
                          </span>
                        </td>
                        <td className="px-3 py-3 text-center align-top">
                          <span
                            className={`inline-flex min-w-10 justify-center rounded border px-2 py-0.5 text-xs font-semibold ${SUPPORT_CELL[r.premium]}`}
                          >
                            {SUPPORT_LABEL[r.premium]}
                          </span>
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            </div>
          ))
        )}
      </section>

      <section className="mb-10 rounded border border-zinc-200 bg-white p-6 dark:border-zinc-800 dark:bg-zinc-950">
        <h2 className="text-lg font-semibold text-zinc-900 dark:text-zinc-100">
          How the tier gets enforced
        </h2>
        <p className="mt-2 text-sm text-zinc-700 dark:text-zinc-300">
          Fleet avoids sprinkling <code>if (premium)</code> through its codebase.
          Instead there are two complementary patterns.
        </p>

        <h3 className="mt-5 text-sm font-semibold uppercase tracking-wide text-zinc-700 dark:text-zinc-300">
          1. Inline license checks
        </h3>
        <p className="mt-1 text-sm text-zinc-700 dark:text-zinc-300">
          Service methods consult <code>license.IsPremium()</code> (sync) or{" "}
          <code>license.IsPremium(ctx)</code> (ctx-based). When a free-tier
          caller hits a premium-only path, the method returns the sentinel{" "}
          <code>ErrMissingLicense</code>, which the HTTP layer renders as a 402.
        </p>
        <ul className="mt-3 space-y-2 text-sm">
          {[CODE_ANCHORS.tierConst, CODE_ANCHORS.isPremium, CODE_ANCHORS.isPremiumCtx, CODE_ANCHORS.errMissingLicense].map((a) => (
            <li
              key={a.path}
              className="rounded border border-zinc-200 bg-zinc-50 p-3 dark:border-zinc-800 dark:bg-zinc-900"
            >
              <a
                href={editorLink(a.path)}
                className="font-mono text-xs text-zinc-800 hover:text-black dark:text-zinc-200 dark:hover:text-white"
              >
                {a.path}
              </a>
              <div className="mt-0.5 text-xs text-zinc-600 dark:text-zinc-400">
                {a.hint}
              </div>
            </li>
          ))}
        </ul>

        <h3 className="mt-6 text-sm font-semibold uppercase tracking-wide text-zinc-700 dark:text-zinc-300">
          2. The enterprise wrapper (<code>ee/</code>)
        </h3>
        <p className="mt-1 text-sm text-zinc-700 dark:text-zinc-300">
          The service is a Go interface. The core implementation lives in{" "}
          <code>server/service/</code>. The <code>ee/</code> implementation
          embeds the core struct and overrides specific methods with
          license-aware behavior. The real service used at runtime is the ee
          wrapper around the core.
        </p>
        <ul className="mt-3 space-y-2 text-sm">
          <li className="rounded border border-zinc-200 bg-zinc-50 p-3 dark:border-zinc-800 dark:bg-zinc-900">
            <a
              href={editorLink(CODE_ANCHORS.eePackage.path)}
              className="font-mono text-xs text-zinc-800 hover:text-black dark:text-zinc-200 dark:hover:text-white"
            >
              {CODE_ANCHORS.eePackage.path}
            </a>
            <div className="mt-0.5 text-xs text-zinc-600 dark:text-zinc-400">
              {CODE_ANCHORS.eePackage.hint}
            </div>
          </li>
        </ul>
        <p className="mt-3 text-sm text-zinc-700 dark:text-zinc-300">
          Open-source contributors can read everything in <code>server/</code>{" "}
          but only Fleet staff change code in <code>ee/</code>. Licensing is
          thus both a product boundary and a code boundary.
        </p>

        <h3 className="mt-6 text-sm font-semibold uppercase tracking-wide text-zinc-700 dark:text-zinc-300">
          3. Fleet Free is the negation
        </h3>
        <p className="mt-1 text-sm text-zinc-700 dark:text-zinc-300">
          There is no <code>IsFree()</code> anywhere in the codebase. Backend
          and frontend both check premium; Free is always derived as the
          negation of premium.
        </p>
        <div className="mt-3 rounded border border-zinc-200 bg-zinc-50 p-3 text-sm text-zinc-800 dark:border-zinc-800 dark:bg-zinc-900 dark:text-zinc-200">
          <div className="font-mono text-xs text-zinc-600 dark:text-zinc-400">
            backend (Go) · frontend (TypeScript)
          </div>
          <div className="mt-2 grid gap-2 sm:grid-cols-2">
            <div>
              <div className="text-xs font-semibold text-zinc-500 dark:text-zinc-400">
                Backend
              </div>
              <code className="text-xs">license.IsPremium()</code> /{" "}
              <code className="text-xs">license.IsPremium(ctx)</code>
            </div>
            <div>
              <div className="text-xs font-semibold text-zinc-500 dark:text-zinc-400">
                Frontend
              </div>
              <code className="text-xs">isPremiumTier</code> (boolean selector
              from <code className="text-xs">AppContext</code>)
            </div>
          </div>
        </div>
        <p className="mt-3 text-sm text-zinc-700 dark:text-zinc-300">
          In frontend code, <code>isPremiumTier</code> is used two ways and the
          direction matters:
        </p>
        <ul className="mt-2 space-y-2 text-sm text-zinc-800 dark:text-zinc-200">
          <li className="rounded border border-zinc-200 bg-zinc-50 p-3 dark:border-zinc-800 dark:bg-zinc-900">
            <div className="font-semibold">
              <code>isPremiumTier</code>: show something only to Premium
            </div>
            <p className="mt-1 text-xs text-zinc-600 dark:text-zinc-400">
              Gate a premium-only control, column, modal, or route. Example:{" "}
              <code>{`{isPremiumTier && <TeamsDropdown />}`}</code> renders the
              dropdown only on Premium; Free sees nothing.
            </p>
          </li>
          <li className="rounded border border-zinc-200 bg-zinc-50 p-3 dark:border-zinc-800 dark:bg-zinc-900">
            <div className="font-semibold">
              <code>!isPremiumTier</code>: show something only to Free
            </div>
            <p className="mt-1 text-xs text-zinc-600 dark:text-zinc-400">
              Swap in an upgrade affordance, a disabled state, or a Free-only
              empty state. Example:{" "}
              <code>{`{!isPremiumTier && <UpgradeCTA />}`}</code> renders the
              CTA only on Free.
            </p>
          </li>
        </ul>
        <p className="mt-3 text-sm text-zinc-700 dark:text-zinc-300">
          The negation is load-bearing. Dropping the <code>!</code> flips the
          audience silently: a Free-only upgrade CTA becomes a
          Premium-only upgrade CTA, which the TypeScript compiler cannot catch
          because the type is still <code>boolean</code>. Check
          <code> !isPremiumTier</code> any time the thing you are rendering is
          FOR Free users.
        </p>
      </section>

      <section className="mb-10 rounded border border-zinc-200 bg-white p-6 dark:border-zinc-800 dark:bg-zinc-950">
        <h2 className="text-lg font-semibold text-zinc-900 dark:text-zinc-100">
          What breaks if you run Free
        </h2>
        <p className="mt-2 text-sm text-zinc-700 dark:text-zinc-300">
          A walkthrough: a free-tier admin tries to use a premium-gated feature
          (say, enabling SSO).
        </p>
        <ol className="mt-4 space-y-3 text-sm text-zinc-800 dark:text-zinc-200">
          <li className="rounded border border-zinc-200 bg-zinc-50 p-3 dark:border-zinc-800 dark:bg-zinc-900">
            <div className="font-semibold">1. Admin opens the Settings page</div>
            <p className="mt-1 text-xs text-zinc-600 dark:text-zinc-400">
              The UI fetches <code>GET /api/latest/fleet/config</code>. The
              response includes a <code>license.tier</code> field.
            </p>
          </li>
          <li className="rounded border border-zinc-200 bg-zinc-50 p-3 dark:border-zinc-800 dark:bg-zinc-900">
            <div className="font-semibold">
              2. UI renders the SSO section as read-only
            </div>
            <p className="mt-1 text-xs text-zinc-600 dark:text-zinc-400">
              Components gated with <code>!isPremiumTier</code> swap the real
              form for an &ldquo;Upgrade to Premium&rdquo; affordance.
            </p>
          </li>
          <li className="rounded border border-zinc-200 bg-zinc-50 p-3 dark:border-zinc-800 dark:bg-zinc-900">
            <div className="font-semibold">
              3. If the UI is bypassed, the server stops the write
            </div>
            <p className="mt-1 text-xs text-zinc-600 dark:text-zinc-400">
              <code>svc.ModifyAppConfig</code> is wrapped in{" "}
              <code>ee/server/service/appconfig.go</code> to call{" "}
              <code>license.IsPremium()</code> before any SSO-field write.
              Free-tier callers get <code>ErrMissingLicense</code>, which the
              HTTP layer renders as HTTP 402 with a human-readable error.
            </p>
          </li>
          <li className="rounded border border-zinc-200 bg-zinc-50 p-3 dark:border-zinc-800 dark:bg-zinc-900">
            <div className="font-semibold">
              4. The activity log records the attempt
            </div>
            <p className="mt-1 text-xs text-zinc-600 dark:text-zinc-400">
              Even failed premium-gated writes land in the audit log. The admin
              can&apos;t silently probe.
            </p>
          </li>
        </ol>
      </section>

      <TierQuiz />
    </div>
  );
}
