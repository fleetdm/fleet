import nock from "nock";

import Fleet from "fleet";
import mocks from "test/mocks";
import { userStub } from "test/stubs";

const { users: userMocks } = mocks;

describe("Kolide - API client (users)", () => {
  afterEach(() => {
    nock.cleanAll();
    Fleet.setBearerToken(null);
  });

  const bearerToken = "valid-bearer-token";

  describe("#changePassword", () => {
    it("calls the appropriate endpoint with the correct parameters", () => {
      const passwordParams = {
        old_password: "password",
        new_password: "p@ssw0rd",
      };
      const request = userMocks.changePassword.valid(
        bearerToken,
        passwordParams
      );

      Fleet.setBearerToken(bearerToken);
      return Fleet.users.changePassword(passwordParams).then(() => {
        expect(request.isDone()).toEqual(true);
      });
    });
  });

  describe("#confirmEmailChange", () => {
    it("calls the appropriate endpoint with the correct parameters", () => {
      const token = "KFBR392";
      const request = userMocks.confirmEmailChange.valid(bearerToken, token);

      Fleet.setBearerToken(bearerToken);
      return Fleet.users.confirmEmailChange(userStub, token).then(() => {
        expect(request.isDone()).toEqual(true);
      });
    });
  });

  describe("#enable", () => {
    it("calls the appropriate endpoint with the correct parameters", () => {
      const enableParams = { enabled: true };
      const request = userMocks.enable.valid(
        bearerToken,
        userStub,
        enableParams
      );

      Fleet.setBearerToken(bearerToken);
      return Fleet.users.enable(userStub, enableParams).then(() => {
        expect(request.isDone()).toEqual(true);
      });
    });
  });

  describe("#loadAll", () => {
    it("calls the appropriate endpoint with the correct parameters", () => {
      const request = userMocks.loadAll.valid(bearerToken);

      Fleet.setBearerToken(bearerToken);
      return Fleet.users.loadAll({}).then(() => {
        expect(request.isDone()).toEqual(true);
      });
    });

    it("calls the appropriate endpoint with the correct query params when passed multiple arguments", () => {
      const request = userMocks.loadAll.validWithParams(bearerToken);
      const page = 3;
      const perPage = 100;
      const globalFilter = "testQuery";
      const sortBy = [{ id: "name", desc: true }];

      Fleet.setBearerToken(bearerToken);
      return Fleet.users
        .loadAll({ page, perPage, globalFilter, sortBy })
        .then(() => {
          expect(request.isDone()).toEqual(true);
        });
    });
  });

  describe("#me", () => {
    it("calls the appropriate endpoint with the correct parameters", () => {
      const request = userMocks.me.valid(bearerToken);

      Fleet.setBearerToken(bearerToken);
      return Fleet.users.me().then(() => {
        expect(request.isDone()).toEqual(true);
      });
    });
  });

  describe("#forgotPassword", () => {
    it("calls the appropriate endpoint with the correct parameters when successful", () => {
      const request = userMocks.forgotPassword.valid();
      const email = "hi@thegnar.co";

      return Fleet.users.forgotPassword({ email }).then(() => {
        expect(request.isDone()).toEqual(true);
      });
    });

    it("return errors correctly for unsuccessful requests", () => {
      const error = { base: "Something went wrong" };
      const errorResponse = {
        message: {
          errors: [error],
        },
      };
      const request = userMocks.forgotPassword.invalid(errorResponse);
      const email = "hi@thegnar.co";

      return Fleet.users.forgotPassword({ email }).catch(() => {
        expect(request.isDone()).toEqual(true);
      });
    });
  });

  describe("#resetPassword", () => {
    const newPassword = "p@ssw0rd";

    it("calls the appropriate endpoint with the correct parameters when successful", () => {
      const passwordResetToken = "password-reset-token";
      const request = userMocks.resetPassword.valid(
        newPassword,
        passwordResetToken
      );
      const formData = {
        new_password: newPassword,
        password_reset_token: passwordResetToken,
      };

      return Fleet.users.resetPassword(formData).then(() => {
        expect(request.isDone()).toEqual(true);
      });
    });

    it("return errors correctly for unsuccessful requests", () => {
      const errorResponse = {
        message: "Resource not found",
        errors: [{ name: "base", reason: "Resource not found" }],
      };
      const passwordResetToken = "invalid-password-reset-token";
      const request = userMocks.resetPassword.invalid(
        newPassword,
        passwordResetToken,
        errorResponse
      );
      const formData = {
        new_password: newPassword,
        password_reset_token: passwordResetToken,
      };

      return Fleet.users.resetPassword(formData).catch(() => {
        expect(request.isDone()).toEqual(true);
      });
    });
  });

  describe("#update", () => {
    it("calls the appropriate endpoint with the correct parameters", () => {
      const formData = { enabled: false };
      const request = userMocks.update.valid(userStub, formData);

      return Fleet.users.update(userStub, formData).then(() => {
        expect(request.isDone()).toEqual(true);
      });
    });
  });

  describe("#updateAdmin", () => {
    it("calls the appropriate endpoint with the correct parameters", () => {
      const adminParams = { admin: true };
      const request = userMocks.updateAdmin.valid(
        bearerToken,
        userStub,
        adminParams
      );

      Fleet.setBearerToken(bearerToken);
      return Fleet.users.updateAdmin(userStub, adminParams).then(() => {
        expect(request.isDone()).toEqual(true);
      });
    });
  });
});
