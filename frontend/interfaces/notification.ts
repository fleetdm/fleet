import PropTypes from "prop-types";

export default PropTypes.shape({
  alertType: PropTypes.string,
  isVisible: PropTypes.bool,
  message: PropTypes.string,
  persistOnPageChange: PropTypes.bool,
});

export type IAlertType = "success" | "error" | "warning-filled";

export interface INotification {
  alertType: IAlertType | null;
  isVisible: boolean;
  message: JSX.Element | string | null;
  persistOnPageChange?: boolean;
  id?: string;
}
