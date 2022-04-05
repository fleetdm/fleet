import { render } from "@testing-library/react";

import ConfirmSSOInvitePage from "pages/ConfirmSSOInvitePage";
import { connectedComponent, reduxMockStore } from "test/helpers";

describe("ConfirmSSOInvitePage - component", () => {
  const inviteToken = "abc123";
  const location = { query: { email: "hi@gnar.dog", name: "Gnar Dog" } };
  const params = { invite_token: inviteToken };
  const mockStore = reduxMockStore({ auth: {}, entities: { users: {} } });
  const component = connectedComponent(ConfirmSSOInvitePage, {
    props: { location, params },
    mockStore,
  });

  it("renders", () => {
    const { container } = render(component);
    expect(container).not.toBeEmptyDOMElement();
  });

  it("renders a ConfirmSSOInviteForm", () => {
    const { container } = render(component);
    expect(
      container.querySelectorAll(".confirm-invite-page__form").length
    ).toEqual(1);
  });

  it("clears errors on unmount", () => {
    const { unmount } = render(component);

    unmount();

    expect(mockStore.getActions()).toContainEqual({
      type: "users_CLEAR_ERRORS",
    });
  });
});
