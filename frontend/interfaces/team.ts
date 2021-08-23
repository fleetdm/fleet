import PropTypes from "prop-types";

export default PropTypes.shape({
  description: PropTypes.string,
  name: PropTypes.string,
  id: PropTypes.number,
  hosts: PropTypes.number,
  members: PropTypes.number,
  role: PropTypes.string,
  agent_options: PropTypes.object, // eslint-disable-line react/forbid-prop-types
});

/**
 * The shape of a team entity
 */
export interface ITeam {
  description: string;
  name: string;
  id: number;
  host_count: number;
  user_count: number;
  display_text?: string;
  count?: number;
  // role value is included when the team is in the context of a user.
  role?: string;
  agent_options?: any;
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
