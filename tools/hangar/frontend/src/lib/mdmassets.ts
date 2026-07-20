// MDM assets export helpers. A "config" is a saved set of inputs for the
// in-repo tools/mdm/assets exporter (MySQL connection, encryption key, output
// dir, optional single-asset filter) — persisted so an export is one click.
import type { MdmAssetsConfig } from "./ipc";

// Valid single-asset names for the -name filter (from tools/mdm/assets).
export const MDM_ASSET_NAMES = [
  "apns_cert",
  "apns_key",
  "ca_cert",
  "ca_key",
  "abm_cert",
  "abm_key",
  "abm_token",
  "scep_challenge",
  "vpp_token",
];

export function genId(): string {
  if (typeof crypto !== "undefined" && "randomUUID" in crypto) {
    return crypto.randomUUID();
  }
  return `cfg-${Date.now().toString(36)}-${Math.floor(Math.random() * 1e6)}`;
}

// newMdmAssetsConfig seeds a fresh config with the exporter's own defaults
// (fleet / insecure / localhost:3306 / fleet) and the given output dir.
export function newMdmAssetsConfig(defaultDir: string): MdmAssetsConfig {
  return {
    id: genId(),
    name: "local",
    db_user: "fleet",
    db_password: "insecure",
    db_address: "localhost:3306",
    db_name: "fleet",
    key: "",
    dir: defaultDir,
    asset_name: "",
    created_at_ms: 0,
    updated_at_ms: 0,
  };
}
