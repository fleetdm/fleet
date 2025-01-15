import PropTypes from "prop-types";
import { CommaSeparatedPlatformString } from "interfaces/platform";
import { IScript } from "./script";

// Legacy PropTypes used on host interface
export default PropTypes.shape({
  author_email: PropTypes.string.isRequired,
  author_id: PropTypes.number.isRequired,
  author_name: PropTypes.string.isRequired,
  created_at: PropTypes.string.isRequired,
  description: PropTypes.string.isRequired,
  id: PropTypes.number.isRequired,
  name: PropTypes.string.isRequired,
  query: PropTypes.string.isRequired,
  resoluton: PropTypes.string.isRequired,
  critical: PropTypes.bool,
  response: PropTypes.string,
  team_id: PropTypes.number,
  updated_at: PropTypes.string.isRequired,
});

export interface IStoredPolicyResponse {
  policy: IPolicy;
}

export interface IPoliciesCountResponse {
  count: number;
}

export interface IPolicy {
  id: number;
  name: string;
  query: string;
  description: string;
  author_id: number;
  author_name: string;
  author_email: string;
  resolution: string;
  platform: CommaSeparatedPlatformString;
  team_id: number | null;
  created_at: string;
  updated_at: string;
  critical: boolean;
  calendar_events_enabled: boolean;
  install_software?: IPolicySoftwareToInstall;
  run_script?: Pick<IScript, "id" | "name">;
}
export interface IPolicySoftwareToInstall {
  name: string;
  software_title_id: number;
}

// Used on the manage hosts page and other places where aggregate stats are displayed
export interface IPolicyStats extends IPolicy {
  passing_host_count: number;
  failing_host_count: number;
  host_count_updated_at: string;
  webhook: string;
  has_run: boolean;
  next_update_ms: number;
}

export interface IPolicyWebhookPreviewPayload {
  id: number;
  name: string;
  query: string;
  description: string;
  author_id: number;
  author_name: string;
  author_email: string;
  resolution: string;
  passing_host_count: number;
  failing_host_count: number;
  critical?: boolean;
}

export type PolicyStatusResponse = "pass" | "fail" | "";

// Used on the host details page and other places where the status of individual hosts are displayed
export interface IHostPolicy extends IPolicy {
  response: PolicyStatusResponse;
}

// Policies API can return {}
export interface ILoadAllPoliciesResponse {
  policies?: IPolicyStats[];
}

// Team policies API can return {}
export interface ILoadTeamPoliciesResponse {
  policies?: IPolicyStats[];
}

export interface ILoadTeamPolicyResponse {
  policy: IPolicyStats;
}

export interface IPolicyFormData {
  description?: string | number | boolean | undefined;
  resolution?: string | number | boolean | undefined;
  critical?: boolean;
  platform?: CommaSeparatedPlatformString;
  name?: string | number | boolean | undefined;
  query?: string | number | boolean | undefined;
  team_id?: number | null;
  id?: number;
  calendar_events_enabled?: boolean;
  software_title_id?: number | null;
  // null for PATCH to unset - note asymmetry with GET/LIST - see IPolicy.run_script
  script_id?: number | null;
}

export interface IPolicyNew {
  id?: number;
  key?: number;
  name: string;
  description: string;
  query: string;
  resolution: string;
  critical: boolean;
  platform: CommaSeparatedPlatformString;
  mdm_required?: boolean;
}
