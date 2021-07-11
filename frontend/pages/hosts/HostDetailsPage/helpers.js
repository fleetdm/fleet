import { push } from "connected-react-router";
import PATHS from "router/paths";
import hostActions from "redux/nodes/entities/hosts/actions";

export const fetchHost = (dispatch, hostID) => {
  return dispatch(hostActions.load(hostID)).catch(() => {
    dispatch(push(PATHS.FLEET_500));

    return false;
  });
};

export const destroyHost = (dispatch, host) => {
  return dispatch(hostActions.destroy(host)).then(() => {
    dispatch(push(PATHS.HOME));
  });
};

export const refetchHost = (dispatch, host) => {
  return dispatch(hostActions.refetchHostStart(host));
};

export const queryHostCustom = (dispatch, host) => {
  return dispatch(
    push({
      pathname: PATHS.NEW_QUERY,
      query: { host_ids: [host.id] },
    })
  );
};

export const queryHostSaved = (dispatch, host, selectedQuery) => {
  return dispatch(
    push({
      pathname: PATHS.EDIT_QUERY(selectedQuery),
      query: { host_ids: [host.id] },
    })
  );
};

export default {
  fetchHost,
  destroyHost,
  queryHostCustom,
  queryHostSaved,
  refetchHost,
};
