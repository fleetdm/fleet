import PropTypes from "prop-types";
import packInterface, { IPack } from "./pack";

export default PropTypes.shape({
  created_at: PropTypes.string,
  updated_at: PropTypes.string,
  id: PropTypes.number,
  name: PropTypes.string,
  description: PropTypes.string,
  query: PropTypes.string,
  saved: PropTypes.bool,
  author_id: PropTypes.number,
  author_name: PropTypes.string,
  observer_can_run: PropTypes.bool,
  packs: PropTypes.arrayOf(packInterface),
  interval: PropTypes.number, // not on fleet/queries
  last_executed: PropTypes.string, // not on fleet/queries
});
export interface IQuery {
  created_at: string;
  updated_at: string;
  id: number;
  name: string;
  description: string;
  query: string;
  saved: boolean;
  author_id: number;
  author_name: string;
  observer_can_run: boolean;
  packs: IPack[];
  interval: number; // not on fleet/queries
  last_executed: string; // not on fleet/queries
}
