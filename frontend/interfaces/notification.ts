import PropTypes from "prop-types";

export default PropTypes.shape({
  alertType: PropTypes.string,
  isVisible: PropTypes.bool,
  message: PropTypes.string,
  dismissOnPageChange: PropTypes.bool,
});

export interface INotification {
  alertType: "success" | "error" | "warning-filled" | null;
  isVisible: boolean;
  message: JSX.Element | string | null;
  dismissOnPageChange?: boolean;
}
