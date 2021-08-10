import PropTypes from "prop-types";

export default PropTypes.shape({
  version: PropTypes.string,
  branch: PropTypes.string,
  revision: PropTypes.string,
  go_version: PropTypes.string,
  build_date: PropTypes.string,
  build_user: PropTypes.string,
});

export interface IInvite {
  version: string;
  branch: string;
  revision: string;
  go_version: string;
  build_date: string;
  build_user: string;
}
