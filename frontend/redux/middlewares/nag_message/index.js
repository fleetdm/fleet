/* eslint-disable no-unused-vars */
import actions from 'redux/nodes/persistent_flash/actions';
import helpers from 'redux/middlewares/nag_message/helpers';
import { LICENSE_FAILURE, LICENSE_SUCCESS } from 'redux/nodes/auth/actions';

const nagMessageMiddleware = store => next => (action) => {
  const { type, payload } = action;

  if (type === LICENSE_SUCCESS) {
    if (helpers.shouldNagUser(payload)) {
      store.dispatch(actions.showPersistentFlash('Please upgrade your Kolide license'));
    } else {
      store.dispatch(actions.hidePersistentFlash);
    }
  }

  return next(action);
};

export default nagMessageMiddleware;
