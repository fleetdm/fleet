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
  // NOTE: the entire cookie including the name and values must be provided
  // to correctly remove. That is why we include the options here as well.
  Cookie.remove("__Host-token", { secure: true, sameSite: "lax" });
};

export default {
  save,
  get,
  remove,
};
