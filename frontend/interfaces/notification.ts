import PropTypes from "prop-types";

export default PropTypes.shape({
  alertType: PropTypes.string,
  isVisible: PropTypes.bool,
  message: PropTypes.string,
  undoAction: PropTypes.func,
});

export interface INotification {
  alertType: "success" | "error" | "warning-filled" | null;
  isVisible: boolean;
  message: JSX.Element | string | null;
  undoAction?: () => void;
}
