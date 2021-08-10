import PropTypes from "prop-types";

export default PropTypes.shape({
  id: PropTypes.number.isRequired,
  created_at: PropTypes.string,
  name: PropTypes.string.isRequired,
  description: PropTypes.string,
  agent_options: PropTypes.object, // eslint-disable-line react/forbid-prop-types
  // hosts: PropTypes.number, // is this used anywhere? it's not returned
  // members: PropTypes.number, // is this used anywhere? it's not returned
  // role: PropTypes.string,  // is this used anywhere? it's not returned
  user_count: PropTypes.number,
  host_count: PropTypes.number,
  secrets: PropTypes.object, // eslint-disable-line react/forbid-prop-types
});

/**
 * The shape of a team entity
 */
export interface ITeam {
  id: number;
  created_at?: string;
  name: string;
  description: string;
  agent_options?: any;
  user_count: number;
  host_count: number;
  secrets?: any;
  // role value is included when the team is in the context of a user.
  // role?: string;  // is this used anywhere? it's not returned
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
