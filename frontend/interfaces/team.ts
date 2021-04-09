import PropTypes from 'prop-types';

export default PropTypes.shape({
  name: PropTypes.string,
  id: PropTypes.number,
  role: PropTypes.string,
});

export interface ITeam {
  name: string;
  id: number;
  role: string;
}
