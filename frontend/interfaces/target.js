import PropTypes from "prop-types";
import hostInterface from "interfaces/host";
import labelInterface from "interfaces/label";
// teamInterface added 5/26
import teamInterface from "interfaces/team";

// teamInterface added 5/26
export default PropTypes.oneOfType([
  hostInterface,
  labelInterface,
  teamInterface,
]);
