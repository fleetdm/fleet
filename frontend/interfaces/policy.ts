import PropTypes from "prop-types";
import { IPlatformString } from "interfaces/platform";

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

export interface IPolicy {
  id: number;
  name: string;
  query: string;
  description: string;
  author_id: number;
  author_name: string;
  author_email: string;
  resolution: string;
  platform: IPlatformString;
  team_id?: number;
  created_at: string;
  updated_at: string;
  critical: boolean;
}

// Used on the manage hosts page and other places where aggregate stats are displayed
export interface IPolicyStats extends IPolicy {
  passing_host_count: number;
  failing_host_count: number;
  webhook: string;
  has_run: boolean;
  osquery_policy_ms: number;
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

// Used on the host details page and other places where the status of individual hosts are displayed
export interface IHostPolicy extends IPolicy {
  response: string;
}

export interface ILoadAllPoliciesResponse {
  policies: IPolicyStats[];
}

export interface ILoadTeamPoliciesResponse {
  policies: IPolicyStats[];
  inherited_policies: IPolicyStats[];
}
export interface IPolicyFormData {
  description?: string | number | boolean | undefined;
  resolution?: string | number | boolean | undefined;
  critical?: boolean;
  platform?: IPlatformString;
  name?: string | number | boolean | undefined;
  query?: string | number | boolean | undefined;
  team_id?: number;
  id?: number;
}

export interface IPolicyNew {
  id?: number;
  key?: number;
  name: string;
  description: string;
  query: string;
  resolution: string;
  critical: boolean;
  platform: IPlatformString;
  mdm_required?: boolean;
}
