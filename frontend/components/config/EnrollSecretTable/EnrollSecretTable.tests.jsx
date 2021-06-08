import React from "react";
import { shallow, mount } from "enzyme";

import * as copy from "utilities/copy_text";
import EnrollSecretTable, {
  EnrollSecretRow,
} from "components/config/EnrollSecretTable/EnrollSecretTable";

describe("EnrollSecretTable", () => {
  const defaultProps = {
    secrets: [
      { secret: "foo_secret" },
      { secret: "bar_secret" },
      { secret: "baz_secret" },
    ],
  };

  it("renders properly filtered rows", () => {
    const table = shallow(<EnrollSecretTable {...defaultProps} />);
    expect(table.find("EnrollSecretRow").length).toEqual(3);
  });

  it("renders text when empty", () => {
    const table = shallow(<EnrollSecretTable secrets={[]} />);
    expect(table.find("EnrollSecretRow").length).toEqual(0);
    expect(table.find("div").text()).toEqual("No active enroll secrets.");
  });
});

describe("EnrollSecretRow", () => {
  const defaultProps = { name: "foo", secret: "bar" };
  it("should hide secret by default", () => {
    const row = mount(<EnrollSecretRow {...defaultProps} />);
    const inputField = row.find("InputField").find("input");
    expect(inputField.prop("type")).toEqual("password");
  });

  it("should show secret when enabled", () => {
    const row = mount(<EnrollSecretRow {...defaultProps} />);
    row.setState({ showSecret: true });
    const inputField = row.find("InputField").find("input");
    expect(inputField.prop("type")).toEqual("text");
  });

  it("should change input type when show/hide is clicked", () => {
    const row = mount(<EnrollSecretRow {...defaultProps} />);

    let inputField = row.find("InputField").find("input");
    expect(inputField.prop("type")).toEqual("password");

    const showLink = row.find(".enroll-secrets__show-secret");
    expect(showLink.find("img").prop("alt")).toEqual("show/hide");

    showLink.simulate("click");

    inputField = row.find("InputField").find("input");
    expect(inputField.prop("type")).toEqual("text");

    const hideLink = row.find(".enroll-secrets__show-secret");
    expect(showLink.find("img").prop("alt")).toEqual("show/hide");

    hideLink.simulate("click");

    inputField = row.find("InputField").find("input");
    expect(inputField.prop("type")).toEqual("password");
  });

  it("should call copy when button is clicked", () => {
    const row = mount(<EnrollSecretRow {...defaultProps} />);
    const spy = jest
      .spyOn(copy, "stringToClipboard")
      .mockImplementation(() => Promise.resolve());

    const copyLink = row
      .find(".enroll-secrets__secret-copy-icon")
      .find("Button");
    copyLink.simulate("click");

    expect(spy).toHaveBeenCalledWith(defaultProps.secret);
  });
});
