import PropTypes from "prop-types";
import enrollSecretInterface, { IEnrollSecret } from "./enroll_secret";

export default PropTypes.shape({
  id: PropTypes.number.isRequired,
  created_at: PropTypes.string,
  name: PropTypes.string.isRequired,
  description: PropTypes.string,
  agent_options: PropTypes.object, // eslint-disable-line react/forbid-prop-types
  role: PropTypes.string, // role value is included when the team is in the context of a user
  user_count: PropTypes.number,
  host_count: PropTypes.number,
  secrets: PropTypes.arrayOf(enrollSecretInterface),
});

/**
 * The shape of a team entity
 */
export interface ITeam {
  id: number;
  display_text?: string;
  count?: number;
  created_at?: string;
  name: string;
  description: string;
  agent_options?: any;
  user_count: number;
  host_count: number;
  secrets?: IEnrollSecret[];
  role?: string; // role value is included when the team is in the context of a user
}

/**
 * The shape of a new member to add to a team
 */
interface INewMember {
  id: number;
  role: string;
}

/**
 * The shape of the body expected from the API when adding new members to teams
 */
export interface INewMembersBody {
  users: INewMember[];
}
export interface IRemoveMembersBody {
  users: { id: number }[];
}
interface INewTeamSecret {
  team_id: number;
  secret: string;
  created_at?: string;
}
export interface INewTeamSecretBody {
  secrets: INewTeamSecret[];
}
export interface IRemoveTeamSecretBody {
  secrets: { secret: string }[];
}
