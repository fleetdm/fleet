import React from "react";

const baseClass = "manage-software-page";

type IEmptySoftware = "search" | "vulnerable" | "default" | "";

const EmptySoftware = (message: IEmptySoftware): JSX.Element => {
  switch (message) {
    case "search":
      return (
        <div className={`${baseClass}__empty-software`}>
          <h1>No software matches the current search criteria.</h1>
          <p>
            Expecting to see software? Try again in a few seconds as the system
            catches up.
          </p>
        </div>
      );
    case "vulnerable":
    default:
      return (
        <div className={`${baseClass}__empty-software`}>
          <h1>
            No installed software{" "}
            {message === "vulnerable"
              ? "with detected vulnerabilities"
              : "detected"}
            .
          </h1>
          <p>
            Expecting to see software? Check out the Fleet documentation on{" "}
            <a
              href="https://fleetdm.com/docs/deploying/configuration#software-inventory"
              target="_blank"
              rel="noopener noreferrer"
            >
              how to configure software inventory
            </a>
            .
          </p>
        </div>
      );
  }
};

export default EmptySoftware;
