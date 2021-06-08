import PropTypes from "prop-types";

export default PropTypes.shape({
  description: PropTypes.string,
  detail_updated_at: PropTypes.string,
  id: PropTypes.number,
  name: PropTypes.string,
  platform: PropTypes.string,
  updated_at: PropTypes.string,
});

export interface IPack {
  description: string;
  detail_updated_at: string;
  id: number;
  name: string;
  platform: string;
  updated_at: string;
}
