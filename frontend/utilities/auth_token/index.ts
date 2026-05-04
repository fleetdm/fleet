/**
 * This contains a collection of utility functions for working with
 * users auth token.
 */
import Cookie from "js-cookie";

const DEFAULT_EXPIRATION_DAYS = 5;

const save = (token: string, expiresAt?: Date): void => {
  Cookie.set("__Host-token", token, {
    secure: true,
    sameSite: "lax",
    expires: expiresAt ?? DEFAULT_EXPIRATION_DAYS,
  });
};

const get = (): string | null => {
  return Cookie.get("__Host-token") || null;
};

const remove = (): void => {
  // NOTE: the secure and sameSite from the cookie must be provided
  // to correctly remove. That is why we include the options here as well.
  Cookie.remove("__Host-token", {
    secure: true,
    sameSite: "lax",
  });
};

export default {
  save,
  get,
  remove,
};
