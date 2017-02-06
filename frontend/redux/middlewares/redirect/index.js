/* eslint-disable no-unused-vars */
import { get } from 'lodash';
import { push } from 'react-router-redux';

import APP_CONSTANTS from 'app_constants';

const { HTTP_STATUS, PATHS } = APP_CONSTANTS;

const redirectMiddleware = store => next => (action) => {
  const { type, payload } = action;

  if (type.endsWith('FAILURE')) {
    const httpStatus = get(payload, 'errors.http_status');

    if (HTTP_STATUS.INTERNAL_ERROR.test(httpStatus)) {
      store.dispatch(push(PATHS.KOLIDE_500));
    }
  }

  return next(action);
};

export default redirectMiddleware;

