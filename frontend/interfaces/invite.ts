import PropTypes from "prop-types";
import teamInterface, { ITeam } from "./team";

export default PropTypes.shape({
  created_at: PropTypes.string,
  updated_at: PropTypes.string,
  id: PropTypes.number,
  invited_by: PropTypes.number,
  email: PropTypes.string,
  name: PropTypes.string,
  sso_enabled: PropTypes.bool,
  global_role: PropTypes.string,
  teams: PropTypes.arrayOf(teamInterface),
});

export interface IInvite {
  created_at: string;
  updated_at: string;
  id: number;
  invited_by: number;
  email: string;
  name: string;
  sso_enabled: boolean;
  global_role: string | null;
  teams: ITeam[];
}
