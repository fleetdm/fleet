import helpers from "pages/queries/QueryPage/helpers";
import { initialState } from "redux/nodes/components/QueryPages/reducer";
import Test from "test";

describe("QueryPage - helpers", () => {
  describe("#selectHosts", () => {
    const createMockStore = () => {
      return Test.Helpers.reduxMockStore({
        components: {
          QueryPages: initialState,
        },
      });
    };

    describe("when there are selected targets and no selected hosts", () => {
      it("does not dispatch an action to set targets", () => {
        const mockStore = createMockStore();
        const selectedTargets = [Test.Stubs.labelStub];

        helpers.selectHosts(mockStore.dispatch, {
          hosts: [],
          selectedTargets,
        });

        expect(mockStore.getActions()).toEqual([]);
      });
    });

    describe("when there are selected hosts and no selected targets", () => {
      it("sets the selected targets to the selected hosts", () => {
        const mockStore = createMockStore();
        const selectedHosts = [Test.Stubs.hostStub];

        helpers.selectHosts(mockStore.dispatch, {
          hosts: selectedHosts,
          selectedTargets: [],
        });

        expect(mockStore.getActions()).toContainEqual({
          type: "SET_SELECTED_TARGETS",
          payload: {
            selectedTargets: selectedHosts,
          },
        });
      });
    });

    describe("when there are selected hosts and selected targets", () => {
      it("sets the selected targets to the combined selected hosts and selected targets", () => {
        const mockStore = createMockStore();

        helpers.selectHosts(mockStore.dispatch, {
          hosts: [Test.Stubs.hostStub],
          selectedTargets: [Test.Stubs.labelStub],
        });

        expect(mockStore.getActions()).toContainEqual({
          type: "SET_SELECTED_TARGETS",
          payload: {
            selectedTargets: [Test.Stubs.hostStub, Test.Stubs.labelStub],
          },
        });
      });
    });

    describe("when a target is duplicated", () => {
      it("does not duplicate the target when setting selected targets", () => {
        const mockStore = createMockStore();

        helpers.selectHosts(mockStore.dispatch, {
          hosts: [Test.Stubs.hostStub],
          selectedTargets: [Test.Stubs.labelStub, Test.Stubs.hostStub],
        });

        expect(mockStore.getActions()).toContainEqual({
          type: "SET_SELECTED_TARGETS",
          payload: {
            selectedTargets: [Test.Stubs.hostStub, Test.Stubs.labelStub],
          },
        });
      });

      it("does not set targets if the hosts and selectedTargets are equal", () => {
        const mockStore = createMockStore();

        helpers.selectHosts(mockStore.dispatch, {
          hosts: [Test.Stubs.hostStub],
          selectedTargets: [Test.Stubs.hostStub],
        });

        expect(mockStore.getActions()).toEqual([]);
      });
    });
  });
});
