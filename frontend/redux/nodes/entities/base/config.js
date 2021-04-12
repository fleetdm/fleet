import BaseConfig from "redux/nodes/entities/base/base_config";
import {
  entitiesExceptID,
  orderExceptId,
} from "redux/nodes/entities/base/helpers";

class ReduxConfig extends BaseConfig {
  get actions() {
    return this.allActions();
  }

  get reducer() {
    const { actionTypes, entityName } = this;

    return (state = BaseConfig.initialState, { type, payload }) => {
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
            originalOrder: payload.originalOrder,
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
            originalOrder: payload.originalOrder,
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
              ...entitiesExceptID(state.data, payload.data),
            },
            originalOrder: orderExceptId(state.originalOrder, payload.data),
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
  }
}

export default ReduxConfig;
