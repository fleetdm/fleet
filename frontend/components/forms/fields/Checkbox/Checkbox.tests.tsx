import React from "react";
import { fireEvent, render, screen } from "@testing-library/react";

import Checkbox from "./Checkbox";

describe("Checkbox - component", () => {
  it("renders", () => {
    expect(render(<Checkbox />).container).not.toBeNull();
  });

  it('calls the "onChange" handler when changed', async () => {
    const onCheckedComponentChangeSpy = jest.fn();
    const onUncheckedComponentChangeSpy = jest.fn();

    const { rerender } = render(
      <Checkbox name="checkbox" onChange={onCheckedComponentChangeSpy} value />
    );

    fireEvent.click(screen.getByRole("checkbox"));

    expect(onCheckedComponentChangeSpy).toHaveBeenCalledWith(false);

    rerender(
      <Checkbox
        name="checkbox"
        onChange={onUncheckedComponentChangeSpy}
        value={false}
      />
    );

    fireEvent.click(screen.getByRole("checkbox"));

    expect(onUncheckedComponentChangeSpy).toHaveBeenCalledWith(true);
  });
});
