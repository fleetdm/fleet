import Link from "next/link";
import Markdown from "@/components/Markdown";

export const instant = false;

const CATEGORIES = [
  {
    label: "Software installers",
    tooltip:
      "Admin-uploaded packages (.pkg/.msi/.deb/.exe). Metadata in software_installers; bytes in the blob store, retrieved by host at install time.",
    body: `
Admin-uploaded packages: \`.msi\`, \`.pkg\`, \`.deb\`, \`.exe\`, etc. These are the actual bytes that get deployed to hosts.

**MySQL metadata**: \`software_installers\` (name, version, hash, size, uploader).
**Blob location**: S3 bucket (prod) or local disk under the install uploads path (dev).
**Delivery**: hosts download via a signed URL / direct stream from Fleet.
`,
  },
  {
    label: "Bootstrap packages",
    tooltip:
      "One admin-authored macOS package installed at the end of DEP setup, typically to deploy a corporate management agent.",
    body: `
A single admin-authored package deployed to a Mac at the end of DEP setup. Think of it as "the thing that installs your corporate agent."

**MySQL metadata**: \`mdm_apple_bootstrap_packages\`.
**Blob location**: same filestore as installers.
`,
  },
  {
    label: "Config profiles",
    tooltip:
      "Apple .mobileconfig XML, Microsoft SyncML XML, Android policy JSON. Small payloads inline in MySQL; large bundled assets in the blob store.",
    body: `
Apple \`.mobileconfig\` XML, Microsoft SyncML XML, Android policy JSON. The data is text, but Fleet treats the payload as an opaque blob.

**MySQL metadata**: \`mdm_apple_configuration_profiles\`, \`mdm_apple_declarations\`, \`mdm_windows_configuration_profiles\`, \`mdm_android_configuration_profiles\`.
**Blob location**: inline in MySQL for the small ones; declarations with asset bundles store blobs externally and reference them via \`mdm_apple_declaration_assets\`.
`,
  },
  {
    label: "Scripts",
    tooltip:
      "Admin-authored shell/PowerShell scripts run on hosts. Content lives in MySQL (dedup by hash in script_contents) rather than the blob store.",
    body: `
Admin-authored shell / PowerShell scripts run on hosts.

**MySQL metadata**: \`scripts\` (per-fleet script list), \`script_contents\` (dedup table keyed by hash).
**Blob location**: content lives in MySQL (small blobs). Larger payloads for setup-experience workflows reference the installer filestore.
`,
  },
  {
    label: "VPP / in-house apps",
    tooltip:
      "App Store VPP apps are delivered by Apple (Fleet stores only tokens/metadata). In-house (private IPA) apps store their payload in the blob store.",
    body: `
Apple App Store volume license (VPP) apps don&apos;t have Fleet-owned installers; delivery is through Apple&apos;s servers. Fleet stores tokens and metadata only.

In-house apps (private IPAs) store their payload in the blob store.

**MySQL metadata**: \`vpp_apps\`, \`vpp_tokens\`, \`in_house_apps\`.
`,
  },
];

export default function FilestorePage() {
  return (
    <div className="mx-auto max-w-3xl px-10 py-10">
      <nav className="mb-6 text-xs text-zinc-500">
        <Link href="/data" className="hover:text-zinc-900 dark:hover:text-zinc-200">
          ← Data layer
        </Link>
      </nav>
      <header className="mb-8 border-b border-zinc-200 pb-6 dark:border-zinc-800">
        <h1 className="text-2xl font-semibold text-zinc-900 dark:text-zinc-100">
          File store
        </h1>
        <p className="mt-2 text-zinc-600 dark:text-zinc-400">
          Blobs that would be awkward in MySQL. S3-compatible in production, a
          local directory in dev. MySQL always holds the metadata; the file
          store holds the bytes.
        </p>
      </header>

      {CATEGORIES.map((c) => (
        <section
          key={c.label}
          className="mb-8 rounded border border-zinc-200 bg-white p-5 dark:border-zinc-800 dark:bg-zinc-950"
        >
          <h2 className="text-lg font-semibold text-zinc-900 dark:text-zinc-100">
            {c.label}
          </h2>
          <div className="mt-3">
            <Markdown>{c.body}</Markdown>
          </div>
        </section>
      ))}
    </div>
  );
}
