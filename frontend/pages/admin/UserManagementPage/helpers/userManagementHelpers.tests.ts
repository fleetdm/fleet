import { userStub, userTeamStub } from "test/stubs";
import { IUserUpdateBody } from "interfaces/user";

import { IFormData, NewUserType } from "../components/UserForm/UserForm";
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

      const formData: IFormData = {
        email: "newemail@test.com",
        sso_enabled: false,
        name: "Gnar Mike",
        newUserType: NewUserType.AdminCreated, // TODO revisit test
        global_role: "admin",
        teams: [updatedTeam, newTeam],
      };
      const updatedData = userManagementHelpers.generateUpdateData(
        userStub,
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
