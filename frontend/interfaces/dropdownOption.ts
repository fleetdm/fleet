import PropTypes from 'prop-types';

export default PropTypes.shape({
  disabled: PropTypes.bool,
  label: PropTypes.string,
  value: PropTypes.string,
});

export interface IDropdownOption {
  disabled: boolean;
  label: string;
  value: string;
}
