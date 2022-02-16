import React from "react";

const SoftwarePageWrapper = ({
  children,
}: {
  children: React.ReactNode;
}): React.ReactNode | null => {
  return children || null;
};

export default SoftwarePageWrapper;
