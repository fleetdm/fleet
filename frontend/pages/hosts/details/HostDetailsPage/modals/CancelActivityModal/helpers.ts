import { hasStatusKey } from "pages/hosts/ManageHostsPage/helpers";

const DEFAULT_ERR_MESSAGE = "Couldn't cancel activity. Please try again.";
const LOCK_WIPE_ERR_MESSAGE =
  "Couldn't cancel activity. Lock and wipe can't be canceled if they're about to run to prevent you from losing access to the host.";
const ACTIVITY_ALREADY_HAPPENED_ERR_MESSAGE =
  "Couldn't cancel activity. Activity already happened.";

// eslint-disable-next-line import/prefer-default-export
export const getErrorMessage = (err: unknown) => {
  if (hasStatusKey(err)) {
    if (err.status === 404) return ACTIVITY_ALREADY_HAPPENED_ERR_MESSAGE;

    // display server error message if error is 400
    if (err.status === 400) return LOCK_WIPE_ERR_MESSAGE;
  }

  return DEFAULT_ERR_MESSAGE;
};
