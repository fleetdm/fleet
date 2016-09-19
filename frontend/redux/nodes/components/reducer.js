import { combineReducers } from 'redux';
import ForgotPasswordPage from './ForgotPasswordPage/reducer';
import ResetPasswordPage from './ResetPasswordPage/reducer';

export default combineReducers({
  ForgotPasswordPage,
  ResetPasswordPage,
});

