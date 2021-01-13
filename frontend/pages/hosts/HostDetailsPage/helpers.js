import { push } from 'react-router-redux';

import PATHS from 'router/paths';
import hostActions from 'redux/nodes/entities/hosts/actions';

export const fetchHost = (dispatch, hostID) => {
  return dispatch(hostActions.load(hostID))
    .catch(() => {
      dispatch(push(PATHS.FLEET_500));

      return false;
    });
};

export default { fetchHost };
