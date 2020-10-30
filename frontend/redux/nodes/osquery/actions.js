import Kolide from 'kolide';

import formatApiErrors from 'utilities/format_api_errors';
// import { frontendFormattedOsqueryOptions } from 'redux/nodes/osquery/helpers';

export const OSQUERY_OPTIONS_FAILURE = 'OSQUERY_OPTIONS_FAILURE';
export const OSQUERY_OPTIONS_START = 'OSQUERY_OPTIONS_START';
export const OSQUERY_OPTIONS_SUCCESS = 'OSQUERY_OPTIONS_SUCCESS';

export const loadOsqueryOptions = { type: OSQUERY_OPTIONS_START };

export const osqueryOptionsSuccess = (data) => {
  return { type: OSQUERY_OPTIONS_SUCCESS, payload: { data }};
}

export const osqueryOptionsFailure = (errors) => {
  return { type: OSQUERY_OPTIONS_FAILURE, payload: { errors }};
}

export const getOsqueryOptions = () => {
  return(dispatch) => {
    dispatch(loadOsqueryOptions);

    return Kolide.osqueryOptions.loadAll()
      .then((osqueryOptions) => {
        // const formattedOsqueryOptions = frontendFormattedOsqueryOptions(osqueryOptions);

        // dispatch(osqueryOptionsSuccess(formattedOsqueryOptions));

        dispatch(osqueryOptionsSuccess(osqueryOptions));

        return osqueryOptions;
      })
      .catch((errors) => {
        console.log(errors)
        dispatch(osqueryOptionsFailure(errors));

        throw errors;
      });
  }
}
