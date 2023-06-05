import React from "react";

interface ITooltipInnerContentActionRequiredProps {
  isDeviceUser: boolean;
  profileName: string;
}

const TooltipInnerContentActionRequired = ({
  isDeviceUser,
  profileName,
}: ITooltipInnerContentActionRequiredProps) => {
  const instructions = profileName ? (
    <>
      <b>{profileName}</b> instructions
    </>
  ) : (
    <>instructions</>
  );

  if (isDeviceUser) {
    return (
      <>
        Follow the {instructions}
        <br />
        on your <b>My device</b> page.
      </>
    );
  }

  return (
    <>
      Ask the end user to follow the {instructions} on their <b>My device</b>{" "}
      page.
    </>
  );
};

export default TooltipInnerContentActionRequired;
