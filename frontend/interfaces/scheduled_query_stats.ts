import PropTypes, { number } from "prop-types";

export default PropTypes.shape({
  p50_user_time: PropTypes.number,
  p95_user_time: PropTypes.number,
  p50_system_time: PropTypes.number,
  p95_system_time: PropTypes.number,
  total_executions: PropTypes.number,
});

export interface IScheduledQueryStats {
  p50_user_time?: number;
  p95_user_time?: number;
  p50_system_time?: number;
  p95_system_time?: number;
  total_executions?: number;
}
