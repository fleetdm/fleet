import Kolide from 'kolide';
import config from 'redux/nodes/entities/labels/config';
import { formatErrorResponse } from 'redux/nodes/entities/base/helpers';

const { actions, extendedActions } = config;
const { loadAllSuccess, loadFailure, successAction } = extendedActions;

const silentLoadAll = () => {
  return (dispatch) => {
    return Kolide.labels.loadAll()
      .then((response) => {
        return dispatch(successAction(response, loadAllSuccess));
      })
      .catch((response) => {
        const errorsObject = formatErrorResponse(response);

        dispatch(loadFailure(errorsObject));

        throw errorsObject;
      });
  };
};

export default { ...actions, silentLoadAll };
