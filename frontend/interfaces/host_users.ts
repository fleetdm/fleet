import PropTypes from "prop-types";

export default PropTypes.shape({
  id: PropTypes.number,
  uid: PropTypes.number,
  username: PropTypes.string,
  type: PropTypes.string,
  groupname: PropTypes.string,
});

export interface IHostUser {
  id: number;
  uid: number;
  username: string;
  type: string;
  groupname: string;
}
