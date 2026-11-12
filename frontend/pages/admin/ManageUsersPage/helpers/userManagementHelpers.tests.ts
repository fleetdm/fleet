import createMockUser from "__mocks__/userMock";
import { IUserUpdateFormData } from "interfaces/user";
import { userTeamStub } from "test/stubs";

import { IUserFormData, NewUserType } from "../components/UserForm/UserForm";

import userManagementHelpers, { roleOptions } from "./userManagementHelpers";

describe("userManagementHelpers module", () => {
  describe("generateUpdatedData function", () => {
    it("returns an object with only the difference between the two", () => {
      const updatedTeam: IUserUpdateFormData = {
        ...userTeamStub,
        role: "maintainer",
      };
      const newTeam: IUserUpdateFormData = {
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
        fleets: [updatedTeam, newTeam],
      };
      const updatedData = userManagementHelpers.generateUpdateData(
        createMockUser({ role: "Observer", global_role: null }),
        formData
      );

      expect(updatedData).toEqual({
        email: "newemail@test.com",
        global_role: "admin",
        fleets: [updatedTeam, newTeam],
      });
    });
  });

  describe("roleOptions function", () => {
    const values = (params: Parameters<typeof roleOptions>[0]) =>
      roleOptions(params).map((o) => o.value);

    it("includes GitOps for API-only users on free tier", () => {
      expect(values({ isPremiumTier: false, isApiOnly: true })).toEqual([
        "observer",
        "gitops",
        "maintainer",
        "admin",
      ]);
    });

    it("includes GitOps for API-only users on premium tier", () => {
      expect(values({ isPremiumTier: true, isApiOnly: true })).toEqual([
        "observer",
        "observer_plus",
        "technician",
        "gitops",
        "maintainer",
        "admin",
      ]);
    });

    it("excludes GitOps for non-API-only users", () => {
      expect(values({ isPremiumTier: false })).toEqual([
        "observer",
        "maintainer",
        "admin",
      ]);
      expect(values({ isPremiumTier: true })).toEqual([
        "observer",
        "observer_plus",
        "technician",
        "maintainer",
        "admin",
      ]);
    });
  });
});
