import { isArray, noop } from 'lodash';
import { normalize, arrayOf } from 'normalizr';

import { entitiesExceptID, formatErrorResponse } from 'redux/nodes/entities/base/helpers';

export const initialState = {
  loading: false,
  errors: {},
  data: {},
};

const reduxConfig = ({
  createFunc = noop,
  destroyFunc,
  entityName,
  loadAllFunc,
  loadFunc,
  parseApiResponseFunc,
  parseEntityFunc,
  schema,
  updateFunc,
}) => {
  const actionTypes = {
    CLEAR_ERRORS: `${entityName}_CLEAR_ERRORS`,
    CREATE_FAILURE: `${entityName}_CREATE_FAILURE`,
    CREATE_REQUEST: `${entityName}_CREATE_REQUEST`,
    CREATE_SUCCESS: `${entityName}_CREATE_SUCCESS`,
    DESTROY_FAILURE: `${entityName}_DESTROY_FAILURE`,
    DESTROY_REQUEST: `${entityName}_DESTROY_REQUEST`,
    DESTROY_SUCCESS: `${entityName}_DESTROY_SUCCESS`,
    LOAD_ALL_SUCCESS: `${entityName}_LOAD_ALL_SUCCESS`,
    LOAD_FAILURE: `${entityName}_LOAD_FAILURE`,
    LOAD_REQUEST: `${entityName}_LOAD_REQUEST`,
    LOAD_SUCCESS: `${entityName}_LOAD_SUCCESS`,
    UPDATE_FAILURE: `${entityName}_UPDATE_FAILURE`,
    UPDATE_REQUEST: `${entityName}_UPDATE_REQUEST`,
    UPDATE_SUCCESS: `${entityName}_UPDATE_SUCCESS`,
  };

  const clearErrors = {
    type: actionTypes.CLEAR_ERRORS,
  };

  const createFailure = (errors) => {
    return {
      type: actionTypes.CREATE_FAILURE,
      payload: { errors },
    };
  };
  const createRequest = { type: actionTypes.CREATE_REQUEST };
  const createSuccess = (data) => {
    return {
      type: actionTypes.CREATE_SUCCESS,
      payload: { data },
    };
  };

  const destroyFailure = (errors) => {
    return {
      type: actionTypes.DESTROY_FAILURE,
      payload: { errors },
    };
  };
  const destroyRequest = { type: actionTypes.DESTROY_REQUEST };
  const destroySuccess = (id) => {
    return {
      type: actionTypes.DESTROY_SUCCESS,
      payload: { id },
    };
  };

  const loadFailure = (errors) => {
    return {
      type: actionTypes.LOAD_FAILURE,
      payload: { errors },
    };
  };
  const loadAllSuccess = (data) => {
    return {
      type: actionTypes.LOAD_ALL_SUCCESS,
      payload: { data },
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

  const parse = (apiResponse) => {
    if (!parseApiResponseFunc && !parseEntityFunc) {
      return apiResponse;
    }

    const entitiesArray = parseApiResponseFunc
      ? parseApiResponseFunc(apiResponse)
      : apiResponse;

    if (!parseEntityFunc) {
      return entitiesArray;
    }

    return entitiesArray.map((entity) => {
      return parseEntityFunc(entity);
    });
  };

  const successAction = (response, thunk) => {
    if (!response) return {};

    const parsable = isArray(response) ? response : [response];

    const { entities } = normalize(parse(parsable), arrayOf(schema));

    return thunk(entities);
  };

  const create = (...args) => {
    return (dispatch) => {
      dispatch(createRequest);

      return createFunc(...args)
        .then((response) => {
          dispatch(successAction(response, createSuccess));

          return response;
        })
        .catch((response) => {
          const errorsObject = formatErrorResponse(response);

          dispatch(createFailure(errorsObject));

          throw errorsObject;
        });
    };
  };

  const destroy = (...args) => {
    return (dispatch) => {
      dispatch(destroyRequest);

      return destroyFunc(...args)
        .then(() => {
          const { id: entityID } = args[0];

          return dispatch(destroySuccess(entityID));
        })
        .catch((response) => {
          const errorsObject = formatErrorResponse(response);

          dispatch(destroyFailure(errorsObject));

          throw errorsObject;
        });
    };
  };

  const load = (...args) => {
    return (dispatch) => {
      dispatch(loadRequest);

      return loadFunc(...args)
        .then((response) => {
          return dispatch(successAction(response, loadSuccess));
        })
        .catch((response) => {
          const errorsObject = formatErrorResponse(response);

          dispatch(loadFailure(errorsObject));

          throw errorsObject;
        });
    };
  };

  const loadAll = (...args) => {
    return (dispatch) => {
      dispatch(loadRequest);

      return loadAllFunc(...args)
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

  const update = (...args) => {
    return (dispatch) => {
      dispatch(updateRequest);

      return updateFunc(...args)
        .then((response) => {
          dispatch(successAction(response, updateSuccess));

          return response;
        })
        .catch((response) => {
          const errorsObject = formatErrorResponse(response);

          dispatch(updateFailure(errorsObject));

          throw errorsObject;
        });
    };
  };

  const actions = {
    clearErrors,
    create,
    destroy,
    load,
    loadAll,
    update,
  };

  const reducer = (state = initialState, { type, payload }) => {
    switch (type) {
      case actionTypes.CLEAR_ERRORS:
        return {
          ...state,
          errors: {},
        };
      case actionTypes.CREATE_REQUEST:
      case actionTypes.DESTROY_REQUEST:
      case actionTypes.LOAD_REQUEST:
      case actionTypes.UPDATE_REQUEST:
        return {
          ...state,
          errors: {},
          loading: true,
        };
      case actionTypes.LOAD_ALL_SUCCESS:
        return {
          ...state,
          loading: false,
          errors: {},
          data: {
            ...payload.data[entityName],
          },
        };
      case actionTypes.CREATE_SUCCESS:
      case actionTypes.UPDATE_SUCCESS:
      case actionTypes.LOAD_SUCCESS:
        return {
          ...state,
          loading: false,
          errors: {},
          data: {
            ...state.data,
            ...payload.data[entityName],
          },
        };
      case actionTypes.DESTROY_SUCCESS: {
        return {
          ...state,
          loading: false,
          errors: {},
          data: {
            ...entitiesExceptID(state.data, payload.id),
          },
        };
      }
      case actionTypes.CREATE_FAILURE:
      case actionTypes.DESTROY_FAILURE:
      case actionTypes.UPDATE_FAILURE:
      case actionTypes.LOAD_FAILURE:
        return {
          ...state,
          loading: false,
          errors: payload.errors,
        };
      default:
        return state;
    }
  };

  return {
    actions,
    extendedActions: {
      clearErrors,
      createFailure,
      createRequest,
      createSuccess,
      destroyFailure,
      destroyRequest,
      destroySuccess,
      loadFailure,
      loadRequest,
      loadSuccess,
      successAction,
      updateFailure,
      updateRequest,
      updateSuccess,
    },
    reducer,
  };
};

export default reduxConfig;
