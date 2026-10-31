import React from "react";

// eslint-disable-next-line import/prefer-default-export
export const renderAppleManualEnrollmentDisabled = (
  type: string,
  isGlobalAdmin?: boolean
) => {
  return (
    <p>
      Only {type} hosts in Apple Business can enroll.{" "}
      {isGlobalAdmin ? (
        <>
          To allow manual enrollment, turn off the &ldquo;Only allow Apple
          Business enrollment&rdquo; setting in{" "}
          <b>Organization settings &gt; Advanced options</b>.
        </>
      ) : (
        "Contact your administrator to allow manual enrollment."
      )}
    </p>
  );
};
