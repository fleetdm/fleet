import helpers from "test/helpers";
import stubs from "test/stubs";
import mocks from "test/mocks";
import targetMock from "test/target_mock";

export default {
  Helpers: helpers,
  Mocks: { ...mocks, targetMock },
  Stubs: stubs,
};
