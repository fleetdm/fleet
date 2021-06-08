import {
  entitiesExceptID,
  orderExceptId,
  formatErrorResponse,
} from "./helpers";

describe("reduxConfig - helpers", () => {
  describe("#entitiesExceptID", () => {
    it("returns an empty object if all ids are deleted", () => {
      const entities = {
        1: { name: "Gnar" },
      };
      const id = 1;

      expect(entitiesExceptID(entities, id)).toEqual({});
    });

    it("removes the object with the key of the specified id", () => {
      const entities = {
        1: { name: "Gnar" },
        2: { name: "Dog" },
      };
      const id = 1;

      expect(entitiesExceptID(entities, id)).toEqual({
        2: { name: "Dog" },
      });
    });
  });

  describe("#orderExceptId", () => {
    it("returns a new orderId array with the deleted entityID no longer in this array", () => {
      const originalOrder = [1, 2, 3, 4];
      expect(orderExceptId(originalOrder, 3)).toEqual([1, 2, 4]);
    });
  });

  describe("#formatErrorResponse", () => {
    it("converts the error response to an object for redux state", () => {
      const errors = [
        { name: "first_name", reason: "is not valid" },
        { name: "first_name", reason: "must be something else" },
        { name: "last_name", reason: "must be changed or something" },
      ];
      const errorResponse = {
        status: 422,
        message: {
          message: "Validation Failed",
          errors,
        },
      };

      expect(formatErrorResponse(errorResponse)).toEqual({
        first_name: "is not valid, must be something else",
        http_status: 422,
        last_name: "must be changed or something",
      });
    });
  });
});
