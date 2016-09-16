/* eslint-disable no-unused-vars */
import kolide from '../../kolide';
import { LOGIN_SUCCESS } from '../nodes/auth/actions';
import local from '../../utilities/local';

const authMiddleware = store => next => action => {
  if (action.type === LOGIN_SUCCESS) {
    const { token } = action.payload.data;
    local.setItem('auth_token', token);
    kolide.setBearerToken(token);
  }

  return next(action);
};

export default authMiddleware;
