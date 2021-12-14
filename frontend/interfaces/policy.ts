import PropTypes from "prop-types";
import { IQueryPlatform } from "interfaces/query";

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
  response: PropTypes.string,
  team_id: PropTypes.number,
  updated_at: PropTypes.string.isRequired,
});

export interface IPolicy {
  id: number;
  name: string;
  query: string;
  description: string;
  author_id: number;
  author_name: string;
  author_email: string;
  resolution: string;
  platform: IQueryPlatform;
  team_id?: number;
  created_at: string;
  updated_at: string;
}

// Used on the manage hosts page and other places where aggregate stats are displayed
export interface IPolicyStats extends IPolicy {
  passing_host_count: number;
  failing_host_count: number;
}

// Used on the host details page and other places where the status of individual hosts are displayed
export interface IHostPolicy extends IPolicy {
  response: string;
}

export interface IPolicyFormData {
  description?: string | number | boolean | any[] | undefined;
  resolution?: string | number | boolean | any[] | undefined;
  platform?: IQueryPlatform;
  name?: string | number | boolean | any[] | undefined;
  query?: string | number | boolean | any[] | undefined;
  team_id?: number;
}

export interface IPolicyNew {
  id?: number;
  key?: number;
  name: string;
  description: string;
  query: string;
  resolution: string;
  platform: IQueryPlatform;
}
