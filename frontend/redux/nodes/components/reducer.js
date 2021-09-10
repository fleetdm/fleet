import { combineReducers } from "redux";

import ForgotPasswordPage from "./ForgotPasswordPage/reducer";
import ManageHostsPage from "./ManageHostsPage/reducer";
import ResetPasswordPage from "./ResetPasswordPage/reducer";

export default combineReducers({
  ForgotPasswordPage,
  ManageHostsPage,
  ResetPasswordPage,
});
