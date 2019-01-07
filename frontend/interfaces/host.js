import PropTypes from 'prop-types';

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
});
