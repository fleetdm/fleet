import { capitalize, isArray } from "lodash";
import { normalize, arrayOf } from "normalizr";

import { formatErrorResponse } from "redux/nodes/entities/base/helpers";

class BaseConfig {
  constructor(inputs) {
    this.createFunc = inputs.createFunc;
    this.destroyFunc = inputs.destroyFunc;
    this.entityName = inputs.entityName;
    this.loadAllFunc = inputs.loadAllFunc;
    this.loadFunc = inputs.loadFunc;
    this.parseApiResponseFunc = inputs.parseApiResponseFunc;
    this.parseEntityFunc = inputs.parseEntityFunc;
    this.schema = inputs.schema;
    this.successAction = this.successAction.bind(this);
    this.updateFunc = inputs.updateFunc;

    this._clearErrors = this._clearErrors.bind(this);
    this._genericActions = this._genericActions.bind(this);
    this._parse = this._parse.bind(this);
  }

  static initialState = {
    loading: false,
    errors: {},
    data: {},
    originalOrder: [],
  };

  static TYPES = {
    CREATE: "CREATE",
    DESTROY: "DESTROY",
    LOAD: "LOAD",
    LOAD_ALL: "LOAD_ALL",
    UPDATE: "UPDATE",
  };

  static failureActionTypeFor(actionTypes, type) {
    const { TYPES } = BaseConfig;

    switch (type) {
      case TYPES.CREATE:
        return actionTypes.CREATE_FAILURE;
      case TYPES.DESTROY:
        return actionTypes.DESTROY_FAILURE;
      case TYPES.LOAD:
      case TYPES.LOAD_ALL:
        return actionTypes.LOAD_FAILURE;
      case TYPES.UPDATE:
        return actionTypes.UPDATE_FAILURE;
      default:
        throw new Error(`Unknown failure type: ${type}`);
    }
  }

  static successActionTypeFor(actionTypes, type) {
    const { TYPES } = BaseConfig;

    switch (type) {
      case TYPES.CREATE:
        return actionTypes.CREATE_SUCCESS;
      case TYPES.DESTROY:
        return actionTypes.DESTROY_SUCCESS;
      case TYPES.LOAD:
        return actionTypes.LOAD_SUCCESS;
      case TYPES.LOAD_ALL:
        return actionTypes.LOAD_ALL_SUCCESS;
      case TYPES.UPDATE:
        return actionTypes.UPDATE_SUCCESS;
      default:
        throw new Error(`Unknown success type: ${type}`);
    }
  }

  get actionTypes() {
    const { entityName } = this;

    return {
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
  }

  allActions() {
    const { TYPES } = BaseConfig;

    return {
      clearErrors: this._clearErrors,
      loadAll: this._genericThunkAction(TYPES.LOAD_ALL),
      loadAllSuccess: this._genericSuccess(TYPES.LOAD_ALL),
      silentLoadAll: this._genericThunkAction(TYPES.LOAD_ALL, { silent: true }),
      successAction: this.successAction,
      ...this._genericActions(TYPES.CREATE),
      ...this._genericActions(TYPES.DESTROY),
      ...this._genericActions(TYPES.LOAD),
      ...this._genericActions(TYPES.UPDATE),
    };
  }

  successAction(apiResponse, thunk) {
    let response = apiResponse;
    if (!response) {
      response = {};
    }

    const { _parse, schema } = this;
    const parsable = isArray(response) ? response : [response];
    const parsed = _parse(parsable);
    const { entities, result } = normalize(parsed, arrayOf(schema));

    return thunk(entities, result);
  }

  // PRIVATE METHODS

  _apiCallForType(type) {
    const { TYPES } = BaseConfig;

    switch (type) {
      case TYPES.CREATE:
        return this.createFunc;
      case TYPES.DESTROY:
        return this.destroyFunc;
      case TYPES.LOAD:
        return this.loadFunc;
      case TYPES.LOAD_ALL:
        return this.loadAllFunc;
      case TYPES.UPDATE:
        return this.updateFunc;
      default:
        throw new Error(`Unknown api call for type: ${type}`);
    }
  }

  _genericActions(type) {
    if (!type) {
      throw new Error("generic action type is not defined");
    }

    const lowerType = type.toLowerCase();
    const capitalType = capitalize(type);

    return {
      [lowerType]: this._genericThunkAction(type),
      [`silent${capitalType}`]: this._genericThunkAction(type, {
        silent: true,
      }),
      [`${lowerType}Request`]: this._genericRequest(type),
      [`${lowerType}Success`]: this._genericSuccess(type),
      [`${lowerType}Failure`]: this._genericFailure(type),
    };
  }

  _genericThunkAction(type, options = {}) {
    const { TYPES } = BaseConfig;
    const apiCall = this._apiCallForType(type);

    return (...args) => {
      return (dispatch) => {
        if (!options.silent) {
          dispatch(this._genericRequest(type)());
        }

        return apiCall(...args)
          .then((response) => {
            let thunk = this._genericSuccess(type);
            if (type === TYPES.DESTROY) {
              // Destroy is a special case in which the API does not return an
              // object, so we need to generate a thunk that stores the entity
              // ID for removal from the entity store.
              thunk = this._destroySuccess(...args);
            }

            dispatch(this.successAction(response, thunk));

            return response;
          })
          .catch((response) => {
            const thunk = this._genericFailure(type);
            const errorsObject = formatErrorResponse(response);

            dispatch(thunk(errorsObject));

            throw errorsObject;
          });
      };
    };
  }

  _genericRequest(type) {
    const { TYPES } = BaseConfig;

    switch (type) {
      case TYPES.CREATE:
        return () => {
          return { type: this.actionTypes.CREATE_REQUEST };
        };
      case TYPES.DESTROY:
        return () => {
          return { type: this.actionTypes.DESTROY_REQUEST };
        };
      case TYPES.LOAD:
      case TYPES.LOAD_ALL:
        return () => {
          return { type: this.actionTypes.LOAD_REQUEST };
        };
      case TYPES.UPDATE:
        return () => {
          return { type: this.actionTypes.UPDATE_REQUEST };
        };
      default:
        throw new Error(`Unknown request type: ${type}`);
    }
  }

  _genericSuccess(type) {
    const { actionTypes } = this;

    return (data, originalOrder) => {
      return {
        type: BaseConfig.successActionTypeFor(actionTypes, type),
        // This originalOrder is included to keep the order of the results sent back from the API
        // This is used in things like server side ordering for data tables (e.g. host data table)
        payload: { data, originalOrder },
      };
    };
  }

  _destroySuccess(entity) {
    const { actionTypes } = this;

    return () => {
      // No data should be returned from a DELETE API call, and instead the
      // payload should be the ID of the entity deleted.
      return {
        type: actionTypes.DESTROY_SUCCESS,
        payload: { data: entity.id },
      };
    };
  }

  _genericFailure(type) {
    const { actionTypes } = this;

    return (errors) => {
      return {
        type: BaseConfig.failureActionTypeFor(actionTypes, type),
        payload: { errors },
      };
    };
  }

  _parse(response) {
    let result = response;
    const { parseApiResponseFunc, parseEntityFunc } = this;

    if (!parseApiResponseFunc && !parseEntityFunc) {
      return result;
    }

    result = parseApiResponseFunc ? parseApiResponseFunc(response) : response;

    if (!isArray(result) && parseEntityFunc) {
      throw new Error(
        "parseEntityFunc must be called on an array. Use the parseApiResponseFunc to format the response correctly."
      );
    }

    result = parseEntityFunc ? result.map((r) => parseEntityFunc(r)) : result;

    return result;
  }

  _clearErrors() {
    return { type: this.actionTypes.CLEAR_ERRORS };
  }
}

export default BaseConfig;
