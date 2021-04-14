import PATHS from "router/paths";
import { push } from "react-router-redux";
import { renderFlash } from "redux/nodes/notifications/actions";
import userActions from "redux/nodes/entities/users/actions";

const confirmEmailChange = (dispatch, token, user) => {
  if (user && token) {
    return dispatch(userActions.confirmEmailChange(user, token))
      .then(() => {
        dispatch(push(PATHS.USER_SETTINGS));
        dispatch(renderFlash("success", "Email updated successfully!"));

        return false;
      })
      .catch(() => {
        dispatch(push(PATHS.LOGIN));

        return false;
      });
  }

  return Promise.resolve();
};

export default { confirmEmailChange };
