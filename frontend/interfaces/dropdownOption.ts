import PropTypes from "prop-types";

export default PropTypes.shape({
  disabled: PropTypes.bool,
  label: PropTypes.string,
  value: PropTypes.oneOfType([PropTypes.string, PropTypes.number]),
});

export interface IDropdownOption {
  disabled: boolean;
  label: string;
  value: string | number;
}
