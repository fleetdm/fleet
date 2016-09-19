/* eslint-disable no-unused-vars */
import { push } from 'react-router-redux';
import kolide from '../../kolide';
import { LOGIN_SUCCESS, LOGOUT_SUCCESS } from '../nodes/auth/actions';
import local from '../../utilities/local';
import paths from '../../router/paths';

const authMiddleware = store => next => action => {
  if (action.type === LOGIN_SUCCESS) {
    const { token } = action.payload.data;

    if (token) {
      local.setItem('auth_token', token);
      kolide.setBearerToken(token);
    }
  }

  if (action.type === LOGOUT_SUCCESS) {
    const { LOGIN } = paths;

    local.clear();
    kolide.setBearerToken(null);
    store.dispatch(push(LOGIN));
  }

  return next(action);
};

export default authMiddleware;
