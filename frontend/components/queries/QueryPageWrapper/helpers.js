import { push } from 'react-router-redux';
import { join, values } from 'lodash';

import queryActions from 'redux/nodes/entities/queries/actions';
import { renderFlash } from 'redux/nodes/notifications/actions';

export const fetchQuery = (dispatch, queryID) => {
  return dispatch(queryActions.load(queryID))
    .catch((errors) => {
      const errorMessage = join(values(errors), ', ');

      dispatch(push('/queries/new'));
      dispatch(renderFlash('error', errorMessage));

      return false;
    });
};

export default { fetchQuery };
