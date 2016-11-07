import { combineReducers } from 'redux';

import ForgotPasswordPage from './ForgotPasswordPage/reducer';
import QueryPages from './QueryPages/reducer';
import ResetPasswordPage from './ResetPasswordPage/reducer';

export default combineReducers({
  ForgotPasswordPage,
  QueryPages,
  ResetPasswordPage,
});

