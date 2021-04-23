import PropTypes from "prop-types";
import { IUser } from "./user";

export default PropTypes.shape({
  name: PropTypes.string,
  id: PropTypes.number,
  hosts: PropTypes.number,
  members: PropTypes.number,
  role: PropTypes.string,
});

/**
 * The shape of a team entity
 */
export interface ITeam {
  name: string;
  id: number;
  hosts: number;
  members: number | IUser[];
  role?: string;
}

/**
 * The shape of a new member to add to a team
 */
export interface INewMember {
  id: number;
  role: string;
}

/**
 * The shape of the body expected from the API when adding new members to teams
 */
export interface INewMembersBody {
  users: INewMember[];
}
