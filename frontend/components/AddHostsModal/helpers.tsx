import React from "react";

// eslint-disable-next-line import/prefer-default-export
export const renderAppleManualEnrollmentDisabled = (
  type: string,
  isGlobalAdmin?: boolean
) => {
  return (
    <div className="platform-wrapper__apple-manual-enrollment-disabled">
      <p>Only {type} hosts in Apple Business can enroll.</p>
      <p>
        {isGlobalAdmin ? (
          <>
            To allow manual enrollment, turn off the{" "}
            <b>Allow only Apple Business enrollments</b> setting in{" "}
            <b>Organization settings &gt; Advanced options</b>.
          </>
        ) : (
          "Contact your administrator to allow manual enrollment."
        )}
      </p>
    </div>
  );
};
