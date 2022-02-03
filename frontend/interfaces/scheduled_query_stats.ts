import PropTypes, { number } from "prop-types";

export default PropTypes.shape({
  user_time_p50: PropTypes.number,
  user_time_p95: PropTypes.number,
  system_time_p50: PropTypes.number,
  system_time_p95: PropTypes.number,
  total_executions: PropTypes.number,
});

export interface IScheduledQueryStats {
  user_time_p50?: number;
  user_time_p95?: number;
  system_time_p50?: number;
  system_time_p95?: number;
  total_executions?: number;
}
