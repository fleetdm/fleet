import PropTypes from "prop-types";

export default PropTypes.shape({
  alertType: PropTypes.string,
  isVisible: PropTypes.bool,
  message: PropTypes.string,
  undoAction: PropTypes.func,
});

export interface INotifications {
  alertType: string;
  isVisible: boolean;
  message: string | JSX.Element;
  undoAction: () => void;
}
