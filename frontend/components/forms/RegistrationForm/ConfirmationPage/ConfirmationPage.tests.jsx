import React from "react";
import { fireEvent, render, screen } from "@testing-library/react";

import ConfirmationPage from "components/forms/RegistrationForm/ConfirmationPage";

describe("ConfirmationPage - form", () => {
  const formData = {
    name: "Test User",
    email: "test@example.com",
    org_name: "Fleet",
    server_url: "http://localhost:8080",
  };
  const handleSubmitSpy = jest.fn();

  it("renders the user information", () => {
    render(
      <ConfirmationPage formData={formData} handleSubmit={handleSubmitSpy} />
    );

    expect(screen.getByText(formData.name)).toBeInTheDocument();
    expect(screen.getByText(formData.email)).toBeInTheDocument();
    expect(screen.getByText(formData.org_name)).toBeInTheDocument();
    expect(screen.getByText(formData.server_url)).toBeInTheDocument();
  });

  it("submits the form", () => {
    render(
      <ConfirmationPage
        formData={formData}
        handleSubmit={handleSubmitSpy}
        currentPage
      />
    );

    fireEvent.click(screen.getByText("Confirm"));

    expect(handleSubmitSpy).toHaveBeenCalled();
  });
});
