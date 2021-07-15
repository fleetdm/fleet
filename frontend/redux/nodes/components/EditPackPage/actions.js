// import Fleet from "fleet";
// import { getPackedSettings } from "node:http2";
// import { formatErrorResponse } from "redux/nodes/entities/base/helpers";
import queryActions from "redux/nodes/entities/queries/actions";

// Actions
export const getQueries = (
  page,
  perPage,
  selectedLabel,
  globalFilter,
  sortBy
) => (dispatch) => {
  dispatch(
    queryActions.loadAll(page, perPage, selectedLabel, globalFilter, sortBy)
  );
};

// This was getHosts
// export const getHosts = (
//   page,
//   perPage,
//   selectedLabel,
//   globalFilter,
//   sortBy
// ) => (dispatch) => {
//   dispatch(
//     hostActions.loadAll(page, perPage, selectedLabel, globalFilter, sortBy)
//   );
// };

export default {
  // getStatusLabelCounts,
  // silentGetStatusLabelCounts,
  // getHostTableData: getHosts,
  getPackTableData: getQueries,
};
