import { mount } from "enzyme";

import ConfirmInvitePage from "pages/ConfirmInvitePage";
import { connectedComponent, reduxMockStore } from "test/helpers";

describe("ConfirmInvitePage - component", () => {
  const inviteToken = "abc123";
  const location = { query: { email: "hi@gnar.dog", name: "Gnar Dog" } };
  const params = { invite_token: inviteToken };
  const mockStore = reduxMockStore({ auth: {}, entities: { users: {} } });
  const component = connectedComponent(ConfirmInvitePage, {
    props: { location, params },
    mockStore,
  });
  const page = mount(component);

  it("renders", () => {
    expect(page.length).toEqual(1);
    expect(page.find("ConfirmInvitePage").prop("inviteFormData")).toEqual({
      email: "hi@gnar.dog",
      invite_token: inviteToken,
      name: "Gnar Dog",
    });
  });

  it("renders a ConfirmInviteForm", () => {
    expect(page.find("ConfirmInviteForm").length).toEqual(1);
  });

  it("clears errors on unmount", () => {
    page.unmount();

    expect(mockStore.getActions()).toContainEqual({
      type: "users_CLEAR_ERRORS",
    });
  });
});
