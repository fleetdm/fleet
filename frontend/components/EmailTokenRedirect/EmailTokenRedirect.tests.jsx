import React from "react";
import { mount } from "enzyme";

import { connectedComponent, reduxMockStore } from "test/helpers";
import ConnectedEmailTokenRedirect, {
  EmailTokenRedirect,
} from "components/EmailTokenRedirect/EmailTokenRedirect";
import Fleet from "fleet";
import { userStub } from "test/stubs";

describe("EmailTokenRedirect - component", () => {
  beforeEach(() => {
    jest
      .spyOn(Fleet.users, "confirmEmailChange")
      .mockImplementation(() =>
        Promise.resolve({ ...userStub, email: "new@email.com" })
      );
  });

  const authStore = {
    auth: {
      user: userStub,
    },
  };
  const token = "KFBR392";
  const defaultProps = {
    params: {
      token,
    },
  };

  describe("componentWillMount", () => {
    it("calls the API when a token and user are present", () => {
      const mockStore = reduxMockStore(authStore);

      mount(
        connectedComponent(ConnectedEmailTokenRedirect, {
          mockStore,
          props: defaultProps,
        })
      );

      expect(Fleet.users.confirmEmailChange).toHaveBeenCalledWith(
        userStub,
        token
      );
    });

    it("does not call the API when only a token is present", () => {
      const mockStore = reduxMockStore({ auth: {} });

      mount(
        connectedComponent(ConnectedEmailTokenRedirect, {
          mockStore,
          props: defaultProps,
        })
      );

      expect(Fleet.users.confirmEmailChange).not.toHaveBeenCalled();
    });
  });

  describe("componentWillReceiveProps", () => {
    it("calls the API when a user is received", () => {
      const mockStore = reduxMockStore();
      const props = { dispatch: mockStore.dispatch, token };
      const Component = mount(<EmailTokenRedirect {...props} />);

      expect(Fleet.users.confirmEmailChange).not.toHaveBeenCalled();

      Component.setProps({ user: userStub });

      expect(Fleet.users.confirmEmailChange).toHaveBeenCalledWith(
        userStub,
        token
      );
    });
  });
});
