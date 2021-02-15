import hostActions from 'redux/nodes/entities/hosts/actions';
import labelActions from 'redux/nodes/entities/labels/actions';

export const GET_HOSTS = 'GET_HOSTS';

export const getHostsDataSuccess = (page, perPage, selectedLabel) => {
  return {
    type: GET_HOSTS,
    payload: {
      page,
      perPage,
      selectedLabel,
    },
  };
};

export const getHostsData = (page, perPage, selectedLabel) => (dispatch) => {
  const promises = [
    dispatch(hostActions.loadAll(page, perPage, selectedLabel)),
    dispatch(labelActions.silentLoadAll()),
    dispatch(silentGetStatusLabelCounts),
  ];

  Promise.all(promises).then(dispatch(getHostsDataSuccess(page, perPage, selectedLabel)));
};
