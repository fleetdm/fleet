import { userTeamStub } from "test/stubs";
import createMockUser from "__mocks__/userMock";
import { IUserUpdateBody } from "interfaces/user";

import { IUserFormData, NewUserType } from "../components/UserForm/UserForm";
import userManagementHelpers from "./userManagementHelpers";

describe("userManagementHelpers module", () => {
  describe("generateUpdatedData function", () => {
    it("returns an object with only the difference between the two", () => {
      const updatedTeam: IUserUpdateBody = {
        ...userTeamStub,
        role: "maintainer",
      };
      const newTeam: IUserUpdateBody = {
        ...userTeamStub,
        id: 2,
        role: "observer",
      };

      const formData: IUserFormData = {
        email: "newemail@test.com",
        sso_enabled: false,
        name: "Test User",
        newUserType: NewUserType.AdminCreated, // TODO revisit test
        global_role: "admin",
        teams: [updatedTeam, newTeam],
      };
      const updatedData = userManagementHelpers.generateUpdateData(
        createMockUser({ role: "Observer", global_role: null }),
        formData
      );

      expect(updatedData).toEqual({
        email: "newemail@test.com",
        global_role: "admin",
        teams: [updatedTeam, newTeam],
      });
    });
  });
});
