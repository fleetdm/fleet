/**
  A collection of utilities to enable easier writting of tests
 */

import { render } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

// eslint-disable-next-line import/prefer-default-export
export const renderWithSetup = (component: JSX.Element) => {
  return {
    user: userEvent.setup(),
    ...render(component),
  };
};
