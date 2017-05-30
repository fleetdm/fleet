import { combineReducers } from 'redux';

import ForgotPasswordPage from './ForgotPasswordPage/reducer';
import ManageHostsPage from './ManageHostsPage/reducer';
import QueryPages from './QueryPages/reducer';
import ResetPasswordPage from './ResetPasswordPage/reducer';
import Decorators from './Decorators/reducer';

export default combineReducers({
  ForgotPasswordPage,
  ManageHostsPage,
  QueryPages,
  ResetPasswordPage,
  Decorators,
});
