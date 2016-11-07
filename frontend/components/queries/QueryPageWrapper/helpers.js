import { push } from 'react-router-redux';

import queryActions from 'redux/nodes/entities/queries/actions';
import { renderFlash } from 'redux/nodes/notifications/actions';

export const fetchQuery = (dispatch, queryID) => {
  return dispatch(queryActions.load(queryID))
    .catch((errorResponse) => {
      const { error } = errorResponse;

      dispatch(push('/queries/new'));
      dispatch(renderFlash('error', error));

      return false;
    });
};

export default { fetchQuery };
