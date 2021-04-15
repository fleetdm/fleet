import PropTypes from "prop-types";

export default PropTypes.shape({
  detail_updated_at: PropTypes.string,
  hostname: PropTypes.string,
  id: PropTypes.number,
  ip: PropTypes.string,
  mac: PropTypes.string,
  memory: PropTypes.number,
  os_version: PropTypes.string,
  osquery_version: PropTypes.string,
  platform: PropTypes.string,
  status: PropTypes.string,
  updated_at: PropTypes.string,
  uptime: PropTypes.number,
  uuid: PropTypes.string,
  seen_time: PropTypes.string,
});

export interface IHost {
  detail_updated_at: string;
  hostname: string;
  id: number;
  ip: string;
  mac: string;
  memory: number;
  os_version: string;
  osquery_version: string;
  platform: string;
  status: string;
  updated_at: string;
  uptime: number;
  uuid: string;
  seen_time: string;
}
