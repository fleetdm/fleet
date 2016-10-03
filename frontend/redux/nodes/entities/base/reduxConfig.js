import { noop } from 'lodash';
import { normalize, arrayOf } from 'normalizr';

const initialState = {
  loading: false,
  errors: {},
  data: {},
};

const reduxConfig = ({
  entityName,
  loadFunc,
  parseFunc = noop,
  schema,
  updateFunc,
}) => {
  const actionTypes = {
    LOAD_FAILURE: `${entityName}_LOAD_FAILURE`,
    LOAD_REQUEST: `${entityName}_LOAD_REQUEST`,
    LOAD_SUCCESS: `${entityName}_LOAD_SUCCESS`,
    UPDATE_FAILURE: `${entityName}_UPDATE_FAILURE`,
    UPDATE_REQUEST: `${entityName}_UPDATE_REQUEST`,
    UPDATE_SUCCESS: `${entityName}_UPDATE_SUCCESS`,
  };

  const loadFailure = (errors) => {
    return {
      type: actionTypes.LOAD_FAILURE,
      payload: { errors },
    };
  };
  const loadRequest = { type: actionTypes.LOAD_REQUEST };
  const loadSuccess = (data) => {
    return {
      type: actionTypes.LOAD_SUCCESS,
      payload: { data },
    };
  };

  const updateFailure = (errors) => {
    return {
      type: actionTypes.UPDATE_FAILURE,
      payload: { errors },
    };
  };
  const updateRequest = { type: actionTypes.UPDATE_REQUEST };
  const updateSuccess = (data) => {
    return {
      type: actionTypes.UPDATE_SUCCESS,
      payload: { data },
    };
  };

  const parsedResponse = (responseArray) => {
    return responseArray.map(response => {
      return parseFunc(response);
    });
  };

  const load = (...args) => {
    return (dispatch) => {
      dispatch(loadRequest);

      return loadFunc(...args)
        .then(response => {
          if (!response) return [];

          const { entities } = normalize(parsedResponse(response), arrayOf(schema));

          return dispatch(loadSuccess(entities));
        })
        .catch(response => {
          const { errors } = response;

          dispatch(loadFailure(errors));
          throw response;
        });
    };
  };

  const update = (...args) => {
    return (dispatch) => {
      dispatch(updateRequest);

      return updateFunc(...args)
        .then(response => {
          if (!response) return {};
          const { entities } = normalize(parsedResponse([response]), arrayOf(schema));

          return dispatch(updateSuccess(entities));
        })
        .catch(response => {
          const { errors } = response;

          dispatch(updateFailure(errors));
          throw response;
        });
    };
  };

  const actions = {
    load,
    update,
  };

  const reducer = (state = initialState, { type, payload }) => {
    switch (type) {
      case actionTypes.UPDATE_REQUEST:
      case actionTypes.LOAD_REQUEST:
        return {
          ...state,
          loading: true,
        };
      case actionTypes.UPDATE_SUCCESS:
      case actionTypes.LOAD_SUCCESS:
        return {
          ...state,
          loading: false,
          data: {
            ...state.data,
            ...payload.data[entityName],
          },
        };
      case actionTypes.UPDATE_FAILURE:
      case actionTypes.LOAD_FAILURE:
        return {
          ...state,
          loading: false,
          errors: {
            ...payload.errors,
          },
        };
      default:
        return state;
    }
  };

  return {
    actions,
    reducer,
  };
};

export default reduxConfig;
