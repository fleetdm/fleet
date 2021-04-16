import PropTypes from "prop-types";
import { IUser } from "./user";

export default PropTypes.shape({
  name: PropTypes.string,
  id: PropTypes.number,
  hosts: PropTypes.number,
  members: PropTypes.number,
  role: PropTypes.string,
});

export interface ITeam {
  name: string;
  id: number;
  hosts: number;
  members: number | IUser[];
  role?: string;
}
