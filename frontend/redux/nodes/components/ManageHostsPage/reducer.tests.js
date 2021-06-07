import Fleet from "fleet";
import { reduxMockStore } from "test/helpers";
import {
  getStatusLabelCounts,
  getStatusLabelCountsFailure,
  getStatusLabelCountsSuccess,
  loadStatusLabelCounts,
  silentGetStatusLabelCounts,
} from "./actions";
import reducer, { initialState } from "./reducer";

describe("ManageHostsPage - reducer", () => {
  it("sets the initial state", () => {
    expect(reducer(undefined, { type: "SOME_ACTION" })).toEqual(initialState);
  });

  describe("#getStatusLabelCounts", () => {
    it("sets the loading boolean", () => {
      expect(reducer(initialState, loadStatusLabelCounts)).toEqual({
        ...initialState,
        status_labels: {
          ...initialState.status_labels,
          loading_counts: true,
        },
      });
    });

    it("dispatches the correct actions when successful", (done) => {
      const statusLabelCounts = {
        online_count: 23,
        offline_count: 100,
        mia_count: 2,
      };
      const store = { components: { ManageHostsPage: initialState } };
      const mockStore = reduxMockStore(store);
      const expectedActions = [
        { type: "LOAD_STATUS_LABEL_COUNTS" },
        {
          type: "GET_STATUS_LABEL_COUNTS_SUCCESS",
          payload: { status_labels: statusLabelCounts },
        },
      ];

      jest
        .spyOn(Fleet.statusLabels, "getCounts")
        .mockImplementation(() => Promise.resolve(statusLabelCounts));

      mockStore.dispatch(getStatusLabelCounts).then(() => {
        expect(mockStore.getActions()).toEqual(expectedActions);

        done();
      });
    });

    it("dispatches the correct actions when unsuccessful", (done) => {
      const store = { components: { ManageHostsPage: initialState } };
      const mockStore = reduxMockStore(store);
      const errors = [{ name: "error_name", reason: "error reason" }];
      const errorObject = { status: 422, message: { message: "oops", errors } };
      const expectedActions = [
        { type: "LOAD_STATUS_LABEL_COUNTS" },
        {
          type: "GET_STATUS_LABEL_COUNTS_FAILURE",
          payload: { errors: { error_name: "error reason", http_status: 422 } },
        },
      ];

      jest
        .spyOn(Fleet.statusLabels, "getCounts")
        .mockImplementation(() => Promise.reject(errorObject));

      mockStore
        .dispatch(getStatusLabelCounts)
        .then(() => {
          throw new Error("Promise should have failed");
        })
        .catch(() => {
          expect(mockStore.getActions()).toEqual(expectedActions);

          done();
        });
    });

    it("adds the label counts to state when successful", () => {
      const statusLabelCounts = {
        online_count: 23,
        offline_count: 100,
        mia_count: 2,
      };
      const successAction = getStatusLabelCountsSuccess(statusLabelCounts);

      expect(reducer(initialState, successAction)).toEqual({
        ...initialState,
        status_labels: {
          ...statusLabelCounts,
          errors: {},
          loading_counts: false,
        },
      });
    });

    it("adds errors to state when unsuccessful", () => {
      const errors = { error_name: "error reason" };
      const failureAction = getStatusLabelCountsFailure(errors);

      expect(reducer(initialState, failureAction)).toEqual({
        ...initialState,
        status_labels: {
          ...initialState.status_labels,
          errors,
          loading_counts: false,
        },
      });
    });
  });

  describe("#silentGetStatusLabelCounts", () => {
    it("dispatches the correct actions when successful", (done) => {
      const statusLabelCounts = {
        online_count: 23,
        offline_count: 100,
        mia_count: 2,
      };
      const store = { components: { ManageHostsPage: initialState } };
      const mockStore = reduxMockStore(store);
      const expectedActions = [
        {
          type: "GET_STATUS_LABEL_COUNTS_SUCCESS",
          payload: { status_labels: statusLabelCounts },
        },
      ];

      jest
        .spyOn(Fleet.statusLabels, "getCounts")
        .mockImplementation(() => Promise.resolve(statusLabelCounts));

      mockStore.dispatch(silentGetStatusLabelCounts).then(() => {
        expect(mockStore.getActions()).toEqual(expectedActions);

        done();
      });
    });

    it("dispatches the correct actions when unsuccessful", (done) => {
      const store = { components: { ManageHostsPage: initialState } };
      const mockStore = reduxMockStore(store);
      const errors = [{ name: "error_name", reason: "error reason" }];
      const errorObject = { status: 422, message: { message: "oops", errors } };
      const expectedActions = [
        {
          type: "GET_STATUS_LABEL_COUNTS_FAILURE",
          payload: { errors: { error_name: "error reason", http_status: 422 } },
        },
      ];

      jest
        .spyOn(Fleet.statusLabels, "getCounts")
        .mockImplementation(() => Promise.reject(errorObject));

      mockStore
        .dispatch(silentGetStatusLabelCounts)
        .then(() => {
          throw new Error("Promise should have failed");
        })
        .catch(() => {
          expect(mockStore.getActions()).toEqual(expectedActions);

          done();
        });
    });
  });
});
