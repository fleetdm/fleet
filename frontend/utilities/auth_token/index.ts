/**
 * This contains a collection of utility functions for working with
 * users auth token.
 */
import Cookie from "js-cookie";

const save = (token: string): void => {
  Cookie.set("__Host-token", token, { secure: true, sameSite: "lax" });
};

const get = (): string | null => {
  return Cookie.get("__Host-token") || null;
};

const remove = (): void => {
  Cookie.remove("__Host-token");
};

export default {
  save,
  get,
  remove,
};
